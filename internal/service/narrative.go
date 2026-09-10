package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/ai/claude"
	"github.com/readygeneration/readygeneration-backend/internal/ai/openai"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// NarrativeService generates AI narrative sections for grant applications.
type NarrativeService struct {
	orgs         repository.OrganizationRepo
	grants       repository.GrantRepo
	scores       repository.ScoringRepo
	applications repository.ApplicationRepo
	products     repository.ProductRepo
	claudeClient *claude.Client
	openAIClient *openai.Client
	grantSvc     *GrantService
}

// NewNarrativeService creates a NarrativeService.
func NewNarrativeService(
	orgs repository.OrganizationRepo,
	grants repository.GrantRepo,
	scores repository.ScoringRepo,
	applications repository.ApplicationRepo,
	products repository.ProductRepo,
	claudeClient *claude.Client,
	openAIClient *openai.Client,
	grantSvc *GrantService,
) *NarrativeService {
	return &NarrativeService{
		orgs:         orgs,
		grants:       grants,
		scores:       scores,
		applications: applications,
		products:     products,
		claudeClient: claudeClient,
		openAIClient: openAIClient,
		grantSvc:     grantSvc,
	}
}

// GenerateRequest is the input for narrative generation.
type GenerateNarrativeRequest struct {
	OrgID         uuid.UUID
	GrantID       uuid.UUID
	ApplicationID *uuid.UUID
	Section       domain.NarrativeSection
	WordTarget    int
	CustomNotes   string
}

// GenerateNarrative generates and persists a narrative section.
func (s *NarrativeService) GenerateNarrative(ctx context.Context, req GenerateNarrativeRequest) (*domain.AINarrative, error) {
	org, err := s.orgs.GetByID(ctx, req.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	profile, err := s.orgs.GetProfile(ctx, req.OrgID)
	if err != nil {
		return nil, fmt.Errorf("get org profile: %w", err)
	}
	grant, err := s.grants.GetByID(ctx, req.GrantID)
	if err != nil {
		return nil, fmt.Errorf("get grant: %w", err)
	}

	// Get compatibility score if available
	var score *domain.CompatibilityScore
	if s, err := s.scores.Get(ctx, req.OrgID, req.GrantID); err == nil {
		score = s
	}

	// Retrieve RAG context for the section
	ragQuery := buildRAGQuery(req.Section, *grant)
	ragContext, _ := s.grantSvc.QueryNOFO(ctx, req.GrantID, ragQuery, 5)

	// Fetch the org's selected products to pass as context to the LLM
	productCtx := s.buildProductContext(ctx, req.OrgID)

	var content string
	var model string
	var wc, ti, to int32

	// Try Claude first
	if s.claudeClient != nil {
		result, err := s.claudeClient.GenerateNarrative(ctx, claude.NarrativeRequest{
			Section:     req.Section,
			Org:         *org,
			Profile:     *profile,
			Grant:       *grant,
			Score:       score,
			RAGContext:  ragContext,
			WordTarget:  req.WordTarget,
			CustomNotes: req.CustomNotes,
			Products:    productCtx,
		})
		if err == nil {
			content = result.Content
			model = result.Model
			wc = result.WordCount
			ti = result.TokensIn
			to = result.TokensOut
		} else if s.openAIClient == nil {
			return nil, fmt.Errorf("claude generate: %w", err)
		}
	}

	// Fallback to OpenAI if Claude was unavailable or failed
	if content == "" && s.openAIClient != nil {
		result, err := s.openAIClient.GenerateNarrative(ctx, openai.NarrativeRequest{
			Section:     req.Section,
			Org:         *org,
			Profile:     *profile,
			Grant:       *grant,
			Score:       score,
			RAGContext:  ragContext,
			WordTarget:  req.WordTarget,
			CustomNotes: req.CustomNotes,
			Products:    productCtx,
		})
		if err != nil {
			return nil, fmt.Errorf("openai generate: %w", err)
		}
		content = result.Content
		model = result.Model
		wc = result.WordCount
		ti = result.TokensIn
		to = result.TokensOut
	}

	if content == "" {
		return nil, fmt.Errorf("no LLM client configured")
	}

	narrative, err := s.applications.CreateNarrative(ctx, repository.CreateNarrativeParams{
		OrgID:         req.OrgID,
		GrantID:       req.GrantID,
		ApplicationID: req.ApplicationID,
		SectionKey:    req.Section,
		Content:       content,
		WordCount:     &wc,
		ModelUsed:     model,
		TokensIn:      &ti,
		TokensOut:     &to,
	})
	if err != nil {
		return nil, fmt.Errorf("persist narrative: %w", err)
	}
	return narrative, nil
}

// buildRAGQuery constructs the vector search query based on the narrative section.
func buildRAGQuery(section domain.NarrativeSection, grant domain.Grant) string {
	sectionQueries := map[domain.NarrativeSection]string{
		domain.SectionNeedStatement:      "community need problem statement target population statistics",
		domain.SectionProjectDescription: "project activities implementation plan services",
		domain.SectionGoalsObjectives:    "goals objectives outcomes measurable targets",
		domain.SectionEvalPlan:           "evaluation criteria performance measures reporting",
		domain.SectionOrgCapacity:        "organization capacity qualifications experience staff",
		domain.SectionBudgetNarrative:    "budget allowable costs match requirements",
		domain.SectionExecutiveSummary:   "project overview summary purpose goals",
	}
	base := sectionQueries[section]
	if base == "" {
		base = string(section)
	}
	parts := []string{base}
	if len(grant.FocusAreas) > 0 {
		parts = append(parts, strings.Join(grant.FocusAreas, " "))
	}
	return strings.Join(parts, " ")
}

// buildProductContext fetches the org's product selections, enriches them with
// catalog details (name, description, category, funding alignment), and returns
// a slice suitable for inclusion in LLM prompts.
func (s *NarrativeService) buildProductContext(ctx context.Context, orgID uuid.UUID) []domain.ProductSelectionContext {
	if s.products == nil {
		return nil
	}
	selections, err := s.products.ListSelections(ctx, orgID)
	if err != nil || len(selections) == 0 {
		return nil
	}

	out := make([]domain.ProductSelectionContext, 0, len(selections))
	for _, sel := range selections {
		product, err := s.products.GetByID(ctx, sel.ProductID)
		if err != nil || product == nil {
			continue
		}
		p := domain.ProductSelectionContext{
			Name:           product.Name,
			Quantity:       sel.Quantity,
			UnitPrice:      formatCents(sel.UnitPriceCents),
			Subtotal:       formatCents(sel.SubtotalCents),
			SelectedAddons: sel.SelectedAddons,
		}
		if product.Description != nil {
			p.Description = *product.Description
		} else if product.ShortDesc != nil {
			p.Description = *product.ShortDesc
		}
		if product.Category != nil {
			p.Category = *product.Category
		}
		if len(product.FundingAlignment) > 0 {
			p.FundingAlignment = product.FundingAlignment
		}
		out = append(out, p)
	}
	return out
}

// formatCents converts a cents amount to a USD string (e.g., 149900 -> "1,499.00").
func formatCents(cents int64) string {
	whole := cents / 100
	frac := cents % 100
	return fmt.Sprintf("%d.%02d", whole, frac)
}
