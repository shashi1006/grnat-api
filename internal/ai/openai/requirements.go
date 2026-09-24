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

// ExtractRequirements asks OpenAI to pull applicant eligibility, submission
// pathway, and program-specific requirements out of raw NOFO text. Mirrors the
// Claude extraction used as the primary path.
func (c *Client) ExtractRequirements(ctx context.Context, grantTitle, nofoText string) (*domain.ExtractedRequirements, error) {
	system := `You are an expert grants analyst extracting structured requirements from a Notice of Funding Opportunity (NOFO).

Respond ONLY with a single JSON object in this exact format:
{
  "eligible_applicants": ["<entity types that may SUBMIT the application, snake_case, e.g. state-administrative-agency, municipality-government, nonprofit-community, higher-ed, k12-schools, hospitals-health-systems, public-safety, tribal>"],
  "submission_pathway": "<direct | pass_through | mixed>",
  "pass_through_note": "<one sentence explaining who must submit and how subawards flow, or empty string if direct>",
  "required_forms": ["<named forms/documents required, e.g. SF-424, Investment Justification>"],
  "narrative_sections": ["<named narrative sections the application requires>"],
  "set_asides": ["<mandatory funding set-asides or minimums, e.g. 35% LETPA minimum>"],
  "unallowable_costs": ["<program-specific unallowable costs/activities or excluded purchases, e.g. target hardening equipment, armed security personnel, food costs>"],
  "certifications": ["<required certifications or memberships, e.g. NIMS implementation, EMAC membership>"],
  "award_constraints": "<one sentence describing award caps, allocation structure, and cost share or empty string>",
  "eligible_org_types": ["<org categories that may receive funds including subrecipients, using the same snake_case labels>"],
  "eligible_states": ["<two-letter state codes if geographically restricted, else empty>"],
  "min_award_dollars": <number or null>,
  "max_award_dollars": <number or null>
}

Rules:
- submission_pathway = "pass_through" when only an administering agency (state agency, SAA, health department, etc.) may submit and other orgs participate as subrecipients. = "mixed" when both a pass-through track and a direct track exist. = "direct" otherwise.
- eligible_applicants MUST reflect who may legally submit, NOT who may benefit. If the NOFO says only the SAA may submit, eligible_applicants is ["state-administrative-agency"] even though universities or nonprofits can receive subawards.
- Only include award dollar figures that appear verbatim in the NOFO. If no per-applicant cap exists, use null — do not invent one.
- Be exhaustive for required_forms, set_asides, unallowable_costs, and certifications — these drive compliance checking.`

	user := fmt.Sprintf("## GRANT\nTitle: %s\n\n## NOFO TEXT\n%s", grantTitle, truncate(nofoText, 45000))

	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.1,
		"max_tokens":  2048,
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
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in openai response")
	}

	content := stripCodeFence(strings.TrimSpace(result.Choices[0].Message.Content))
	var parsed domain.ExtractedRequirements
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("parse requirements response: %w (raw: %s)", err, truncate(content, 200))
	}

	switch parsed.SubmissionPathway {
	case "direct", "pass_through", "mixed":
	default:
		parsed.SubmissionPathway = "direct"
	}
	parsed.PassThroughNote = strings.TrimSpace(parsed.PassThroughNote)

	return &parsed, nil
}
