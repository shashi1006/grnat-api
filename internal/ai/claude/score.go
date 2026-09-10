package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
)

// GrantFitRequest holds the context needed to evaluate an org/grant fit.
type GrantFitRequest struct {
	Org        domain.Organization
	Profile    domain.OrganizationProfile
	Grant      domain.Grant
	Score      *domain.CompatibilityScore
	RAGContext string
	Products   []domain.ProductSelectionContext
}

// GrantFitResult holds the LLM-generated fit score and rationale.
type GrantFitResult struct {
	Score     float64
	Rationale string
	TokensIn  int32
	TokensOut int32
	Model     string
}

// ScoreGrantFit asks Claude to rate how well an organization fits a grant.
// It returns a 0-100 score and a short rationale.
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
	temp := 0.1
	maxTok := int64(256)

	resp, err := c.Generate(ctx, GenerateRequest{
		SystemPrompt: system,
		UserPrompt:   user,
		Temperature:  &temp,
		MaxTokens:    &maxTok,
	})
	if err != nil {
		return nil, fmt.Errorf("claude grant fit: %w", err)
	}

	raw := stripCodeFence(resp.Content)
	var parsed struct {
		Score     float64 `json:"score"`
		Rationale string  `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse grant fit response: %w (raw: %s)", err, truncate(raw, 200))
	}

	parsed.Score = math.Max(0, math.Min(100, parsed.Score))
	parsed.Rationale = strings.TrimSpace(parsed.Rationale)
	if parsed.Rationale == "" {
		parsed.Rationale = "No rationale provided."
	}

	return &GrantFitResult{
		Score:     math.Round(parsed.Score*10) / 10,
		Rationale: parsed.Rationale,
		TokensIn:  resp.TokensIn,
		TokensOut: resp.TokensOut,
		Model:     resp.Model,
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

	if len(req.Products) > 0 {
		b.WriteString("\n## SELECTED SOLUTIONS & PRODUCTS\n")
		b.WriteString("The organization plans to deploy the following preparedness solutions. " +
			"Consider how well these products align with the grant's goals and priorities.\n\n")
		for i, p := range req.Products {
			b.WriteString(fmt.Sprintf("%d. %s (Qty: %d, Unit Cost: $%s, Subtotal: $%s)\n", i+1, p.Name, p.Quantity, p.UnitPrice, p.Subtotal))
			if p.Description != "" {
				b.WriteString(fmt.Sprintf("   Description: %s\n", p.Description))
			}
			if p.Category != "" {
				b.WriteString(fmt.Sprintf("   Category: %s\n", p.Category))
			}
			if len(p.FundingAlignment) > 0 {
				b.WriteString(fmt.Sprintf("   Funding Alignment: %s\n", strings.Join(p.FundingAlignment, ", ")))
			}
		}
		b.WriteString("\n")
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
