package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/drawiks/odota-cli/parser"
)

var version = "dev"

func main() {
	url := flag.String("url", "http://localhost:5600", "odota/parser URL")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [--url URL] <match.dem>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Parse a Dota 2 replay and output structured JSON.\n")
		fmt.Fprintf(os.Stderr, "Output goes to stdout: %s match.dem > match.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "If input file has .ndjson extension, reads as raw parser output (no HTTP).\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	var events []parser.RawEvent
	var err error

	if strings.HasSuffix(inputFile, ".ndjson") || strings.HasSuffix(inputFile, ".json") {
		events, err = parser.ReadNDJSONFile(inputFile)
	} else {
		demData, rerr := os.ReadFile(inputFile)
		if rerr != nil {
			log.Fatalf("read dem file: %v", rerr)
		}
		events, err = parser.FetchFromParser(demData, *url)
	}
	if err != nil {
		log.Fatalf("parse: %v", err)
	}

	match, err := parser.Aggregate(events)
	if err != nil {
		log.Fatalf("aggregate: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(match); err != nil {
		log.Fatalf("encode: %v", err)
	}
}
