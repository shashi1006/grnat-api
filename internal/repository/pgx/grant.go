package pgxrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type grantRepo struct {
	db *pgxpool.Pool
}

// NewGrantRepo creates a pgx-backed GrantRepo.
func NewGrantRepo(db *pgxpool.Pool) repository.GrantRepo {
	return &grantRepo{db: db}
}

func (r *grantRepo) Create(ctx context.Context, p repository.CreateGrantParams) (*domain.Grant, error) {
	const q = `
		INSERT INTO grants (
			slug, title, funder_name, funder_type, program_number, opportunity_number,
			agency, description, synopsis, category, focus_areas,
			eligible_org_types, eligible_populations, eligible_states,
			requires_501c3, requires_audited_fin, requires_match, match_percentage,
			min_award_amount, max_award_amount, total_funding_available,
			application_url, status, deadline, open_date,
			difficulty_level, competition_level, tags, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,
			$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29
		) RETURNING *`
	var deadline, openDate *time.Time
	if p.Deadline != nil {
		t, _ := time.Parse("2006-01-02", *p.Deadline)
		deadline = &t
	}
	if p.OpenDate != nil {
		t, _ := time.Parse("2006-01-02", *p.OpenDate)
		openDate = &t
	}
	row := r.db.QueryRow(ctx, q,
		p.Slug, p.Title, p.FunderName, string(p.FunderType), p.ProgramNumber,
		p.OpportunityNumber, p.Agency, p.Description, p.Synopsis, p.Category,
		p.FocusAreas, p.EligibleOrgTypes, p.EligiblePopulations, p.EligibleStates,
		p.Requires501c3, p.RequiresAuditedFin, p.RequiresMatch, p.MatchPercentage,
		p.MinAwardAmount, p.MaxAwardAmount, p.TotalFundingAvailable,
		p.ApplicationURL, string(p.Status), deadline, openDate,
		string(p.DifficultyLevel), string(p.CompetitionLevel), p.Tags, p.CreatedBy,
	)
	return scanGrant(row)
}

func (r *grantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Grant, error) {
	return scanGrant(r.db.QueryRow(ctx, `SELECT * FROM grants WHERE id=$1`, id))
}

func (r *grantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Grant, error) {
	return scanGrant(r.db.QueryRow(ctx, `SELECT * FROM grants WHERE slug=$1`, slug))
}

func (r *grantRepo) List(ctx context.Context, statuses []string, limit, offset int32) ([]*domain.Grant, error) {
	if len(statuses) == 0 {
		statuses = []string{"active"}
	}
	rows, err := r.db.Query(ctx,
		`SELECT * FROM grants WHERE status = ANY($1) ORDER BY deadline ASC NULLS LAST LIMIT $2 OFFSET $3`,
		statuses, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list grants: %w", err)
	}
	defer rows.Close()
	return scanGrants(rows)
}

func (r *grantRepo) ListByCategory(ctx context.Context, category string, limit, offset int32) ([]*domain.Grant, error) {
	rows, err := r.db.Query(ctx,
		`SELECT * FROM grants WHERE category=$1 AND status='active' ORDER BY deadline ASC NULLS LAST LIMIT $2 OFFSET $3`,
		category, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list grants by category: %w", err)
	}
	defer rows.Close()
	return scanGrants(rows)
}

func (r *grantRepo) Search(ctx context.Context, query string, limit, offset int32) ([]*domain.Grant, error) {
	rows, err := r.db.Query(ctx,
		`SELECT * FROM grants WHERE title ILIKE '%'||$1||'%' AND status='active' ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		query, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("search grants: %w", err)
	}
	defer rows.Close()
	return scanGrants(rows)
}

func (r *grantRepo) Update(ctx context.Context, p repository.UpdateGrantParams) (*domain.Grant, error) {
	const q = `
		UPDATE grants
		SET title       = COALESCE($2, title),
		    description = COALESCE($3, description),
		    synopsis    = COALESCE($4, synopsis),
		    status      = COALESCE($5, status),
		    updated_at  = NOW()
		WHERE id=$1 RETURNING *`
	return scanGrant(r.db.QueryRow(ctx, q, p.ID, p.Title, p.Description, p.Synopsis, p.Status))
}

func (r *grantRepo) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error {
	v := pgvector.NewVector(embedding)
	_, err := r.db.Exec(ctx, `UPDATE grants SET embedding=$2, updated_at=NOW() WHERE id=$1`, id, v)
	return err
}

func (r *grantRepo) UpdateNOFO(ctx context.Context, id uuid.UUID, text string) error {
	_, err := r.db.Exec(ctx, `UPDATE grants SET full_nofo_text=$2, updated_at=NOW() WHERE id=$1`, id, text)
	return err
}

func (r *grantRepo) Archive(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE grants SET status='archived', updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *grantRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM grants WHERE status != 'archived'`).Scan(&n)
	return n, err
}

func (r *grantRepo) SearchBySimilarity(ctx context.Context, embedding []float32, limit int32) ([]*repository.GrantWithDistance, error) {
	v := pgvector.NewVector(embedding)
	rows, err := r.db.Query(ctx,
		`SELECT *, (embedding <=> $1) AS distance FROM grants WHERE status='active' ORDER BY distance ASC LIMIT $2`,
		v, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("similarity search: %w", err)
	}
	defer rows.Close()
	var results []*repository.GrantWithDistance
	for rows.Next() {
		var g repository.GrantWithDistance
		grant, err := scanGrantWithExtra(rows, &g.Distance)
		if err != nil {
			return nil, err
		}
		g.Grant = *grant
		results = append(results, &g)
	}
	return results, rows.Err()
}

func (r *grantRepo) UpsertNOFOChunk(ctx context.Context, p repository.UpsertChunkParams) (*domain.NOFOChunk, error) {
	var emb *pgvector.Vector
	if len(p.Embedding) > 0 {
		v := pgvector.NewVector(p.Embedding)
		emb = &v
	}
	const q = `
		INSERT INTO nofo_chunks (grant_id, chunk_index, section, content, token_count, embedding)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (grant_id, chunk_index) DO UPDATE SET
			section = EXCLUDED.section, content = EXCLUDED.content,
			token_count = EXCLUDED.token_count, embedding = EXCLUDED.embedding
		RETURNING id, grant_id, chunk_index, section, content, token_count, created_at`
	var c domain.NOFOChunk
	err := r.db.QueryRow(ctx, q, p.GrantID, p.ChunkIndex, p.Section, p.Content, p.TokenCount, emb).
		Scan(&c.ID, &c.GrantID, &c.ChunkIndex, &c.Section, &c.Content, &c.TokenCount, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert nofo chunk: %w", err)
	}
	return &c, nil
}

func (r *grantRepo) ListNOFOChunks(ctx context.Context, grantID uuid.UUID) ([]*domain.NOFOChunk, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, grant_id, chunk_index, section, content, token_count, created_at FROM nofo_chunks WHERE grant_id=$1 ORDER BY chunk_index`,
		grantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list nofo chunks: %w", err)
	}
	defer rows.Close()
	var chunks []*domain.NOFOChunk
	for rows.Next() {
		var c domain.NOFOChunk
		if err := rows.Scan(&c.ID, &c.GrantID, &c.ChunkIndex, &c.Section, &c.Content, &c.TokenCount, &c.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, &c)
	}
	return chunks, rows.Err()
}

func (r *grantRepo) SearchChunksBySimilarity(ctx context.Context, grantID uuid.UUID, embedding []float32, limit int32) ([]*repository.ChunkWithDistance, error) {
	v := pgvector.NewVector(embedding)
	rows, err := r.db.Query(ctx,
		`SELECT id, grant_id, chunk_index, section, content, token_count, created_at, (embedding <=> $1) AS distance
		 FROM nofo_chunks WHERE grant_id=$2 ORDER BY distance ASC LIMIT $3`,
		v, grantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("chunk similarity search: %w", err)
	}
	defer rows.Close()
	var results []*repository.ChunkWithDistance
	for rows.Next() {
		var cwd repository.ChunkWithDistance
		err := rows.Scan(&cwd.ID, &cwd.GrantID, &cwd.ChunkIndex, &cwd.Section, &cwd.Content, &cwd.TokenCount, &cwd.CreatedAt, &cwd.Distance)
		if err != nil {
			return nil, err
		}
		results = append(results, &cwd)
	}
	return results, rows.Err()
}

func (r *grantRepo) UpsertScoringCriterion(ctx context.Context, p repository.UpsertCriterionParams) (*domain.GrantScoringCriterion, error) {
	const q = `
		INSERT INTO grant_scoring_criteria (grant_id, criterion_key, weight, is_required, disqualifying, description)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (grant_id, criterion_key) DO UPDATE SET
			weight=$3, is_required=$4, disqualifying=$5, description=$6
		RETURNING id, grant_id, criterion_key, weight, is_required, disqualifying, description, created_at`
	var c domain.GrantScoringCriterion
	err := r.db.QueryRow(ctx, q, p.GrantID, p.CriterionKey, p.Weight, p.IsRequired, p.Disqualifying, p.Description).
		Scan(&c.ID, &c.GrantID, &c.CriterionKey, &c.Weight, &c.IsRequired, &c.Disqualifying, &c.Description, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert criterion: %w", err)
	}
	return &c, nil
}

func (r *grantRepo) ListScoringCriteria(ctx context.Context, grantID uuid.UUID) ([]*domain.GrantScoringCriterion, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, grant_id, criterion_key, weight, is_required, disqualifying, description, created_at FROM grant_scoring_criteria WHERE grant_id=$1 ORDER BY criterion_key`,
		grantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var criteria []*domain.GrantScoringCriterion
	for rows.Next() {
		var c domain.GrantScoringCriterion
		if err := rows.Scan(&c.ID, &c.GrantID, &c.CriterionKey, &c.Weight, &c.IsRequired, &c.Disqualifying, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		criteria = append(criteria, &c)
	}
	return criteria, rows.Err()
}

// scanGrant scans a full grants row into domain.Grant.
func scanGrant(row scannable) (*domain.Grant, error) {
	return scanGrantWithExtra(row, nil)
}

func scanGrantWithExtra(row scannable, distance *float64) (*domain.Grant, error) {
	var g domain.Grant
	var funderType, status, diff, comp string
	var emb *pgvector.Vector
	dest := []any{
		&g.ID, &g.Slug, &g.Title, &g.FunderName, &funderType, &g.ProgramNumber, &g.OpportunityNumber,
		&g.Agency, &g.SubAgency, &g.Description, &g.Synopsis, &g.FullNOFOText,
		&g.Category, &g.Subcategory, &g.FocusAreas, &g.EligibleOrgTypes, &g.EligiblePopulations,
		&g.EligibleStates, &g.EligibleCounties, &g.Requires501c3, &g.RequiresAuditedFin,
		&g.RequiresIndirectRate, &g.RequiresMatch, &g.MatchPercentage,
		&g.MinAwardAmount, &g.MaxAwardAmount, &g.AvgAwardAmount, &g.TotalFundingAvailable,
		&g.NumAwardsExpected, &g.ApplicationURL, &g.FAQURL, &g.WebinarURL,
		&status, &g.Deadline, &g.OpenDate, &g.PeriodOfPerformance,
		&g.IsRecurring, &g.RecurrenceNotes, &diff, &comp,
		&g.Tags, &emb, &g.Metadata, &g.CreatedBy,
		&g.CreatedAt, &g.UpdatedAt,
	}
	if distance != nil {
		dest = append(dest, distance)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, fmt.Errorf("scan grant: %w", err)
	}
	g.FunderType = domain.FunderType(funderType)
	g.Status = domain.GrantStatus(status)
	g.DifficultyLevel = domain.DifficultyLevel(diff)
	g.CompetitionLevel = domain.CompetitionLevel(comp)
	return &g, nil
}

func scanGrants(rows interface{ Next() bool; Scan(...any) error; Err() error }) ([]*domain.Grant, error) {
	var grants []*domain.Grant
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}
