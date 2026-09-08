package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
)

// NarrativeRequest mirrors claude.NarrativeRequest for OpenAI fallback.
type NarrativeRequest struct {
	Section     domain.NarrativeSection
	Org         domain.Organization
	Profile     domain.OrganizationProfile
	Grant       domain.Grant
	Score       *domain.CompatibilityScore
	RAGContext  string
	WordTarget  int
	CustomNotes string
}

// NarrativeResult mirrors claude.NarrativeResult.
type NarrativeResult struct {
	Content   string
	WordCount int32
	TokensIn  int32
	TokensOut int32
	Model     string
}

// GenerateNarrative produces an AI narrative section via OpenAI.
func (c *Client) GenerateNarrative(ctx context.Context, req NarrativeRequest) (*NarrativeResult, error) {
	system := buildNarrativeSystemPrompt()
	user := buildNarrativeUserPrompt(req)

	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.7,
		"max_tokens":  1024,
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

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	words := int32(len(strings.Fields(content)))

	return &NarrativeResult{
		Content:   content,
		WordCount: words,
		TokensIn:  int32(result.Usage.PromptTokens),
		TokensOut: int32(result.Usage.CompletionTokens),
		Model:     result.Model,
	}, nil
}

func buildNarrativeSystemPrompt() string {
	return `You are an expert grant writer with 20+ years of experience writing successful
federal, state, and foundation grant applications for nonprofits and community organizations.

Your writing is:
- Clear, compelling, and data-driven
- Responsive to the specific funder's priorities and evaluation criteria
- Free of jargon but professionally authoritative
- Structured with strong topic sentences and logical flow
- Specific — using numbers, outcomes, and evidence whenever available

You write in first person from the organization's perspective.
Do not include section titles or headers in your response — only the narrative content itself.
Do not fabricate statistics or data not provided in the context.`
}

func buildNarrativeUserPrompt(req NarrativeRequest) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Generate a %s for a grant application.\n\n", narrativeSectionLabel(req.Section)))

	b.WriteString("## ORGANIZATION PROFILE\n")
	b.WriteString(fmt.Sprintf("Name: %s\n", req.Org.Name))
	if req.Org.Mission != nil {
		b.WriteString(fmt.Sprintf("Mission: %s\n", *req.Org.Mission))
	}
	b.WriteString(fmt.Sprintf("Type: %s\n", req.Org.OrgType))
	if req.Org.State != nil {
		b.WriteString(fmt.Sprintf("Location: %s\n", *req.Org.State))
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
	if req.Profile.NumEmployees != nil {
		b.WriteString(fmt.Sprintf("Staff: %d employees\n", *req.Profile.NumEmployees))
	}
	if req.Profile.YearsOperating != nil {
		b.WriteString(fmt.Sprintf("Years Operating: %d\n", *req.Profile.YearsOperating))
	}

	b.WriteString("\n## GRANT OPPORTUNITY\n")
	b.WriteString(fmt.Sprintf("Title: %s\n", req.Grant.Title))
	b.WriteString(fmt.Sprintf("Funder: %s\n", req.Grant.FunderName))
	if req.Grant.Description != nil {
		b.WriteString(fmt.Sprintf("Description: %s\n", *req.Grant.Description))
	}
	if len(req.Grant.FocusAreas) > 0 {
		b.WriteString(fmt.Sprintf("Focus Areas: %s\n", strings.Join(req.Grant.FocusAreas, ", ")))
	}
	if req.Grant.MaxAwardAmount != nil {
		b.WriteString(fmt.Sprintf("Max Award: $%d\n", *req.Grant.MaxAwardAmount/100))
	}

	if req.RAGContext != "" {
		b.WriteString("\n## RELEVANT GRANT REQUIREMENTS (from NOFO)\n")
		b.WriteString(req.RAGContext)
		b.WriteString("\n")
	}

	if req.Score != nil && !req.Score.Disqualified {
		if len(req.Score.Strengths) > 0 {
			b.WriteString("\n## ORGANIZATIONAL STRENGTHS FOR THIS GRANT\n")
			for _, s := range req.Score.Strengths {
				b.WriteString(fmt.Sprintf("- %s\n", s))
			}
		}
	}

	if req.CustomNotes != "" {
		b.WriteString("\n## ADDITIONAL NOTES FROM APPLICANT\n")
		b.WriteString(req.CustomNotes)
		b.WriteString("\n")
	}

	wordTarget := req.WordTarget
	if wordTarget == 0 {
		wordTarget = defaultNarrativeWordTarget(req.Section)
	}
	b.WriteString("\n## INSTRUCTIONS\n")
	b.WriteString(fmt.Sprintf("Write the %s in approximately %d words.\n", narrativeSectionLabel(req.Section), wordTarget))
	b.WriteString("Be specific to this organization and this grant opportunity.\n")
	b.WriteString("Focus on outcomes, community impact, and organizational qualifications.\n")
	b.WriteString("Do not include a title or header — begin directly with the narrative content.\n")

	return b.String()
}

func narrativeSectionLabel(s domain.NarrativeSection) string {
	labels := map[domain.NarrativeSection]string{
		domain.SectionNeedStatement:      "Statement of Need",
		domain.SectionProjectDescription: "Project Description",
		domain.SectionGoalsObjectives:    "Goals and Objectives",
		domain.SectionEvalPlan:           "Evaluation Plan",
		domain.SectionOrgCapacity:        "Organizational Capacity Statement",
		domain.SectionBudgetNarrative:    "Budget Narrative",
		domain.SectionExecutiveSummary:   "Executive Summary",
	}
	if l, ok := labels[s]; ok {
		return l
	}
	return string(s)
}

func defaultNarrativeWordTarget(s domain.NarrativeSection) int {
	targets := map[domain.NarrativeSection]int{
		domain.SectionNeedStatement:      500,
		domain.SectionProjectDescription: 750,
		domain.SectionGoalsObjectives:    400,
		domain.SectionEvalPlan:           400,
		domain.SectionOrgCapacity:        350,
		domain.SectionBudgetNarrative:    300,
		domain.SectionExecutiveSummary:   250,
	}
	if t, ok := targets[s]; ok {
		return t
	}
	return 500
}
