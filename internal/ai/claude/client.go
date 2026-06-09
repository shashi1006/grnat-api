// Package claude wraps the Anthropic Claude API for narrative generation
// and any other LLM tasks in the ReadyGeneration platform.
package claude

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/readygeneration/readygeneration-backend/internal/config"
)

// Client wraps the Anthropic SDK.
type Client struct {
	sdk       *anthropic.Client
	model     string
	maxTokens int64
}

// NewClient constructs a Claude client from config.
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.Claude.APIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required for Claude integration")
	}
	sdk := anthropic.NewClient(
		option.WithAPIKey(cfg.Claude.APIKey),
	)
	return &Client{
		sdk:       sdk,
		model:     cfg.Claude.DefaultModel,
		maxTokens: int64(cfg.Claude.MaxTokens),
	}, nil
}

// GenerateRequest is the input for text generation.
type GenerateRequest struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    *int64
	Temperature  *float64
}

// GenerateResponse is the output from text generation.
type GenerateResponse struct {
	Content   string
	TokensIn  int32
	TokensOut int32
	Model     string
	Latency   time.Duration
}

// Generate sends a message to Claude and returns the text response.
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	maxTok := c.maxTokens
	if req.MaxTokens != nil {
		maxTok = *req.MaxTokens
	}

	start := time.Now()

	params := anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.Model(c.model)),
		MaxTokens: anthropic.F(maxTok),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.UserPrompt)),
		}),
	}

	if req.SystemPrompt != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(req.SystemPrompt),
		})
	}

	msg, err := c.sdk.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("claude generate: %w", err)
	}

	var sb strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	return &GenerateResponse{
		Content:   sb.String(),
		TokensIn:  int32(msg.Usage.InputTokens),
		TokensOut: int32(msg.Usage.OutputTokens),
		Model:     string(msg.Model),
		Latency:   time.Since(start),
	}, nil
}

// Embed is a placeholder — Anthropic does not provide embeddings.
// Embedding should be routed to OpenAI text-embedding-3-small or similar.
// This method exists for interface completeness.
func (c *Client) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("claude does not provide embeddings — use an embedding service")
}
