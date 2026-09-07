// Package openai provides LLM scoring via OpenAI's chat completions API,
// used as a fallback when Anthropic Claude is unavailable.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
)

const (
	defaultChatModel = "gpt-4o-mini"
	chatURL          = "https://api.openai.com/v1/chat/completions"
)

// Client wraps the OpenAI chat completions API for grant-fit scoring.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates an OpenAI scoring client.
func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = defaultChatModel
	}
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// GrantFitRequest holds the context needed to evaluate an org/grant fit.
type GrantFitRequest struct {
	Org        domain.Organization
	Profile    domain.OrganizationProfile
	Grant      domain.Grant
	Score      *domain.CompatibilityScore
	RAGContext string
}

// GrantFitResult holds the LLM-generated fit score and rationale.
type GrantFitResult struct {
	Score     float64
	Rationale string
	TokensIn  int32
	TokensOut int32
	Model     string
}

// ScoreGrantFit asks OpenAI to rate how well an organization fits a grant.
func (c *Client) ScoreGrantFit(ctx context.Context, req GrantFitRequest) (*GrantFitResult, error) {
	system := `You are an expert grant program officer evaluating how well an organization fits a specific funding opportunity.

Respond ONLY with a single JSON object in this exact format:
{"score": <number 0-100>, "rationale": "<one to two sentences explaining the fit>"}

Scoring guidelines:
- 90-100: Exceptional fit; the organization strongly matches every major eligibility and strategic requirement.
- 70-89: Good fit; most criteria align and the org can credibly compete.
- 50-69: Partial fit; some alignment but notable gaps or weak evidence.
- 25-49: Poor fit; major mismatches in eligibility, mission, or capacity.
- 0-24: Very unlikely; hard disqualifiers or mission/location/capacity mismatch.

Be concise and evidence-based. Do not invent facts not present in the context. If key data is missing, lower the score accordingly and explain what is missing.`

	user := buildGrantFitPrompt(req)

	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.1,
		"max_tokens":  256,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call openai: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai error (status %d): %s", resp.StatusCode, truncate(string(raw), 300))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in openai response")
	}

	content := stripCodeFence(strings.TrimSpace(result.Choices[0].Message.Content))
	var parsed struct {
		Score     float64 `json:"score"`
		Rationale string  `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("parse grant fit response: %w (raw: %s)", err, truncate(content, 200))
	}

	parsed.Score = math.Max(0, math.Min(100, parsed.Score))
	parsed.Rationale = strings.TrimSpace(parsed.Rationale)
	if parsed.Rationale == "" {
		parsed.Rationale = "No rationale provided."
	}

	return &GrantFitResult{
		Score:     math.Round(parsed.Score*10) / 10,
		Rationale: parsed.Rationale,
		TokensIn:  int32(result.Usage.PromptTokens),
		TokensOut: int32(result.Usage.CompletionTokens),
		Model:     result.Model,
	}, nil
}

func buildGrantFitPrompt(req GrantFitRequest) string {
	var b strings.Builder

	b.WriteString("## ORGANIZATION\n")
	b.WriteString(fmt.Sprintf("Name: %s\n", req.Org.Name))
	b.WriteString(fmt.Sprintf("Type: %s\n", req.Org.OrgType))
	if req.Org.State != nil {
		b.WriteString(fmt.Sprintf("State: %s\n", *req.Org.State))
	}
	if req.Org.Mission != nil {
		b.WriteString(fmt.Sprintf("Mission: %s\n", *req.Org.Mission))
	}
	if req.Profile.Narrative != nil {
		b.WriteString(fmt.Sprintf("About: %s\n", *req.Profile.Narrative))
	}
	if len(req.Profile.PopulationsServed) > 0 {
		b.WriteString(fmt.Sprintf("Populations Served: %s\n", strings.Join(req.Profile.PopulationsServed, ", ")))
	}
	if len(req.Profile.ProgramAreas) > 0 {
		b.WriteString(fmt.Sprintf("Program Areas: %s\n", strings.Join(req.Profile.ProgramAreas, ", ")))
	}
	if len(req.Profile.FocusIssues) > 0 {
		b.WriteString(fmt.Sprintf("Focus Issues: %s\n", strings.Join(req.Profile.FocusIssues, ", ")))
	}
	if req.Profile.NumEmployees != nil {
		b.WriteString(fmt.Sprintf("Staff: %d employees\n", *req.Profile.NumEmployees))
	}
	if req.Profile.YearsOperating != nil {
		b.WriteString(fmt.Sprintf("Years Operating: %d\n", *req.Profile.YearsOperating))
	}
	b.WriteString(fmt.Sprintf("Has 501(c)(3): %v\n", req.Profile.Has501c3))
	b.WriteString(fmt.Sprintf("Has Audited Financials: %v\n", req.Profile.HasAuditedFinancials))
	b.WriteString(fmt.Sprintf("Prior Federal Grants: %v\n", req.Profile.PriorFederalGrants))

	b.WriteString("\n## GRANT OPPORTUNITY\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", req.Grant.Title))
	b.WriteString(fmt.Sprintf("Funder: %s\n", req.Grant.FunderName))
	if req.Grant.Description != nil {
		b.WriteString(fmt.Sprintf("Description: %s\n", *req.Grant.Description))
	}
	if len(req.Grant.FocusAreas) > 0 {
		b.WriteString(fmt.Sprintf("Focus Areas: %s\n", strings.Join(req.Grant.FocusAreas, ", ")))
	}
	if len(req.Grant.EligibleOrgTypes) > 0 {
		b.WriteString(fmt.Sprintf("Eligible Org Types: %s\n", strings.Join(req.Grant.EligibleOrgTypes, ", ")))
	}
	if len(req.Grant.EligiblePopulations) > 0 {
		b.WriteString(fmt.Sprintf("Eligible Populations: %s\n", strings.Join(req.Grant.EligiblePopulations, ", ")))
	}
	if len(req.Grant.EligibleStates) > 0 {
		b.WriteString(fmt.Sprintf("Eligible States: %s\n", strings.Join(req.Grant.EligibleStates, ", ")))
	}
	b.WriteString(fmt.Sprintf("Requires 501(c)(3): %v\n", req.Grant.Requires501c3))
	b.WriteString(fmt.Sprintf("Requires Audited Financials: %v\n", req.Grant.RequiresAuditedFin))
	b.WriteString(fmt.Sprintf("Requires Match: %v\n", req.Grant.RequiresMatch))
	if req.Grant.MinAwardAmount != nil {
		b.WriteString(fmt.Sprintf("Min Award: $%d\n", *req.Grant.MinAwardAmount/100))
	}
	if req.Grant.MaxAwardAmount != nil {
		b.WriteString(fmt.Sprintf("Max Award: $%d\n", *req.Grant.MaxAwardAmount/100))
	}

	if req.Score != nil && !req.Score.Disqualified {
		b.WriteString(fmt.Sprintf("\n## RULE-BASED COMPATIBILITY SCORE\nTotal: %.1f\nTier: %s\n", req.Score.TotalScore, req.Score.Tier))
		if len(req.Score.Strengths) > 0 {
			b.WriteString("Strengths: " + strings.Join(req.Score.Strengths, "; ") + "\n")
		}
		if len(req.Score.Gaps) > 0 {
			b.WriteString("Gaps: " + strings.Join(req.Score.Gaps, "; ") + "\n")
		}
	}

	if req.RAGContext != "" {
		b.WriteString("\n## RELEVANT NOFO PASSAGES\n")
		b.WriteString(req.RAGContext)
		b.WriteString("\n")
	}

	b.WriteString("\n## TASK\nEvaluate the overall fit between this organization and this grant.")
	return b.String()
}

func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		} else {
			s = strings.TrimPrefix(s, "```")
		}
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
