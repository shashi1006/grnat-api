package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractedRequirements is the structured requirement set pulled from a NOFO.
// EligibleApplicants lists the entity types that may SUBMIT an application
// (snake_case labels, e.g. "state-administrative-agency"), as distinct from
// orgs that may only benefit via subawards.
type ExtractedRequirements struct {
	EligibleApplicants  []string `json:"eligible_applicants"`
	SubmissionPathway   string   `json:"submission_pathway"` // direct | pass_through | mixed
	PassThroughNote     string   `json:"pass_through_note"`
	RequiredForms       []string `json:"required_forms"`
	NarrativeSections   []string `json:"narrative_sections"`
	SetAsides           []string `json:"set_asides"`
	Certifications      []string `json:"certifications"`
	AwardConstraints    string   `json:"award_constraints"`
	EligibleOrgTypes    []string `json:"eligible_org_types"`
	EligibleStates      []string `json:"eligible_states"`
	MinAwardDollars     *int64   `json:"min_award_dollars"`
	MaxAwardDollars     *int64   `json:"max_award_dollars"`
}

// SubmissionRequirementsMap returns the fields that belong in
// grants.submission_requirements (everything except the top-level columns).
func (r *ExtractedRequirements) SubmissionRequirementsMap() map[string]interface{} {
	return map[string]interface{}{
		"required_forms":     r.RequiredForms,
		"narrative_sections": r.NarrativeSections,
		"set_asides":         r.SetAsides,
		"certifications":     r.Certifications,
		"award_constraints":  r.AwardConstraints,
	}
}

// ExtractRequirements asks Claude to pull applicant eligibility, submission
// pathway, and program-specific requirements out of raw NOFO text.
func (c *Client) ExtractRequirements(ctx context.Context, grantTitle, nofoText string) (*ExtractedRequirements, error) {
	system := `You are an expert grants analyst extracting structured requirements from a Notice of Funding Opportunity (NOFO).

Respond ONLY with a single JSON object in this exact format:
{
  "eligible_applicants": ["<entity types that may SUBMIT the application, snake_case, e.g. state-administrative-agency, municipality-government, nonprofit-community, higher-ed, k12-schools, hospitals-health-systems, public-safety, tribal>"],
  "submission_pathway": "<direct | pass_through | mixed>",
  "pass_through_note": "<one sentence explaining who must submit and how subawards flow, or empty string if direct>",
  "required_forms": ["<named forms/documents required, e.g. SF-424, Investment Justification>"],
  "narrative_sections": ["<named narrative sections the application requires>"],
  "set_asides": ["<mandatory funding set-asides or minimums, e.g. 35% LETPA minimum>"],
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
- Be exhaustive for required_forms, set_asides, and certifications — these drive compliance checking.`

	user := fmt.Sprintf("## GRANT\nTitle: %s\n\n## NOFO TEXT\n%s", grantTitle, truncate(nofoText, 45000))
	temp := 0.1
	maxTok := int64(2048)

	resp, err := c.Generate(ctx, GenerateRequest{
		SystemPrompt: system,
		UserPrompt:   user,
		Temperature:  &temp,
		MaxTokens:    &maxTok,
	})
	if err != nil {
		return nil, fmt.Errorf("claude extract requirements: %w", err)
	}

	raw := stripCodeFence(resp.Content)
	var parsed ExtractedRequirements
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse requirements response: %w (raw: %s)", err, truncate(raw, 200))
	}

	switch parsed.SubmissionPathway {
	case "direct", "pass_through", "mixed":
	default:
		parsed.SubmissionPathway = "direct"
	}
	parsed.PassThroughNote = strings.TrimSpace(parsed.PassThroughNote)

	return &parsed, nil
}
