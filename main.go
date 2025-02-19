package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	headerSize      = 4377
	chunkSize       = 512
	headerChunkEOF  = "\x00"
)

type FileHeader struct {
	Name   string
	Size   int64
	MTime  time.Time
	Prefix string
}

func main() {
	inputFile := flag.String("input", "", "Path to .wpress file")
	outputDir := flag.String("out", "", "Output directory")
	force := flag.Bool("force", false, "Overwrite existing files")
	flag.Parse()

	// Handle drag-and-drop for Windows users
	if *inputFile == "" {
		args := flag.Args()
		if len(args) > 0 {
			*inputFile = args[0]
		} else {
			fmt.Println("Error: Input file required")
			os.Exit(1)
		}
	}

	if err := extract(*inputFile, *outputDir, *force); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// [Rest of the existing code remains unchanged...]
