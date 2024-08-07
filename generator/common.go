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

func writeInterface(srv *unordered.Service, name string, withError bool) string {
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
