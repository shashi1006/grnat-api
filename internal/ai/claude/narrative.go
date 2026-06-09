package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
)

// NarrativeRequest bundles all context needed to generate a grant narrative section.
type NarrativeRequest struct {
	Section     domain.NarrativeSection
	Org         domain.Organization
	Profile     domain.OrganizationProfile
	Grant       domain.Grant
	Score       *domain.CompatibilityScore
	RAGContext  string // relevant NOFO passages retrieved via vector search
	WordTarget  int
	CustomNotes string
}

// NarrativeResult holds the generated text and metadata.
type NarrativeResult struct {
	Content   string
	WordCount int32
	TokensIn  int32
	TokensOut int32
	Model     string
}

// GenerateNarrative produces an AI narrative section for a grant application.
func (c *Client) GenerateNarrative(ctx context.Context, req NarrativeRequest) (*NarrativeResult, error) {
	system := buildSystemPrompt()
	user := buildUserPrompt(req)

	resp, err := c.Generate(ctx, GenerateRequest{
		SystemPrompt: system,
		UserPrompt:   user,
	})
	if err != nil {
		return nil, fmt.Errorf("generate narrative: %w", err)
	}

	words := int32(len(strings.Fields(resp.Content)))
	return &NarrativeResult{
		Content:   resp.Content,
		WordCount: words,
		TokensIn:  resp.TokensIn,
		TokensOut: resp.TokensOut,
		Model:     resp.Model,
	}, nil
}

func buildSystemPrompt() string {
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

func buildUserPrompt(req NarrativeRequest) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Generate a %s for a grant application.\n\n", sectionLabel(req.Section)))

	// Organization context
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

	// Grant context
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

	// RAG context from NOFO
	if req.RAGContext != "" {
		b.WriteString("\n## RELEVANT GRANT REQUIREMENTS (from NOFO)\n")
		b.WriteString(req.RAGContext)
		b.WriteString("\n")
	}

	// Scoring insights
	if req.Score != nil && !req.Score.Disqualified {
		if len(req.Score.Strengths) > 0 {
			b.WriteString("\n## ORGANIZATIONAL STRENGTHS FOR THIS GRANT\n")
			for _, s := range req.Score.Strengths {
				b.WriteString(fmt.Sprintf("- %s\n", s))
			}
		}
	}

	// Custom notes
	if req.CustomNotes != "" {
		b.WriteString("\n## ADDITIONAL NOTES FROM APPLICANT\n")
		b.WriteString(req.CustomNotes)
		b.WriteString("\n")
	}

	// Instructions
	wordTarget := req.WordTarget
	if wordTarget == 0 {
		wordTarget = defaultWordTarget(req.Section)
	}
	b.WriteString("\n## INSTRUCTIONS\n")
	b.WriteString(fmt.Sprintf("Write the %s in approximately %d words.\n", sectionLabel(req.Section), wordTarget))
	b.WriteString("Be specific to this organization and this grant opportunity.\n")
	b.WriteString("Focus on outcomes, community impact, and organizational qualifications.\n")
	b.WriteString("Do not include a title or header — begin directly with the narrative content.\n")

	return b.String()
}

func sectionLabel(s domain.NarrativeSection) string {
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

func defaultWordTarget(s domain.NarrativeSection) int {
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
