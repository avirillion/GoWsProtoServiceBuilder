package main

import (
	servicebuilder "GoWsProtoServiceBuilder/generator"
	"fmt"
	"log"
	"os"

	pp "github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

func main() {
	if len(os.Args) != 6 {
		printHelp()
		os.Exit(1)
	}
	protoBufPath := os.Args[1]
	protoBufFile := os.Args[2]
	goBaseDir := os.Args[3]
	goPackage := os.Args[4]
	tsBaseDir := os.Args[5]

	pbuf, err := parseProtoBuf(protoBufFile)
	if err != nil {
		log.Fatal(err)
	}

	err = servicebuilder.GenerateGoCommon(goBaseDir, goPackage)
	if err != nil {
		log.Fatal(err)
	}
	err = servicebuilder.GenerateGoRpcService(pbuf, goBaseDir, goPackage)
	if err != nil {
		log.Fatal(err)
	}
	err = servicebuilder.GenerateGoSspService(pbuf, goBaseDir, goPackage)
	if err != nil {
		log.Fatal(err)
	}

	err = servicebuilder.GenerateTypeScriptFile(pbuf, protoBufPath, protoBufFile, tsBaseDir)
	if err != nil {
		log.Fatal(err)
	}
}

func printHelp() {
	fmt.Println(`Usage:
	service-builder <protobuf-path> <protobuf-file> <go-base-dir> <go-package> <ts-service-dir>`)
}

func parseProtoBuf(file string) (*parser.Proto, error) {
	reader, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s, err %v", file, err)
	}
	defer reader.Close()

	got, err := pp.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse, err %v", err)
	}
	return got, nil
}
