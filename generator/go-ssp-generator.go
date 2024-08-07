package generator

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

func GenerateGoSspService(pbuf *parser.Proto, goBaseDir string, pkg string) error {
	pb, err := protoparser.UnorderedInterpret(pbuf)
	if err != nil {
		return err
	}

	code, err := generateGoSspHandler(pb, pkg)
	if err != nil {
		return fmt.Errorf("error generating go code: %v \n%s", err, code)
	}

	filename := fmt.Sprintf("%s/%s/%s.go", goBaseDir, pkg, "ssp-handler_gen")
	err = writeFile(filename, code)
	if err != nil {
		return err
	}
	return nil
}

// generateGoRpcHandler Generates go code to dispatch an incoming message,
// call the corresponding handler function and manage all de-/serialization of parameters and responses
func generateGoSspHandler(pb *unordered.Proto, pkg string) (string, error) {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s + "\n") }

	w("package " + pkg + "\n")
	w("import \"google.golang.org/protobuf/proto\"")
	w(generatorWarning)

	for _, srv := range pb.ProtoBody.Services {
		if !hasServiceOption(srv, "(is_ssp)") {
			continue
		}

		// Write interface
		w(generateGoInterface(srv, srv.ServiceName, false))

		// Write data-struct and constructor
		w("type " + srv.ServiceName + `Impl struct {
		ws WebSocket
		log Logger
		}
		`)

		w("func New" + srv.ServiceName + "(ws WebSocket, log Logger) " + srv.ServiceName + `{
		return &` + srv.ServiceName + `Impl {
			ws: ws,
			log: log,
			}
		}
		`)

		for _, rpc := range srv.ServiceBody.RPCs {
			for _, c := range rpc.Comments {
				w(c.Raw)
			}

			w("func (impl *" + srv.ServiceName + "Impl) " + rpc.RPCName + "(p0 *" + rpc.RPCRequest.MessageType + `) {
				data, err := proto.Marshal(p0)
				if err != nil {
					impl.log.Logf("Error in ` + srv.ServiceName + "." + rpc.RPCName + `: %v", err)
					return
				}
				sendPushMessage(impl.ws, "` + rpc.RPCName + `", impl.log, data)
			}`)
		}
	}

	formattedCode, err := format.Source([]byte(sb.String()))
	if err != nil {
		return sb.String(), err
	}
	return string(formattedCode), nil
}
