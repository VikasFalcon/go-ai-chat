package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/VikasFalcon/go-ai-chat/internal/model"
)

func Generate(req model.EmbeddingReq) (*model.EmbeddingRes, error) {
	embeddedURL := "http://localhost:11434/api/embed"
	payload := model.EmbedReq{
		Model: "nomic-embed-text",
		Input: req.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 50 * time.Second,
	}

	resp, err := client.Post(embeddedURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Embedding returned %d: %s", resp.StatusCode, string(b))
	}

	var EmbedRes model.EmbedRes
	if err := json.NewDecoder(resp.Body).Decode(&EmbedRes); err != nil {
		return nil, err
	}

	return &model.EmbeddingRes{
		Response: EmbedRes.Embeddings,
	}, nil

}
