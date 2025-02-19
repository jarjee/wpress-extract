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
	headerChunkEOF  = "\x00" // Simplified EOF check
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

	if *inputFile == "" {
		fmt.Println("Error: Input file required")
		os.Exit(1)
	}

	if err := extract(*inputFile, *outputDir, *force); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func extract(inputPath, outputPath string, force bool) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer file.Close()

	if outputPath == "" {
		base := filepath.Base(inputPath)
		outputPath = base[:len(base)-len(filepath.Ext(base))]
	}

	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("output directory exists - use --force to overwrite")
		}
	}

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	for {
		header, err := readHeader(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read header: %w", err)
		}

		destPath := filepath.Join(outputPath, header.Prefix, header.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("create directory structure: %w", err)
		}

		if err := writeFile(file, destPath, header.Size); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	return nil
}

func readHeader(r io.Reader) (*FileHeader, error) {
	buf := make([]byte, headerSize)
	n, err := r.Read(buf)
	if err != nil || n < headerSize {
		return nil, io.EOF
	}

	if bytes.Equal(buf[:1], []byte(headerChunkEOF)) {
		return nil, io.EOF
	}

	return &FileHeader{
		Name:   string(bytes.Trim(buf[0:255], "\x00")),
		Size:   readInt64(buf[255:269]),
		MTime:  time.Unix(readInt64(buf[269:281]), 0),
		Prefix: string(bytes.Trim(buf[281:headerSize], "\x00")),
	}, nil
}

func readInt64(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b[:8]))
}

func writeFile(r io.Reader, dest string, size int64) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	remaining := size
	buf := make([]byte, chunkSize)

	for remaining > 0 {
		readSize := chunkSize
		if remaining < chunkSize {
			readSize = int(remaining)
		}

		n, err := r.Read(buf[:readSize])
		if err != nil {
			return err
		}

		if _, err := f.Write(buf[:n]); err != nil {
			return err
		}

		remaining -= int64(n)
	}

	return nil
}
