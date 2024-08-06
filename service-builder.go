package main

import (
	servicebuilder "GoWsProtoServiceBuilder/service-builder"
	"fmt"
	"log"
	"os"

	pp "github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

func main() {
	if len(os.Args) != 4 {
		printHelp()
		os.Exit(1)
	}
	protoBufFile := os.Args[1]
	goServiceFile := os.Args[2]

	pbuf, err := parseProtoBuf(protoBufFile)
	if err != nil {
		log.Fatal(err)
	}

	err = servicebuilder.GenerateGoService(pbuf, goServiceFile)
	if err != nil {
		log.Fatal(err)
	}
}

func printHelp() {
	fmt.Println(`Usage:
	service-builder <protobuf-file> <go-service-dir> <ts-service-dir>`)
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
