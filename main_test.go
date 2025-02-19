package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
		copy(buf[255:269], []byte("1024\x00\x00\x00\x00\x00\x00\x00\x00\x00"))  // Size field as string
		copy(buf[269:281], []byte("1672531200\x00"))                          // MTime as string
		copy(buf[281:], []byte("wp-content/uploads\x00"))

		r := bytes.NewReader(buf)
		header, err := readHeader(r)
		if err != nil {
			t.Fatal(err)
		}

		if header.Name != "test.txt" {
			t.Errorf("Expected name 'test.txt', got '%s'", header.Name)
		}
		if header.Size != 1024 {
			t.Errorf("Expected size 1024, got %d", header.Size)
		}
		if !header.MTime.Equal(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("Unexpected mtime: %v", header.MTime)
		}
		if header.Prefix != "wp-content/uploads" {
			t.Errorf("Unexpected prefix: %s", header.Prefix)
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
			t.Fatal(err)
		}

		err = extract("testdata/valid.wpress", filepath.Join(tmpDir, "existing"), false)
		if err == nil {
			t.Error("Expected error about existing directory")
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

func TestCLIArguments(t *testing.T) {
	// Backup and restore original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("PositionalArgument", func(t *testing.T) {
		os.Args = []string{"cmd", "test.wpress"}
		main()
	})

	t.Run("FlagArgument", func(t *testing.T) {
		os.Args = []string{"cmd", "-input", "test.wpress"}
		main()
	})
}
