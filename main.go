package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"strconv"
	"os"
	"path/filepath"
	"time"
)

const (
	headerSize      = 4377
	chunkSize       = 512
)

var emptyHeader = make([]byte, headerSize)

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
	mode := flag.String("mode", "extract", "Operation mode: extract|compress")
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

	switch *mode {
	case "extract":
		if err := extract(*inputFile, *outputDir, *force); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "compress":
		if *outputDir == "" {
			*outputDir = *inputFile + ".wpress"
		}
		if err := compress(*inputFile, *outputDir); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Error: Invalid mode '%s'\n", *mode)
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

func writeHeader(w io.Writer, h *FileHeader) error {
	buf := make([]byte, headerSize)
	
	copy(buf[0:255], []byte(h.Name))
	copy(buf[255:269], []byte(strconv.FormatInt(h.Size, 10)))
	copy(buf[269:281], []byte(strconv.FormatInt(h.MTime.Unix(), 10)))
	copy(buf[281:], []byte(h.Prefix))
	
	_, err := w.Write(buf)
	return err
}

func compress(inputPath, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
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
