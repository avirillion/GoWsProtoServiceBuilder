package generator

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/yoheimuta/go-protoparser/v4"
	pp "github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

func GenerateTypeScriptFile(pbuf *parser.Proto, protoDir string, protoFile string, tsBaseDir string) error {
	pb, err := protoparser.UnorderedInterpret(pbuf)
	if err != nil {
		return err
	}

	code := generateTypeScriptCode(pb, protoDir, protoFile)

	filename := fmt.Sprintf("%s/%s.ts", tsBaseDir, "rpc-handler_gen")
	err = writeFile(filename, code)
	if err != nil {
		return err
	}
	return nil
}

func generateTypeScriptCode(pb *unordered.Proto, protoDir string, protoFile string) string {
	dtoCollector := make(dtoCollectorType)
	dtoCollector["Error"] = struct{}{}

	types := generateTypeScriptHeader()
	serverClass := generateRpcServerClass(pb)
	encoder := generateEncoder()
	rpcServiceImpls := generateRpcServices(pb, dtoCollector)
	sspServiceImpls := generateSspServices(pb, dtoCollector)
	imports := generateImports(pb, protoDir, protoFile, dtoCollector)

	return imports + types + serverClass + encoder + rpcServiceImpls + sspServiceImpls
}

func generateTypeScriptHeader() string {
	return generatorWarning + `
type ResolveFunctions = {
  name: string;
  resolve?: (arg: any) => void;
  reject?: (arg: any) => void;
};

`
}

func generateRpcServerClass(pb *unordered.Proto) string {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s + "\n") }

	sb.WriteString(`export class Server {
  private readonly ws: WebSocket;
  private readonly requestMap: { [key: number]: ResolveFunctions } = {};
  private readonly callbackListeners: { [key: string]: ((data: unknown) => void)[] } = {};
  private nextMessageId: number = 1;
`)
	for _, srv := range pb.ProtoBody.Services {
		w("  readonly " + firstCharToLower(srv.ServiceName) + ": " + srv.ServiceName + "Impl;")
	}
	w("")

	w(`  constructor(ws: WebSocket) {
    this.ws = ws;
`)
	for _, srv := range pb.ProtoBody.Services {
		w("    this." + firstCharToLower(srv.ServiceName) + " = new " + srv.ServiceName + "Impl(this);")
	}
	w("")
	w("    this.initMessageHandler();")
	w("  }\n")

	w(`  private initMessageHandler() {
    this.ws.onmessage = (evt) => {
      const msg = decode(evt.data);

      if (msg.id) {
        const promises = this.requestMap[msg.id] || this.requestMap[-msg.id];
        if (!promises) {
          console.error('No promise found for id ' + msg.id);
          return;
        }

        if (msg.id > 0) {
          promises.resolve!(msg.data);
        } else {
          let err = Error.decode(msg.data);
          promises.reject!(err.Error);
        }

        delete this.requestMap[msg.id]
      } else {
        // call all registered listeners
        const listeners = this.callbackListeners[msg.name!];
        if (listeners) {
          listeners.forEach((cb) => cb(msg.data));
        } else {
          console.error("No listener for: ", msg.name);
        }
      }
    }
  }
`)

	w(`  registerCallbackHandler(name: string, cb: (data: Uint8Array) => void) {
    let listeners = this.callbackListeners[name];
    if (!listeners) {
      listeners = [];
      this.callbackListeners[name] = listeners;
    }
    listeners.push(cb as (data: unknown) => void);
  }
`)

	w(`  rpc(name: string, data: Uint8Array): Promise<Uint8Array> {
    const id = this.nextMessageId++;
    const request = encode(id, name, data);
    const promiseFunctions: ResolveFunctions = { name };
    const promise = new Promise((resolve, reject) => {
      promiseFunctions.resolve = resolve;
      promiseFunctions.reject = reject;
    });
    this.requestMap[id] = promiseFunctions;
    this.ws.send(request);
    return promise as Promise<Uint8Array>;
  }`)

	w("}")
	return sb.String()
}

func generateEncoder() string {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s + "\n") }

	w(`
export type ResponseContainer = {
  name?: string;
  id: number;
  data: Uint8Array;
};

/**
 * Encodes an RPC request to a binary representation
 */
function encode(id: number, name: string, data: Uint8Array): Uint8Array {
  // Convert name
  const encoder = new TextEncoder();
  const nameAsBytes = encoder.encode(name);

  // Convert number
  const arrayBuffer = new ArrayBuffer(4); // 4 bytes for a 32-bit integer
  const dataView = new DataView(arrayBuffer);
  dataView.setUint32(0, id, false); // Big-endian byte order
  const idAsBytes = new Uint8Array(arrayBuffer);

  // Concatenate arrays
  const len = nameAsBytes.length + idAsBytes.length + data.length;
  const result = new Uint8Array(len);
  result.set(nameAsBytes, 0);
  result.set(idAsBytes, nameAsBytes.length);
  result.set(data, nameAsBytes.length + idAsBytes.length);

  return result;
}

/**
 * Decodes an RPC or callback response from the binary representation
 */
function decode(data: ArrayBuffer): ResponseContainer {
  let name = "";
  // Find first 0 or FF
  let arr = new Uint8Array(data);
  for (let i = 0; i < arr.length; ++i) {
    if (arr[i] == 0 || arr[i] == 255) {
      const nameSlice = data.slice(0, i);
      name = new TextDecoder().decode(nameSlice);
      break;
    }
  }

  let id = 0;
  let dataOffset = 1;
  if (name === "") {
    id = new DataView(data, name.length, name.length + 4).getInt32(0);
    dataOffset = 4;
  }

  return {
    id,
    name,
    data: new Uint8Array(data.slice(name.length + dataOffset)),
  };
}
`)
	return sb.String()
}

func generateRpcServices(pb *unordered.Proto, dto dtoCollectorType) string {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s) }
	wn := func(s string) { sb.WriteString(s + "\n") }

	wn("")
	for _, srv := range pb.ProtoBody.Services {
		if !hasServiceOption(srv, "(is_rpc)") {
			continue
		}

		wn("export class " + srv.ServiceName + "Impl {")
		wn("  constructor(private server: Server) {}")
		wn("")
		for _, rpc := range srv.ServiceBody.RPCs {
			dto[rpc.RPCRequest.MessageType] = struct{}{}
			dto[rpc.RPCResponse.MessageType] = struct{}{}
			w("  public async " + firstCharToLower(rpc.RPCName) + "(")
			if rpc.RPCRequest.MessageType != voidTypeName {
				w("prm: " + rpc.RPCRequest.MessageType)
			}
			w(")")

			if rpc.RPCResponse.MessageType != voidTypeName {
				wn(": Promise<" + rpc.RPCResponse.MessageType + "> {")
			} else {
				wn(": Promise<void> {")
			}

			if rpc.RPCRequest.MessageType != voidTypeName {
				wn("    const data = " + rpc.RPCRequest.MessageType + ".encode(prm).finish();")
			} else {
				wn("    const data = new Uint8Array([]);")
			}

			if rpc.RPCResponse.MessageType != voidTypeName {
				wn("    const responseData = await this.server.rpc('" + rpc.RPCName + "', data);")
				wn("    const responseObj = " + rpc.RPCResponse.MessageType + ".decode(responseData);")
				wn("    return responseObj;")
			} else {
				wn("    await this.server.rpc('" + rpc.RPCName + "', data);")
			}
			wn("  }\n")
		}
		wn("}")
	}
	return sb.String()
}

func generateSspServices(pb *unordered.Proto, dto dtoCollectorType) string {
	var sb strings.Builder
	w := func(s string) { sb.WriteString(s) }
	wn := func(s string) { sb.WriteString(s + "\n") }

	wn("")
	for _, srv := range pb.ProtoBody.Services {
		if !hasServiceOption(srv, "(is_ssp)") {
			continue
		}

		wn("export class " + srv.ServiceName + "Impl {")
		wn("  private readonly callbackListeners: { [key: string]: ((data: unknown) => void)[] } = {};\n")
		wn("  constructor(server: Server) {")
		for _, rpc := range srv.ServiceBody.RPCs {
			wn("    server.registerCallbackHandler('" + rpc.RPCName + "', this." + firstCharToLower(rpc.RPCName) + ".bind(this));")
		}
		wn("  }\n")

		wn(`  private registerCallbackHandler(name: string, cb: (data: unknown) => void) {
    let listeners = this.callbackListeners[name];
    if (!listeners) {
      listeners = [];
      this.callbackListeners[name] = listeners;
    }
    listeners.push(cb as (data: unknown) => void);
  }

  private callback(name: string, data: unknown) {
    const listeners = this.callbackListeners[name];
    if (listeners) {
      listeners.forEach((cb) => cb(data));
    } else {
      console.error("No listener for: ", name);
    }
  }
`)

		for _, rpc := range srv.ServiceBody.RPCs {
			dto[rpc.RPCRequest.MessageType] = struct{}{}
			dto[rpc.RPCResponse.MessageType] = struct{}{}

			w("  public on" + firstCharToUpper(rpc.RPCName) + "(")
			if rpc.RPCRequest.MessageType != voidTypeName {
				w("cb: (p: " + rpc.RPCRequest.MessageType + ") => void")
			}
			wn(") {")
			wn("    this.registerCallbackHandler('" + rpc.RPCName + "', cb as (data: unknown) => void);")
			wn("  }\n")

			wn("  private " + firstCharToLower(rpc.RPCName) + "(rawData: Uint8Array) {")
			wn("    const obj = " + rpc.RPCRequest.MessageType + ".decode(rawData);")
			wn("    this.callback('" + rpc.RPCName + "', obj);")
			wn("  }\n")

		}
		wn("}")
	}
	return sb.String()
}

func generateImports(pb *unordered.Proto, protoDir string, protoFile string, dto dtoCollectorType) string {
	var sb strings.Builder

	if strings.HasPrefix(protoFile, protoDir) {
		protoFile = protoFile[len(protoDir)+1:]
	}
	imports := make(map[string]*unordered.Proto)
	imports[protoFile[:len(protoFile)-6]] = pb

	// Generate map of all imported files
	for _, imp := range pb.ProtoBody.Imports {
		file := imp.Location[1 : len(imp.Location)-1]
		reader, err := os.Open(protoDir + "/" + file)
		if err != nil {
			log.Printf("Warning: Failed to open '%s', error: %v; Skipping file.", file, err)
			continue
		}
		defer reader.Close()

		got, err := pp.Parse(reader)
		if err != nil {
			log.Printf("Warning: Failed to parse '%s', error: %v", file, err)
		}
		pb, err := protoparser.UnorderedInterpret(got)
		if err != nil {
			log.Printf("Warning: Failed to interpret '%s', error: %v", file, err)
		}
		fileNameBase := file[:len(file)-6]
		imports[fileNameBase] = pb
	}

	// Convert from type1:file1, type2:file1 to file1:[type1,type2]
	fileImports := make(map[string][]string)
	for typ := range dto {
		if typ == "Void" {
			continue
		}
		// Find references
	out:
		for pbFile, pbData := range imports {
			for _, msg := range pbData.ProtoBody.Messages {
				if msg.MessageName == typ {
					fileImports[pbFile] = append(fileImports[pbFile], typ)
					break out
				}
			}
		}
	}

	w := func(s string) { sb.WriteString(s + "\n") }

	for file, types := range fileImports {
		slices.Sort(types)
		w("import { " + strings.Join(types, ", ") + " } from './" + file + "';")
	}
	return sb.String()
}
