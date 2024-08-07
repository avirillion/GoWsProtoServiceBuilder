package generator

import (
	"strings"

	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
)

func generateTsInterface(srv *unordered.Service, name string, namePrefix string) string {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s) }
	wn := func(s string) { sb.WriteString(s + "\n") }

	wn("export interface " + name + " {")
	for i, rpc := range srv.ServiceBody.RPCs {

		if len(rpc.Comments) > 0 && i > 0 {
			wn("")
		}
		for _, c := range rpc.Comments {
			wn("  " + c.Raw)
		}
		if namePrefix != "" {
			w("  " + namePrefix + firstCharToUpper(rpc.RPCName) + "(")
		} else {
			w("  " + firstCharToLower(rpc.RPCName) + "(")
		}

		// parameter
		if rpc.RPCRequest.MessageType != voidTypeName {
			w("param: " + rpc.RPCRequest.MessageType)
		}
		w(")")

		// response
		if rpc.RPCResponse.MessageType != voidTypeName {
			wn(": Promise<" + rpc.RPCResponse.MessageType + ">")
		} else {
			wn(": Promise<void>")
		}
	}
	wn("}\n")

	return sb.String()
}
