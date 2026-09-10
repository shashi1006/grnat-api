package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/ai/claude"
	"github.com/readygeneration/readygeneration-backend/internal/ai/openai"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/scoring"
)

// ScoringService orchestrates compatibility scoring between orgs and grants.
type ScoringService struct {
	orgs         repository.OrganizationRepo
	grants       repository.GrantRepo
	scores       repository.ScoringRepo
	products     repository.ProductRepo
	engine       *scoring.Engine
	claudeClient *claude.Client
	openAIClient *openai.Client
	grantSvc     *GrantService
}

// NewScoringService creates a ScoringService.
func NewScoringService(
	orgs repository.OrganizationRepo,
	grants repository.GrantRepo,
	scores repository.ScoringRepo,
	products repository.ProductRepo,
	engine *scoring.Engine,
	claudeClient *claude.Client,
	openAIClient *openai.Client,
	grantSvc *GrantService,
) *ScoringService {
	return &ScoringService{
		orgs:         orgs,
		grants:       grants,
		scores:       scores,
		products:     products,
		engine:       engine,
		claudeClient: claudeClient,
		openAIClient: openAIClient,
		grantSvc:     grantSvc,
	}
}

// ComputeScore calculates and persists the compatibility score for an org/grant pair.
func (s *ScoringService) ComputeScore(ctx context.Context, orgID, grantID uuid.UUID) (*domain.CompatibilityScore, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	profile, err := s.orgs.GetProfile(ctx, orgID)
	if err != nil {
		// Use a default empty profile if none exists yet
		profile = &domain.OrganizationProfile{OrgID: orgID}
	}
	grant, err := s.grants.GetByID(ctx, grantID)
	if err != nil {
		return nil, fmt.Errorf("get grant: %w", err)
	}

	result := s.engine.Compute(domain.ScoringInput{
		Org:     *org,
		Profile: *profile,
		Grant:   *grant,
	})

	saved, err := s.scores.Upsert(ctx, repository.UpsertScoreParams{
		OrgID:             orgID,
		GrantID:           grantID,
		TotalScore:        result.TotalScore,
		Tier:              result.Tier,
		DimensionScores:   result.DimensionScores,
		Disqualified:      result.Disqualified,
		DisqualifyReasons: result.DisqualifyReasons,
		Strengths:         result.Strengths,
		Gaps:              result.Gaps,
		Recommendations:   result.Recommendations,
		SemanticScore:     result.SemanticScore,
		EngineVersion:     "v1",
	})
	if err != nil {
		return nil, fmt.Errorf("persist score: %w", err)
	}
	return saved, nil
}

// ComputeAllGrantsForOrg scores an org against every active grant.
func (s *ScoringService) ComputeAllGrantsForOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("get org: %w", err)
	}
	profile, err := s.orgs.GetProfile(ctx, orgID)
	if err != nil {
		// Use a default empty profile if none exists yet
		profile = &domain.OrganizationProfile{OrgID: orgID}
	}

	// Load all active grants in pages
	var offset int32
	const pageSize = 100
	computed := 0

	for {
		grants, err := s.grants.List(ctx, []string{"active"}, pageSize, offset)
		if err != nil {
			return computed, fmt.Errorf("list grants: %w", err)
		}
		if len(grants) == 0 {
			break
		}

		for _, grant := range grants {
			result := s.engine.Compute(domain.ScoringInput{
				Org:     *org,
				Profile: *profile,
				Grant:   *grant,
			})
			_, err := s.scores.Upsert(ctx, repository.UpsertScoreParams{
				OrgID:             orgID,
				GrantID:           grant.ID,
				TotalScore:        result.TotalScore,
				Tier:              result.Tier,
				DimensionScores:   result.DimensionScores,
				Disqualified:      result.Disqualified,
				DisqualifyReasons: result.DisqualifyReasons,
				Strengths:         result.Strengths,
				Gaps:              result.Gaps,
				Recommendations:   result.Recommendations,
				SemanticScore:     result.SemanticScore,
				EngineVersion:     "v1",
			})
			if err != nil {
				return computed, fmt.Errorf("persist score for grant %s: %w", grant.ID, err)
			}
			computed++
		}

		offset += pageSize
		if int32(len(grants)) < pageSize {
			break
		}
	}

	// Auto-trigger LLM enrichment after deterministic scoring so that the
	// scores the user sees immediately after "Save & Score Grants" already
	// include the LLM alignment dimension. Failures are non-fatal — the
	// deterministic scores remain valid on their own.
	if s.claudeClient != nil || s.openAIClient != nil {
		if _, enrichErr := s.EnrichTopGrantsWithLLM(ctx, orgID, 50); enrichErr != nil {
			// Log but don't fail — deterministic scores are still valid
			fmt.Printf("[WARN] LLM enrichment after score-all failed for org %s: %v\n", orgID, enrichErr)
		}
	}

	return computed, nil
}

// EnrichTopGrantsResult reports the outcome of adding an LLM dimension to top grants.
type EnrichTopGrantsResult struct {
	Enriched int      `json:"enriched"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

// EnrichTopGrantsWithLLM adds an LLM alignment dimension to the top-scored grants for an org.
// It leaves disqualified grants unchanged and skips grants whose LLM call fails.
func (s *ScoringService) EnrichTopGrantsWithLLM(ctx context.Context, orgID uuid.UUID, limit int32) (*EnrichTopGrantsResult, error) {
	if limit <= 0 {
		limit = 20
	}

	top, err := s.scores.ListTopGrantsForOrg(ctx, orgID, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("list top grants: %w", err)
	}

	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	profile, err := s.orgs.GetProfile(ctx, orgID)
	if err != nil {
		// Use a default empty profile if none exists yet
		profile = &domain.OrganizationProfile{OrgID: orgID}
	}

	if s.claudeClient == nil && s.openAIClient == nil {
		return nil, fmt.Errorf("no LLM client configured (need ANTHROPIC_API_KEY or OPENAI_API_KEY)")
	}

	result := &EnrichTopGrantsResult{}
	for i := range top {
		sg := top[i]
		grant, err := s.grants.GetByID(ctx, sg.GrantID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("grant %s: %v", sg.GrantID, err))
			result.Skipped++
			continue
		}

		llmScore, rationale, err := s.scoreLLM(ctx, *org, *profile, *grant, &sg.CompatibilityScore, orgID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("grant %s: %v", grant.ID, err))
			result.Skipped++
			continue
		}

		updated := applyLLMToScore(&sg.CompatibilityScore, llmScore, rationale)
		_, err = s.scores.Upsert(ctx, repository.UpsertScoreParams{
			OrgID:             updated.OrgID,
			GrantID:           updated.GrantID,
			TotalScore:        updated.TotalScore,
			Tier:              updated.Tier,
			DimensionScores:   updated.DimensionScores,
			Disqualified:      updated.Disqualified,
			DisqualifyReasons: updated.DisqualifyReasons,
			Strengths:         updated.Strengths,
			Gaps:              updated.Gaps,
			Recommendations:   updated.Recommendations,
			SemanticScore:     updated.SemanticScore,
			EngineVersion:     updated.EngineVersion,
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("grant %s: %v", grant.ID, err))
			result.Skipped++
			continue
		}
		result.Enriched++
	}

	return result, nil
}

// scoreLLM calls an LLM (Claude preferred, OpenAI fallback) to produce a qualitative fit score.
func (s *ScoringService) scoreLLM(ctx context.Context, org domain.Organization, profile domain.OrganizationProfile, grant domain.Grant, score *domain.CompatibilityScore, orgID uuid.UUID) (float64, string, error) {
	ragQuery := buildFitRAGQuery(grant)
	ragContext := ""
	if s.grantSvc != nil {
		ragContext, _ = s.grantSvc.QueryNOFO(ctx, grant.ID, ragQuery, 3)
	}

	productCtx := s.buildProductContext(ctx, orgID)

	// Try Claude first
	if s.claudeClient != nil {
		fit, err := s.claudeClient.ScoreGrantFit(ctx, claude.GrantFitRequest{
			Org:        org,
			Profile:    profile,
			Grant:      grant,
			Score:      score,
			RAGContext: ragContext,
			Products:   productCtx,
		})
		if err == nil {
			return fit.Score, fit.Rationale, nil
		}
		// If Claude fails, fall through to OpenAI
		if s.openAIClient == nil {
			return 0, "", err
		}
	}

	// OpenAI fallback
	fit, err := s.openAIClient.ScoreGrantFit(ctx, openai.GrantFitRequest{
		Org:        org,
		Profile:    profile,
		Grant:      grant,
		Score:      score,
		RAGContext: ragContext,
		Products:   productCtx,
	})
	if err != nil {
		return 0, "", err
	}
	return fit.Score, fit.Rationale, nil
}

// buildProductContext fetches the org's product selections enriched with catalog
// details for inclusion in LLM scoring prompts.
func (s *ScoringService) buildProductContext(ctx context.Context, orgID uuid.UUID) []domain.ProductSelectionContext {
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
			UnitPrice:      fmt.Sprintf("%d.%02d", sel.UnitPriceCents/100, sel.UnitPriceCents%100),
			Subtotal:       fmt.Sprintf("%d.%02d", sel.SubtotalCents/100, sel.SubtotalCents%100),
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

// applyLLMToScore blends an LLM alignment dimension into an existing compatibility score.
// The LLM dimension is weighted at 10% and the remaining dimensions are rescaled to 90%.
func applyLLMToScore(score *domain.CompatibilityScore, llmScore float64, rationale string) *domain.CompatibilityScore {
	updated := *score

	nonLLM := make([]domain.DimensionScore, 0, len(score.DimensionScores))
	var ruleWeight, ruleWeightedSum float64
	for _, d := range score.DimensionScores {
		if d.Key == "llm_alignment" {
			continue
		}
		nonLLM = append(nonLLM, d)
		ruleWeight += d.Weight
		ruleWeightedSum += d.Score * d.Weight
	}

	if ruleWeight > 0 {
		scale := 90.0 / ruleWeight
		for i := range nonLLM {
			nonLLM[i].Weight = math.Round(nonLLM[i].Weight*scale*10) / 10
		}
		total := (ruleWeightedSum*scale + llmScore*10.0) / 100.0
		updated.TotalScore = math.Round(total*10) / 10
		updated.Tier = domain.ScoreTierFromScore(updated.TotalScore)
	}

	updated.DimensionScores = append(nonLLM, domain.DimensionScore{
		Key:         "llm_alignment",
		Score:       math.Round(llmScore*10) / 10,
		MaxScore:    100,
		Weight:      10,
		Explanation: rationale,
	})
	updated.EngineVersion = engineVersionWithLLM(updated.EngineVersion)
	updated.ComputedAt = time.Now()
	return &updated
}

func buildFitRAGQuery(grant domain.Grant) string {
	parts := []string{grant.Title, grant.FunderName}
	parts = append(parts, grant.FocusAreas...)
	return strings.Join(parts, " ")
}

func engineVersionWithLLM(v string) string {
	if strings.Contains(v, "llm") {
		return v
	}
	if v == "" {
		return "v1+llm"
	}
	return v + "+llm"
}

// GetScore retrieves a stored compatibility score.
func (s *ScoringService) GetScore(ctx context.Context, orgID, grantID uuid.UUID) (*domain.CompatibilityScore, error) {
	score, err := s.scores.Get(ctx, orgID, grantID)
	if err != nil {
		return nil, fmt.Errorf("score not found: %w", err)
	}
	return score, nil
}

// ListTopGrantsForOrg returns the top-scored grants for an org.
func (s *ScoringService) ListTopGrantsForOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]*repository.ScoredGrant, error) {
	return s.scores.ListTopGrantsForOrg(ctx, orgID, limit, offset)
}
