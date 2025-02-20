package main

import (
	"os"
	"testing"
	"bytes"
    "fmt"
    "io"
    "os/exec"
     "path/filepath"
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
	if err != nil || string(content1) != "test" {
		t.Error("First file content mismatch")
	}

	content2, err := os.ReadFile(filepath.Join(extractDir, "sub", "file2.txt"))
	if err != nil || string(content2) != "test2" {
		t.Error("Second file content mismatch")
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
		t.Errorf("Header name mismatch")
	}
	if parsed.Size != h.Size {
		t.Errorf("Header size mismatch")
	}
	if !parsed.MTime.Equal(h.MTime) {
		t.Errorf("Header mtime mismatch")
	}
	if parsed.Prefix != h.Prefix {
		t.Errorf("Header prefix mismatch")
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

func TestCLIArguments(t *testing.T) {
	// Backup and restore original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Create temporary test file
	tmpFile, err := os.CreateTemp("", "test-*.wpress")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	
	// Write minimal valid header
	tmpFile.Write(make([]byte, headerSize))
	tmpFile.Close()

	t.Run("PositionalArgument", func(t *testing.T) {
		os.Args = []string{"cmd", tmpFile.Name()}
		main()
	})

	t.Run("FlagArgument", func(t *testing.T) {
		os.Args = []string{"cmd", "-input", tmpFile.Name()}
		main()
	})
}
