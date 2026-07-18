package prompt

import (
	"bytes"
	"embed"
	"fmt"
	"text/template" // was html/template — that HTML-escapes quotes/&/etc in your context, corrupting the prompt
)

//go:embed rag.tmpl
var templateFS embed.FS

type Builder struct{ tmpl *template.Template }

func NewBuilder() (*Builder, error) {
	tmpl, err := template.ParseFS(templateFS, "rag.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse prompt template: %w", err)
	}
	return &Builder{tmpl: tmpl}, nil
}

type promptData struct{ Chunks, Question string }

func (b *Builder) Build(chunks, question string) (string, error) {
	var buf bytes.Buffer
	if err := b.tmpl.Execute(&buf, promptData{chunks, question}); err != nil {
		return "", fmt.Errorf("render prompt template: %w", err)
	}
	return buf.String(), nil
}
