package generator

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
)

func GenerateGoCommon(goBaseDir string, pkg string) error {
	code, err := generateGoCommonCode(pkg)
	if err != nil {
		return fmt.Errorf("error generating go code: %v \n%s", err, code)
	}

	filename := fmt.Sprintf("%s/%s/%s.go", goBaseDir, pkg, "common_gen")
	err = writeFile(filename, code)
	if err != nil {
		return err
	}

	return nil
}

// generateGoCommonCode creates types for all generated services
func generateGoCommonCode(pkg string) (string, error) {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s + "\n") }

	w("package " + pkg + "\n")

	w(`import (
		"bytes"
		"encoding/binary"
	)
	`)

	w(generatorWarning)

	w(`type WebSocket interface {
		Write(msg []byte) error
		WriteBinary(msg []byte) error
		Set(key string, value interface{})
		Get(key string) (value interface{}, exists bool)
	}
	`)

	w(`type Logger interface {
		Log(str string)
		Logf(format string, a ...any)
	}
	`)

	// Write generic send data function
	w(`func sendPushMessage(ws WebSocket, name string, log Logger, data []byte) {
		log.Logf("Sending push message '%s' (%d bytes)", name, len(data))
		l := len(name) + 1 + len(data)
		payload := make([]byte, l)
		copy(payload, []byte(name))
		copy(payload[len(name)+1:], data)
		ws.WriteBinary(payload)
	}
	`)

	w(`func byteArrayToInt(b []byte) int {
		buf := bytes.NewBuffer(b)
		var n int32
		err := binary.Read(buf, binary.BigEndian, &n)
		if err != nil {
			return 0
		}
		return int(n)
	}
	`)

	w(`func intToByteArray(n int) []byte {
		// Create a buffer to hold the 4-byte array
		buf := new(bytes.Buffer)
		// Write the integer to the buffer in BigEndian format
		err := binary.Write(buf, binary.BigEndian, int32(n))
		if err != nil {
			panic(err)
		}
		// Return the byte slice
		return buf.Bytes()
	}`)

	formattedCode, err := format.Source([]byte(sb.String()))
	if err != nil {
		return sb.String(), err
	}
	return string(formattedCode), nil
}

func generateGoInterface(srv *unordered.Service, name string, withError bool) string {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s) }
	wn := func(s string) { sb.WriteString(s + "\n") }

	wn("type " + name + " interface {")
	for i, rpc := range srv.ServiceBody.RPCs {

		if len(rpc.Comments) > 0 && i > 0 {
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
			if withError {
				wn(" (*" + rpc.RPCResponse.MessageType + ", error)")
			} else {
				wn(" *" + rpc.RPCResponse.MessageType)
			}
		} else {
			if withError {
				wn(" error")
			} else {
				wn("")
			}
		}
	}
	wn("}")

	return sb.String()
}
