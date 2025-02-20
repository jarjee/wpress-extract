package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Generate test data before running tests
	if err := exec.Command("./generate_testdata.sh").Run(); err != nil {
		fmt.Printf("Failed to generate testdata: %v", err)
		os.Exit(1)
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestReadHeader(t *testing.T) {
	t.Run("EOF", func(t *testing.T) {
		buf := make([]byte, headerSize)
		buf[0] = 0x00
		r := bytes.NewReader(buf)

		_, err := readHeader(r)
		if err != io.EOF {
			t.Errorf("Expected EOF, got %v", err)
		}
	})

	t.Run("ValidHeader", func(t *testing.T) {
		buf := make([]byte, headerSize)
		copy(buf[0:255], []byte("test.txt\x00"))
		copy(buf[255:269], []byte("1024\x00\x00\x00\x00\x00\x00\x00\x00\x00")) // Size field as string
		copy(buf[269:281], []byte("1672531200\x00"))                           // MTime as string
		copy(buf[281:], []byte("wp-content/uploads\x00"))

		r := bytes.NewReader(buf)
		header, err := readHeader(r)
		if err != nil {
			t.Fatal("Failed to compress directory with trailing slash:", err)
		}

		if header.Name != "test.txt" {
			t.Errorf("Header.Name: expected 'test.txt', got '%s'", header.Name)
		}
		if header.Size != 1024 {
			t.Errorf("Header.Size: expected 1024, got %d", header.Size)
		}
		expectedTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		if !header.MTime.Equal(expectedTime) {
			t.Errorf("Header.MTime: expected %v, got %v", expectedTime, header.MTime)
		}
		if header.Prefix != "wp-content/uploads" {
			t.Errorf("Header.Prefix: expected 'wp-content/uploads', got '%s'", header.Prefix)
		}
	})
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "testfile.txt")

	r := bytes.NewReader([]byte("test content"))
	err := writeFile(r, testPath, 12)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "test content" {
		t.Errorf("Unexpected file content: %s", content)
	}
}

func TestExtract(t *testing.T) {
	t.Run("OutputDirectoryExists", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := os.Mkdir(filepath.Join(tmpDir, "existing"), 0755)
		if err != nil {
			t.Fatal("Failed to create nested output directory:", err)
		}

		err = extract("testdata/valid.wpress", filepath.Join(tmpDir, "existing"), false)
		if err == nil {
			t.Error("Expected 'directory exists' error, got nil")
		} else if err.Error() != "output directory already exists" {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("ForceOverwrite", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := os.Mkdir(filepath.Join(tmpDir, "existing"), 0755)
		if err != nil {
			t.Fatal(err)
		}

		err = extract("testdata/valid.wpress", filepath.Join(tmpDir, "existing"), true)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func TestCompress(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdata")
	os.MkdirAll(filepath.Join(testDir, "sub"), 0755)
	os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "sub", "file2.txt"), []byte("test2"), 0644)

	output := filepath.Join(tmpDir, "test.wpress")
	err := compress(testDir, output)
	if err != nil {
		t.Fatal(err)
	}

	// Verify by extracting
	extractDir := filepath.Join(tmpDir, "extracted")
	err = extract(output, extractDir, true)
	if err != nil {
		t.Fatal(err)
	}

	// Verify file contents
	content1, err := os.ReadFile(filepath.Join(extractDir, "file.txt"))
	if err != nil {
		t.Error("Failed to read file.txt:", err)
	} else if string(content1) != "test" {
		t.Error("Compress/extract: file.txt content mismatch")
	}

	content2, err := os.ReadFile(filepath.Join(extractDir, "sub", "file2.txt"))
	if err != nil {
		t.Error("Failed to read sub/file2.txt:", err)
	} else if string(content2) != "test2" {
		t.Error("Compress/extract: sub/file2.txt content mismatch")
	}
}

func TestWriteHeader(t *testing.T) {
	h := &FileHeader{
		Name:   "test.txt",
		Size:   1234,
		MTime:  time.Unix(1672531200, 0),
		Prefix: "subdir",
	}

	buf := &bytes.Buffer{}
	err := writeHeader(buf, h)
	if err != nil {
		t.Fatal(err)
	}

	if len(buf.Bytes()) != headerSize {
		t.Errorf("Invalid header size")
	}

	// Verify header can be read back
	parsed, err := readHeader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name != h.Name {
		t.Errorf("Header.Name mismatch: expected '%s', got '%s'", h.Name, parsed.Name)
	}
	if parsed.Size != h.Size {
		t.Errorf("Header.Size mismatch: expected %d, got %d", h.Size, parsed.Size)
	}
	if !parsed.MTime.Equal(h.MTime) {
		t.Errorf("Header.MTime mismatch: expected %v, got %v", h.MTime, parsed.MTime)
	}
	if parsed.Prefix != h.Prefix {
		t.Errorf("Header.Prefix mismatch: expected '%s', got '%s'", h.Prefix, parsed.Prefix)
	}
}

func TestCompressPaths(t *testing.T) {
	t.Run("TrailingSlashInput", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "testinput") + string(filepath.Separator)
		os.MkdirAll(testDir, 0755)
		os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test"), 0644)

		output := filepath.Join(tmpDir, "output.wpress")
		err := compress(testDir, output)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("NestedOutputPath", func(t *testing.T) {
		tmpDir := t.TempDir()
		testDir := filepath.Join(tmpDir, "testdata")
		os.MkdirAll(testDir, 0755)
		os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test"), 0644)

		output := filepath.Join(tmpDir, "nonexistent/directory/output.wpress")
		err := compress(testDir, output)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestRun(t *testing.T) {
	// Create temporary test file
	tmpFile, err := os.CreateTemp("", "test-*.wpress")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write minimal valid header
	tmpFile.Write(make([]byte, headerSize))
	tmpFile.Close()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "No arguments",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "Positional argument",
			args:    []string{tmpFile.Name()},
			wantErr: false,
		},
		{
			name:    "Flag argument",
			args:    []string{"-input", tmpFile.Name()},
			wantErr: false,
		},
		{
			name:    "Invalid mode",
			args:    []string{"-mode", "invalid", tmpFile.Name()},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
