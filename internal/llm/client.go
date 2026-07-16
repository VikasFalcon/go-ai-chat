package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/VikasFalcon/go-ai-chat/internal/model"
)

func Generate(prompt string) (*model.ChatResp, error) {
	ollamaURL := "http://localhost:11434/api/generate"
	payload := model.OllamaRequest{
		Model:  "qwen3:8b",
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(body))
	// if err != nil {
	// 	return nil, err
	// }

	client := &http.Client{
		Timeout: 3000 * time.Second,
	}

	resp, err := client.Post(ollamaURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(b))
	}

	var ollamaResp model.OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, err
	}

	return &model.ChatResp{
		Answer: ollamaResp.Response,
	}, nil
}
