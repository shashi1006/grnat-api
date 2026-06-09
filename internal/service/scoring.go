package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	"github.com/readygeneration/readygeneration-backend/internal/scoring"
)

// ScoringService orchestrates compatibility scoring between orgs and grants.
type ScoringService struct {
	orgs    repository.OrganizationRepo
	grants  repository.GrantRepo
	scores  repository.ScoringRepo
	engine  *scoring.Engine
}

// NewScoringService creates a ScoringService.
func NewScoringService(
	orgs repository.OrganizationRepo,
	grants repository.GrantRepo,
	scores repository.ScoringRepo,
	engine *scoring.Engine,
) *ScoringService {
	return &ScoringService{orgs: orgs, grants: grants, scores: scores, engine: engine}
}

// ComputeScore calculates and persists the compatibility score for an org/grant pair.
func (s *ScoringService) ComputeScore(ctx context.Context, orgID, grantID uuid.UUID) (*domain.CompatibilityScore, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org: %w", err)
	}
	profile, err := s.orgs.GetProfile(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org profile: %w", err)
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
		return 0, fmt.Errorf("get org profile: %w", err)
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

	return computed, nil
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
