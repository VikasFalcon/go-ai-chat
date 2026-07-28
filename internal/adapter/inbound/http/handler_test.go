package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

type recordingChatService struct {
	ingestPath  string
	ingestCalls int
}

func (s *recordingChatService) Chat(context.Context, string) (string, error) { return "", nil }
func (s *recordingChatService) Ask(context.Context, string) (string, error)  { return "", nil }
func (s *recordingChatService) Ingest(context.Context, string) error         { return nil }
func (s *recordingChatService) DocumentCount() int                           { return 0 }
func (s *recordingChatService) IngestPDF(_ context.Context, path string) (int, error) {
	s.ingestCalls++
	s.ingestPath = path
	if _, err := os.Stat(path); err != nil {
		return 0, err
	}
	return 2, nil
}

func TestIngestUsesIsolatedTemporaryFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	service := &recordingChatService{}
	h := NewHandler(service, nil, 1, tmpDir)
	router := NewRouter(h)

	body, contentType := multipartUpload(t, "nested/guide.PDF", []byte("%PDF-1.7\nexample"))
	req := httptest.NewRequest(http.MethodPost, "/api/ingest", body)
	req.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.ingestCalls != 1 {
		t.Fatalf("ingest calls = %d, want 1", service.ingestCalls)
	}
	if filepath.Base(service.ingestPath) != "guide.PDF" {
		t.Fatalf("stored filename = %q, want sanitized basename", service.ingestPath)
	}
	if filepath.Dir(service.ingestPath) == tmpDir {
		t.Fatal("upload was written directly to the shared temporary directory")
	}

	var result IngestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Source != "guide.PDF" || result.ChunksIngested != 2 {
		t.Fatalf("unexpected response: %+v", result)
	}
	if _, err := os.Stat(service.ingestPath); !os.IsNotExist(err) {
		t.Fatalf("temporary upload was not removed, stat error = %v", err)
	}
}

func TestIngestRejectsNonPDFContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &recordingChatService{}
	router := NewRouter(NewHandler(service, nil, 1, t.TempDir()))

	body, contentType := multipartUpload(t, "renamed.pdf", []byte("not a PDF"))
	req := httptest.NewRequest(http.MethodPost, "/api/ingest", body)
	req.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.ingestCalls != 0 {
		t.Fatalf("ingest calls = %d, want 0", service.ingestCalls)
	}
	var result errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Error != "uploaded file is not a valid PDF" {
		t.Fatalf("error = %q", result.Error)
	}
}

func multipartUpload(t *testing.T, filename string, content []byte) (io.Reader, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}
