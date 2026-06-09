package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type scoringRepo struct {
	db *pgxpool.Pool
}

// NewScoringRepo creates a pgx-backed ScoringRepo.
func NewScoringRepo(db *pgxpool.Pool) repository.ScoringRepo {
	return &scoringRepo{db: db}
}

func (r *scoringRepo) Upsert(ctx context.Context, p repository.UpsertScoreParams) (*domain.CompatibilityScore, error) {
	dimJSON, err := json.Marshal(p.DimensionScores)
	if err != nil {
		return nil, fmt.Errorf("marshal dimension scores: %w", err)
	}

	const q = `
		INSERT INTO compatibility_scores (
			org_id, grant_id, total_score, tier, dimension_scores,
			disqualified, disqualify_reasons, strengths, gaps, recommendations,
			semantic_score, engine_version
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (org_id, grant_id) DO UPDATE SET
			total_score        = EXCLUDED.total_score,
			tier               = EXCLUDED.tier,
			dimension_scores   = EXCLUDED.dimension_scores,
			disqualified       = EXCLUDED.disqualified,
			disqualify_reasons = EXCLUDED.disqualify_reasons,
			strengths          = EXCLUDED.strengths,
			gaps               = EXCLUDED.gaps,
			recommendations    = EXCLUDED.recommendations,
			semantic_score     = EXCLUDED.semantic_score,
			engine_version     = EXCLUDED.engine_version,
			computed_at        = NOW(),
			updated_at         = NOW()
		RETURNING id, org_id, grant_id, total_score, tier, dimension_scores,
		          disqualified, disqualify_reasons, strengths, gaps, recommendations,
		          semantic_score, engine_version, computed_at, created_at, updated_at`

	row := r.db.QueryRow(ctx, q,
		p.OrgID, p.GrantID, p.TotalScore, string(p.Tier), dimJSON,
		p.Disqualified, p.DisqualifyReasons, p.Strengths, p.Gaps, p.Recommendations,
		p.SemanticScore, p.EngineVersion,
	)
	return scanScore(row)
}

func (r *scoringRepo) Get(ctx context.Context, orgID, grantID uuid.UUID) (*domain.CompatibilityScore, error) {
	const q = `SELECT id, org_id, grant_id, total_score, tier, dimension_scores,
	                  disqualified, disqualify_reasons, strengths, gaps, recommendations,
	                  semantic_score, engine_version, computed_at, created_at, updated_at
	           FROM compatibility_scores WHERE org_id=$1 AND grant_id=$2`
	return scanScore(r.db.QueryRow(ctx, q, orgID, grantID))
}

func (r *scoringRepo) ListTopGrantsForOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]*repository.ScoredGrant, error) {
	const q = `
		SELECT cs.id, cs.org_id, cs.grant_id, cs.total_score, cs.tier, cs.dimension_scores,
		       cs.disqualified, cs.disqualify_reasons, cs.strengths, cs.gaps, cs.recommendations,
		       cs.semantic_score, cs.engine_version, cs.computed_at, cs.created_at, cs.updated_at,
		       g.title, g.funder_name, g.category, g.deadline, g.min_award_amount, g.max_award_amount, g.status
		FROM compatibility_scores cs
		JOIN grants g ON g.id = cs.grant_id
		WHERE cs.org_id=$1 AND cs.disqualified=FALSE AND g.status='active'
		ORDER BY cs.total_score DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, q, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list top grants: %w", err)
	}
	defer rows.Close()

	var results []*repository.ScoredGrant
	for rows.Next() {
		var sg repository.ScoredGrant
		score, err := scanScoreWithExtra(rows, &sg.GrantTitle, &sg.FunderName, &sg.Category, &sg.Deadline, &sg.MinAward, &sg.MaxAward, &sg.GrantStatus)
		if err != nil {
			return nil, err
		}
		sg.CompatibilityScore = *score
		results = append(results, &sg)
	}
	return results, rows.Err()
}

func (r *scoringRepo) ListOrgsForGrant(ctx context.Context, grantID uuid.UUID, limit, offset int32) ([]*repository.ScoredOrg, error) {
	const q = `
		SELECT cs.id, cs.org_id, cs.grant_id, cs.total_score, cs.tier, cs.dimension_scores,
		       cs.disqualified, cs.disqualify_reasons, cs.strengths, cs.gaps, cs.recommendations,
		       cs.semantic_score, cs.engine_version, cs.computed_at, cs.created_at, cs.updated_at,
		       o.name, o.state, o.org_type
		FROM compatibility_scores cs
		JOIN organizations o ON o.id = cs.org_id
		WHERE cs.grant_id=$1 AND cs.disqualified=FALSE
		ORDER BY cs.total_score DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, q, grantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orgs for grant: %w", err)
	}
	defer rows.Close()

	var results []*repository.ScoredOrg
	for rows.Next() {
		var so repository.ScoredOrg
		var orgType string
		score, err := scanScoreWithExtra(rows, &so.OrgName, &so.State, &orgType)
		if err != nil {
			return nil, err
		}
		so.CompatibilityScore = *score
		so.OrgType = orgType
		results = append(results, &so)
	}
	return results, rows.Err()
}

func (r *scoringRepo) DeleteForOrg(ctx context.Context, orgID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM compatibility_scores WHERE org_id=$1`, orgID)
	return err
}

func (r *scoringRepo) DeleteForGrant(ctx context.Context, grantID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM compatibility_scores WHERE grant_id=$1`, grantID)
	return err
}

func scanScore(row scannable) (*domain.CompatibilityScore, error) {
	return scanScoreWithExtra(row)
}

func scanScoreWithExtra(row scannable, extra ...any) (*domain.CompatibilityScore, error) {
	var s domain.CompatibilityScore
	var tierStr string
	var dimJSON []byte

	dest := []any{
		&s.ID, &s.OrgID, &s.GrantID, &s.TotalScore, &tierStr, &dimJSON,
		&s.Disqualified, &s.DisqualifyReasons, &s.Strengths, &s.Gaps, &s.Recommendations,
		&s.SemanticScore, &s.EngineVersion, &s.ComputedAt, &s.CreatedAt, &s.UpdatedAt,
	}
	dest = append(dest, extra...)

	if err := row.Scan(dest...); err != nil {
		return nil, fmt.Errorf("scan score: %w", err)
	}
	s.Tier = domain.CompatibilityTier(tierStr)
	if len(dimJSON) > 0 {
		if err := json.Unmarshal(dimJSON, &s.DimensionScores); err != nil {
			return nil, fmt.Errorf("unmarshal dimension scores: %w", err)
		}
	}
	return &s, nil
}
