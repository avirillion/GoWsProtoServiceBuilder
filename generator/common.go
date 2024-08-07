package generator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
)

const generatorWarning = "// THIS FILE WAS AUTOMATICALLY GENERATED\n// DO NOT MODIFY!\n\n"
const voidTypeName = "Void"
const errorTypeName = "Error"

type dtoCollectorType = map[string]struct{}

func writeFile(filename string, text string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	file.WriteString(text)
	return nil
}

func hasServiceOption(srv *unordered.Service, name string) bool {
	hasOption := false
	for _, opt := range srv.ServiceBody.Options {
		if opt.OptionName == name {
			hasOption = true
			break
		}
	}
	return hasOption
}

func firstCharToUpper(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func firstCharToLower(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[0:1]) + s[1:]
}
