// Package embedding provides vector embedding generation via OpenAI's API.
// Anthropic Claude does not offer embeddings, so we use OpenAI
// text-embedding-3-small (1536 dimensions) which matches the pgvector schema.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultModel     = "text-embedding-3-small"
	defaultDimension = 1536
	openAIEmbedURL   = "https://api.openai.com/v1/embeddings"
)

// Service generates text embeddings via OpenAI.
type Service struct {
	apiKey     string
	model      string
	dimensions int
	httpClient *http.Client
}

// NewService creates an embedding Service.
func NewService(apiKey, model string, dimensions int) *Service {
	if model == "" {
		model = defaultModel
	}
	if dimensions == 0 {
		dimensions = defaultDimension
	}
	return &Service{
		apiKey:     apiKey,
		model:      model,
		dimensions: dimensions,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type embedRequest struct {
	Input      string `json:"input"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Embed generates a vector embedding for the given text.
func (s *Service) Embed(ctx context.Context, text string) ([]float32, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not configured")
	}
	if text == "" {
		return nil, fmt.Errorf("cannot embed empty text")
	}

	payload := embedRequest{
		Input:      text,
		Model:      s.model,
		Dimensions: s.dimensions,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEmbedURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call openai embed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embed response: %w", err)
	}

	var result embedResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai embed error: %s", result.Error.Message)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Data[0].Embedding, nil
}

// EmbedBatch generates embeddings for a batch of texts.
func (s *Service) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, t := range texts {
		emb, err := s.Embed(ctx, t)
		if err != nil {
			return nil, fmt.Errorf("embed index %d: %w", i, err)
		}
		results[i] = emb
	}
	return results, nil
}
