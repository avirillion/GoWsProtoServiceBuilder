package generator

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

func GenerateGoRpcService(pbuf *parser.Proto, goBaseDir string, pkg string) error {
	pb, err := protoparser.UnorderedInterpret(pbuf)
	if err != nil {
		return err
	}

	code, err := generateGoRpcInterface(pb, pkg)
	if err != nil {
		return fmt.Errorf("error generating go code: %v \n%s", err, code)
	}

	filename := fmt.Sprintf("%s/%s/%s.go", goBaseDir, pkg, "rpc-service_gen")
	err = writeFile(filename, code)
	if err != nil {
		return err
	}

	code, err = generateGoRpcHandler(pb, pkg)
	if err != nil {
		return fmt.Errorf("error generating go code: %v \n%s", err, code)
	}

	filename = fmt.Sprintf("%s/%s/%s.go", goBaseDir, pkg, "rpc-handler_gen")
	err = writeFile(filename, code)
	if err != nil {
		return err
	}
	return nil
}

// generateGoRpcInterface writes the interface definition for the service
func generateGoRpcInterface(pb *unordered.Proto, pkg string) (string, error) {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s + "\n") }
	w("package " + pkg)

	for _, srv := range pb.ProtoBody.Services {
		if !hasServiceOption(srv, "(is_rpc)") {
			continue
		}

		w(generateGoInterface(srv, srv.ServiceName, true))
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
	w := func(s string) { sb.WriteString(s + "\n") }

	w("package " + pkg + "\n")
	w("import \"google.golang.org/protobuf/proto\"")
	w(generatorWarning)

	w(`func sendAndReturnError(s WebSocket, requestId int, err error) error {
		errResponse := &` + errorTypeName + `{
			Error: err.Error(),
		}
		errData, _ := proto.Marshal(errResponse)
		responseId := intToByteArray(-requestId)
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
		w("func Handle" + srv.ServiceName + "Request(s WebSocket, handler " + srv.ServiceName + `, log Logger, inData []byte) error {
			// get rpc function name
			var name string
			for i := 0; i < len(inData); i++ {
				if inData[i] == 0 {
					name = string(inData[0:i])
					break
				}
			}
			
			// get request id
			requestId := byteArrayToInt(inData[len(name):len(name)+4])
			inData = inData[len(name)+4:]
			var outData []byte

			// dispatch function call
			switch name {`)

		for _, rpc := range srv.ServiceBody.RPCs {
			w("	case \"" + rpc.RPCName + "\":")
			w("	log.Log(\"Request: '" + rpc.RPCName + "'\")")

			// De-Serialize input parameter
			param := ""
			if rpc.RPCRequest.MessageType != voidTypeName {
				w("	prm := &" + rpc.RPCRequest.MessageType + "{}")
				w(`	if err := proto.Unmarshal(inData, prm); err != nil {
					return sendAndReturnError(s, requestId, err)
				}`)
				param = "prm"
			}

			// Invoke handler
			if rpc.RPCResponse.MessageType != voidTypeName {
				w(fmt.Sprintf("	resp, err := handler.%s(%s)", rpc.RPCName, param))
			} else {
				w(fmt.Sprintf("	err := handler.%s(%s)", rpc.RPCName, param))
			}
			w(`	if err != nil {
			return sendAndReturnError(s, requestId, err)
			}`)

			// Serialize result data
			if rpc.RPCResponse.MessageType != voidTypeName {
				w(`	outData, err = proto.Marshal(resp)
				if err != nil {
				return sendAndReturnError(s, requestId, err)
				}`)
			}

			w("")
		}

		w(`	default:
					log.Log("Invalid rpc call: \"" + name + "\"")
				}`)

		// Send response
		w(`if len(outData) == 0 {
			outData, _ = proto.Marshal(&` + voidTypeName + `{})
		}
		responseId := intToByteArray(requestId);
		response := make([]byte, len(responseId) + len(outData))
		copy(response, responseId)
		copy(response[len(responseId):], outData)
		s.WriteBinary(response)
		`)

		// No error
		w(`	return nil
		}
		`)
	}

	formattedCode, err := format.Source([]byte(sb.String()))
	if err != nil {
		return sb.String(), err
	}
	return string(formattedCode), nil
}
