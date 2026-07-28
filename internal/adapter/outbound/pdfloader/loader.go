// Package pdfloader extracts plain text from PDF files. It implements
// port.DocumentLoader using a pure-Go PDF parser, so no external binaries
// (poppler, pdftotext, etc.) are required.
package pdfloader

import (
	"bytes"
	"fmt"

	"github.com/ledongthuc/pdf"
)

type Loader struct{}

func NewLoader() *Loader { return &Loader{} }

// Load reads the PDF at path and returns its concatenated plain text across
// all pages.
func (l *Loader) Load(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	textReader, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extract pdf text: %w", err)
	}
	if _, err := buf.ReadFrom(textReader); err != nil {
		return "", fmt.Errorf("read extracted pdf text: %w", err)
	}
	return buf.String(), nil
}
