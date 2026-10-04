package chess

import (
	"io"
	"os"
	"testing"
)

func captureStdout(t *testing.T, print func()) string {
	t.Helper()

	previous := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned an error: %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = previous
	}()

	print()
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() returned an error: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() returned an error: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close() returned an error: %v", err)
	}

	return string(output)
}
