package servicebuilder

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

const voidTypeName = "Void"

func GenerateGoService(pbuf *parser.Proto, goServiceFile string) error {
	pb, err := protoparser.UnorderedInterpret(pbuf)
	if err != nil {
		return err
	}
	pkg := "api"

	code, err := generateGoServiceInterface(pb, pkg)
	if err != nil {
		return fmt.Errorf("error generating go code: %v \n%s", err, code)
	}

	filename := fmt.Sprintf("%s/%s.go", goServiceFile, "rpc-service_gen")
	err = writeFile(filename, code)
	if err != nil {
		return err
	}

	code, err = generateGoRpcHandler(pb, pkg)
	if err != nil {
		return fmt.Errorf("error generating go code: %v \n%s", err, code)
	}

	filename = fmt.Sprintf("%s/%s.go", goServiceFile, "rpc-handler_gen")
	err = writeFile(filename, code)
	if err != nil {
		return err
	}
	return nil
}

// generateGoServiceInterface writes the interface definition for the service
func generateGoServiceInterface(pb *unordered.Proto, pkg string) (string, error) {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s) }
	wn := func(s string) { sb.WriteString(s + "\n") }
	wn("package " + pkg)

	for _, srv := range pb.ProtoBody.Services {
		if !hasServiceOption(srv, "(is_rpc)") {
			continue
		}

		wn("type " + srv.ServiceName + " interface {")
		for _, rpc := range srv.ServiceBody.RPCs {

			if len(rpc.Comments) > 0 {
				wn("")
			}
			for _, c := range rpc.Comments {
				wn(c.Raw)
			}
			w(rpc.RPCName + "(")

			// parameter
			if rpc.RPCRequest.MessageType != voidTypeName {
				w("param *" + rpc.RPCRequest.MessageType)
			}
			w(")")

			// response
			if rpc.RPCResponse.MessageType != voidTypeName {
				wn(" (*" + rpc.RPCResponse.MessageType + ", error)")
			} else {
				wn(" error")
			}
		}
		wn("}")
	}

	formattedCode, err := format.Source([]byte(sb.String()))
	if err != nil {
		return sb.String(), err
	}
	return string(formattedCode), nil
}

// generateGoRpcHandler Generates go code to dispatch an incoming message,
// call the corresponding handler function and manage all de-/serialization of parameters and responses
func generateGoRpcHandler(pb *unordered.Proto, pkg string) (string, error) {
	var sb strings.Builder
	wn := func(s string) { sb.WriteString(s + "\n") }

	wn("package " + pkg + "\n")
	wn("import \"google.golang.org/protobuf/proto\"")
	wn(generatorWarning)

	wn(`type WebSocket interface {
		Write(msg []byte) error
		WriteBinary(msg []byte) error
		Set(key string, value interface{})
		Get(key string) (value interface{}, exists bool)
	}
	`)

	wn(`type Logger interface {
		Log(str string)
		Logf(format string, a ...any)
	}
	`)

	wn(`func sendAndReturnError(s WebSocket, responseId []byte, err error) error {
	  errResponse := &ErrorDto{
			Error: err.Error(),
		}
		errData, _ := proto.Marshal(errResponse)
		response := make([]byte, len(responseId) + len(errData))
		copy(response, responseId)
		copy(response[len(responseId):], errData)
		s.WriteBinary(response)
		return err
	}
	`)

	for _, srv := range pb.ProtoBody.Services {
		if !hasServiceOption(srv, "(is_rpc)") {
			continue
		}
		wn("func Handle" + srv.ServiceName + "Request(s WebSocket, handler " + srv.ServiceName + `, log Logger, inData []byte) error {
			// get rpc function name
			var name string
			for i := 0; i < len(inData); i++ {
				if inData[i] == 0 {
					name = string(inData[0:i])
					break
				}
			}
			
			// prepare response and set response id
			responseId := inData[len(name):len(name)+4]
			inData = inData[len(name)+4:]
			var outData []byte

			// dispatch function call
			switch name {`)

		for _, rpc := range srv.ServiceBody.RPCs {
			wn("	case \"" + rpc.RPCName + "\":")
			wn("	log.Log(\"Request: '" + rpc.RPCName + "'\")")

			// De-Serialize input parameter
			param := ""
			if rpc.RPCRequest.MessageType != voidTypeName {
				wn("	prm := &" + rpc.RPCRequest.MessageType + "{}")
				wn(`	if err := proto.Unmarshal(inData, prm); err != nil {
					return sendAndReturnError(s, responseId, err)
				}`)
				param = "prm"
			}

			// Invoke handler
			if rpc.RPCResponse.MessageType != voidTypeName {
				wn(fmt.Sprintf("	resp, err := handler.%s(%s)", rpc.RPCName, param))
			} else {
				wn(fmt.Sprintf("	err := handler.%s(%s)", rpc.RPCName, param))
			}
			wn(`	if err != nil {
			return sendAndReturnError(s, responseId, err)
			}`)

			// Serialize result data
			if rpc.RPCResponse.MessageType != voidTypeName {
				wn(`	outData, err = proto.Marshal(resp)
				if err != nil {
				return sendAndReturnError(s, responseId, err)
				}`)
			}

			wn("")
		}

		wn(`	default:
					log.Log("Invalid rpc call: \"" + name + "\"")
				}`)

		// Send response
		wn(`if len(outData) == 0 {
			outData, _ = proto.Marshal(&` + voidTypeName + `{})
		}
		response := make([]byte, len(responseId) + len(outData))
		copy(response, responseId)
		copy(response[len(responseId):], outData)
		s.WriteBinary(response)
		`)

		// No error
		wn(`	return nil
		}
		`)
	}

	formattedCode, err := format.Source([]byte(sb.String()))
	if err != nil {
		return sb.String(), err
	}
	return string(formattedCode), nil
}
