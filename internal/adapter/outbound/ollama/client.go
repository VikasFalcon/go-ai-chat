package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	chatModel  string
	embedModel string
}

func NewClient(baseURL, chatModel, embedModel string, timeout time.Duration) *Client {
	return &Client{httpClient: &http.Client{Timeout: timeout}, baseURL: baseURL, chatModel: chatModel, embedModel: embedModel}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}
type generateResponse struct {
	Response string `json:"response"`
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	var out generateResponse
	if err := c.postJSON(ctx, "/api/generate", generateRequest{Model: c.chatModel, Prompt: prompt}, &out); err != nil {
		return "", fmt.Errorf("ollama generate: %w", err)
	}
	return out.Response, nil
}

type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}
type embedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	var out embedResponse
	if err := c.postJSON(ctx, "/api/embed", embedRequest{Model: c.embedModel, Input: text}, &out); err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	if len(out.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embed: no embeddings returned")
	}
	return out.Embeddings[0], nil
}

func (c *Client) postJSON(ctx context.Context, path string, reqBody, out any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s returned %d: %s", path, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
