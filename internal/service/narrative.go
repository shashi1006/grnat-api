package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/ai/claude"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// NarrativeService generates AI narrative sections for grant applications.
type NarrativeService struct {
	orgs         repository.OrganizationRepo
	grants       repository.GrantRepo
	scores       repository.ScoringRepo
	applications repository.ApplicationRepo
	claudeClient *claude.Client
	grantSvc     *GrantService
}

// NewNarrativeService creates a NarrativeService.
func NewNarrativeService(
	orgs repository.OrganizationRepo,
	grants repository.GrantRepo,
	scores repository.ScoringRepo,
	applications repository.ApplicationRepo,
	claudeClient *claude.Client,
	grantSvc *GrantService,
) *NarrativeService {
	return &NarrativeService{
		orgs:         orgs,
		grants:       grants,
		scores:       scores,
		applications: applications,
		claudeClient: claudeClient,
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

	result, err := s.claudeClient.GenerateNarrative(ctx, claude.NarrativeRequest{
		Section:     req.Section,
		Org:         *org,
		Profile:     *profile,
		Grant:       *grant,
		Score:       score,
		RAGContext:  ragContext,
		WordTarget:  req.WordTarget,
		CustomNotes: req.CustomNotes,
	})
	if err != nil {
		return nil, fmt.Errorf("claude generate: %w", err)
	}

	wc := result.WordCount
	ti := result.TokensIn
	to := result.TokensOut
	narrative, err := s.applications.CreateNarrative(ctx, repository.CreateNarrativeParams{
		OrgID:         req.OrgID,
		GrantID:       req.GrantID,
		ApplicationID: req.ApplicationID,
		SectionKey:    req.Section,
		Content:       result.Content,
		WordCount:     &wc,
		ModelUsed:     result.Model,
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
