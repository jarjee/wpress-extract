package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	headerSize = 4377
	chunkSize  = 512
)

var emptyHeader = make([]byte, headerSize)

type FileHeader struct {
	Name   string
	Size   int64
	MTime  time.Time
	Prefix string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("wpress", flag.ContinueOnError)
	inputFile := fs.String("input", "", "Path to .wpress file")
	outputDir := fs.String("out", "", "Output directory") 
	force := fs.Bool("force", false, "Overwrite existing files")
	mode := fs.String("mode", "extract", "Operation mode: extract|compress")

	// Parse flags with error handling
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Handle drag-and-drop for basic users
	if *inputFile == "" {
		remaining := fs.Args()
		if len(remaining) > 0 {
			*inputFile = remaining[0]
		} else {
			return fmt.Errorf("input file required")
		}
	}

	switch *mode {
	case "extract":
		if err := extract(*inputFile, *outputDir, *force); err != nil {
			return fmt.Errorf("extract: %w", err)
		}
	case "compress":
		outputFile := ""
		if *outputDir == "" {
			outputFile = filepath.Base(*inputFile) + ".wpress"
		} else {
			cleaned := filepath.Clean(*outputDir)
			if strings.HasSuffix(cleaned, string(filepath.Separator)) || isExistingDir(cleaned) {
				outputFile = filepath.Join(cleaned, filepath.Base(*inputFile)+".wpress")
			} else {
				outputFile = cleaned
			}
		}
		if err := compress(*inputFile, outputFile); err != nil {
			return fmt.Errorf("compress: %w", err)
		}
	default:
		return fmt.Errorf("invalid mode '%s'", *mode)
	}

	return nil
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
			return fmt.Errorf("output directory already exists")
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

func writeHeader(w io.Writer, h *FileHeader) error {
	buf := make([]byte, headerSize)

	copy(buf[0:255], []byte(h.Name))
	copy(buf[255:269], []byte(strconv.FormatInt(h.Size, 10)))
	copy(buf[269:281], []byte(strconv.FormatInt(h.MTime.Unix(), 10)))
	copy(buf[281:], []byte(h.Prefix))

	_, err := w.Write(buf)
	return err
}

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func compress(inputPath, outputPath string) error {
	inputPath = filepath.Clean(inputPath) // Normalize input path

	// Create parent directories for output file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output directory structure: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	// Get absolute path of output file to exclude from processing
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("absolute output path: %w", err)
	}

	return filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and the output file itself
		currentAbs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("absolute path: %w", err)
		}
		if currentAbs == outputAbs {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(inputPath, path)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		prefix := filepath.Dir(relPath)
		if prefix == "." {
			prefix = ""
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		header := &FileHeader{
			Name:   filepath.Base(relPath),
			Size:   info.Size(),
			MTime:  info.ModTime(),
			Prefix: prefix,
		}

		if err := writeHeader(file, header); err != nil {
			return err
		}

		_, err = io.Copy(file, f)
		return err
	})
}

func readHeader(r io.Reader) (*FileHeader, error) {
	buf := make([]byte, headerSize)
	n, err := io.ReadFull(r, buf)
	if err == io.EOF || n < headerSize {
		return nil, io.EOF
	}

	// Check for full empty header (all 0x00 bytes)
	if bytes.Equal(buf, emptyHeader) {
		return nil, io.EOF
	}

	// Parse Size
	sizeStr := string(bytes.TrimRight(buf[255:269], "\x00"))
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size: %w", err)
	}

	// Parse MTime
	mtimeStr := string(bytes.TrimRight(buf[269:281], "\x00"))
	mtime, err := strconv.ParseInt(mtimeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid mtime: %w", err)
	}

	return &FileHeader{
		Name:   string(bytes.TrimRight(buf[0:255], "\x00")),
		Size:   size,
		MTime:  time.Unix(mtime, 0),
		Prefix: string(bytes.TrimRight(buf[281:headerSize], "\x00")),
	}, nil
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
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			remaining -= int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	if remaining != 0 {
		return fmt.Errorf("incomplete file data - expected %d more bytes", remaining)
	}

	return nil
}
