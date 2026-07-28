package pdfloader

import (
	"strings"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	l := NewLoader()
	text, err := l.Load("testdata/sample.pdf")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(text, "Hyperledger Fabric") {
		t.Fatalf("expected extracted text to contain known content, got: %q", text)
	}
	if !strings.Contains(text, "Kafka") {
		t.Fatalf("expected extracted text to contain 'Kafka', got: %q", text)
	}
}

func TestLoader_MissingFile(t *testing.T) {
	l := NewLoader()
	if _, err := l.Load("testdata/does-not-exist.pdf"); err == nil {
		t.Fatalf("expected an error for a missing file")
	}
}
