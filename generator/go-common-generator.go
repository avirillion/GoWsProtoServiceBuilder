package generator

import (
	"fmt"
	"go/format"
	"strings"
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
	wn := func(s string) { sb.WriteString(s + "\n") }

	wn("package " + pkg + "\n")
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

	// Write generic send data function
	wn(`func sendPushMessage(ws WebSocket, name string, log Logger, data []byte) {
		log.Logf("Sending push message '%s' (%d bytes)", name, len(data))
		l := len(name) + 1 + len(data)
		payload := make([]byte, l)
		copy(payload, []byte(name))
		copy(payload[len(name)+1:], data)
		ws.WriteBinary(payload)
	}`)

	formattedCode, err := format.Source([]byte(sb.String()))
	if err != nil {
		return sb.String(), err
	}
	return string(formattedCode), nil
}
