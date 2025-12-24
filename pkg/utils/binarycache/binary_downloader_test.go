package binarycache

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBinaryDownloader_VerboseOutput(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create downloader with verbose=true
	bd, err := NewBinaryDownloader(&Config{Verbose: true})
	if err != nil {
		if strings.Contains(err.Error(), "fetch trusted root") ||
			strings.Contains(err.Error(), "TUF") ||
			strings.Contains(err.Error(), "network") {
			t.Skipf("Skipping test: requires network access: %v", err)
		}
		t.Fatalf("Failed to create downloader: %v", err)
	}

	// Test that printf outputs when verbose=true
	bd.printf("test message %s\n", "hello")

	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe writer: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "test message hello") {
		t.Errorf("Expected verbose output, got: %q", output)
	}
}

func TestBinaryDownloader_SilentMode(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create downloader with verbose=false
	bd, err := NewBinaryDownloader(DefaultConfig())
	if err != nil {
		if strings.Contains(err.Error(), "fetch trusted root") ||
			strings.Contains(err.Error(), "TUF") ||
			strings.Contains(err.Error(), "network") {
			t.Skipf("Skipping test: requires network access: %v", err)
		}
		t.Fatalf("Failed to create downloader: %v", err)
	}

	// Test that printf does NOT output when verbose=false
	bd.printf("test message %s\n", "hello")

	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe writer: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output := buf.String()

	if strings.Contains(output, "test message hello") {
		t.Errorf("Expected no output in silent mode, got: %q", output)
	}
}
