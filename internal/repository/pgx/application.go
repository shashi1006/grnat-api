package pgxrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type applicationRepo struct {
	db *pgxpool.Pool
}

// NewApplicationRepo creates an ApplicationRepo backed by pgx.
func NewApplicationRepo(db *pgxpool.Pool) repository.ApplicationRepo {
	return &applicationRepo{db: db}
}

func (r *applicationRepo) Create(ctx context.Context, p repository.CreateApplicationParams) (*domain.GrantApplication, error) {
	var internalDeadline *time.Time
	if p.InternalDeadline != nil {
		t, err := time.Parse(time.RFC3339, *p.InternalDeadline)
		if err != nil {
			return nil, fmt.Errorf("parse internal_deadline: %w", err)
		}
		internalDeadline = &t
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO grant_applications
			(org_id, grant_id, assigned_to, status, stage, priority,
			 compatibility_score, notes, internal_deadline)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, org_id, grant_id, assigned_to, status, stage, priority,
		          compatibility_score, submission_date, award_amount, award_date,
		          rejection_reason, notes, internal_deadline, created_at, updated_at`,
		p.OrgID, p.GrantID, p.AssignedTo, p.Status, p.Stage, p.Priority,
		p.CompatibilityScore, p.Notes, internalDeadline,
	)
	return scanApplication(row)
}

func (r *applicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.GrantApplication, error) {
	row := r.db.QueryRow(ctx, `
		SELECT a.id, a.org_id, a.grant_id, a.assigned_to, a.status, a.stage, a.priority,
		       a.compatibility_score, a.submission_date, a.award_amount, a.award_date,
		       a.rejection_reason, a.notes, a.internal_deadline, a.created_at, a.updated_at,
		       g.title, g.funder_name, g.deadline
		FROM grant_applications a
		JOIN grants g ON g.id = a.grant_id
		WHERE a.id = $1`, id)
	return scanApplicationWithGrant(row)
}

func (r *applicationRepo) GetByOrgAndGrant(ctx context.Context, orgID, grantID uuid.UUID) (*domain.GrantApplication, error) {
	row := r.db.QueryRow(ctx, `
		SELECT a.id, a.org_id, a.grant_id, a.assigned_to, a.status, a.stage, a.priority,
		       a.compatibility_score, a.submission_date, a.award_amount, a.award_date,
		       a.rejection_reason, a.notes, a.internal_deadline, a.created_at, a.updated_at,
		       g.title, g.funder_name, g.deadline
		FROM grant_applications a
		JOIN grants g ON g.id = a.grant_id
		WHERE a.org_id = $1 AND a.grant_id = $2`, orgID, grantID)
	return scanApplicationWithGrant(row)
}

func (r *applicationRepo) ListForOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]*domain.GrantApplication, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.org_id, a.grant_id, a.assigned_to, a.status, a.stage, a.priority,
		       a.compatibility_score, a.submission_date, a.award_amount, a.award_date,
		       a.rejection_reason, a.notes, a.internal_deadline, a.created_at, a.updated_at,
		       g.title, g.funder_name, g.deadline
		FROM grant_applications a
		JOIN grants g ON g.id = a.grant_id
		WHERE a.org_id = $1
		ORDER BY a.updated_at DESC
		LIMIT $2 OFFSET $3`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.GrantApplication
	for rows.Next() {
		a, err := scanApplicationWithGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *applicationRepo) ListByStatus(ctx context.Context, status string, limit, offset int32) ([]*domain.GrantApplication, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.org_id, a.grant_id, a.assigned_to, a.status, a.stage, a.priority,
		       a.compatibility_score, a.submission_date, a.award_amount, a.award_date,
		       a.rejection_reason, a.notes, a.internal_deadline, a.created_at, a.updated_at,
		       g.title, g.funder_name, g.deadline
		FROM grant_applications a
		JOIN grants g ON g.id = a.grant_id
		WHERE a.status = $1
		ORDER BY a.updated_at DESC
		LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.GrantApplication
	for rows.Next() {
		a, err := scanApplicationWithGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *applicationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApplicationStatus, stage *domain.ApplicationStage) (*domain.GrantApplication, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE grant_applications
		SET status = $2, stage = COALESCE($3, stage), updated_at = NOW()
		WHERE id = $1
		RETURNING id, org_id, grant_id, assigned_to, status, stage, priority,
		          compatibility_score, submission_date, award_amount, award_date,
		          rejection_reason, notes, internal_deadline, created_at, updated_at`,
		id, status, stage)
	return scanApplication(row)
}

func (r *applicationRepo) Update(ctx context.Context, p repository.UpdateApplicationParams) (*domain.GrantApplication, error) {
	var submissionDate, internalDeadline *time.Time
	if p.SubmissionDate != nil {
		t, err := time.Parse(time.RFC3339, *p.SubmissionDate)
		if err != nil {
			return nil, fmt.Errorf("parse submission_date: %w", err)
		}
		submissionDate = &t
	}
	if p.InternalDeadline != nil {
		t, err := time.Parse(time.RFC3339, *p.InternalDeadline)
		if err != nil {
			return nil, fmt.Errorf("parse internal_deadline: %w", err)
		}
		internalDeadline = &t
	}

	row := r.db.QueryRow(ctx, `
		UPDATE grant_applications SET
			assigned_to       = COALESCE($2, assigned_to),
			status            = COALESCE($3, status),
			stage             = COALESCE($4, stage),
			priority          = COALESCE($5, priority),
			notes             = COALESCE($6, notes),
			internal_deadline = COALESCE($7, internal_deadline),
			submission_date   = COALESCE($8, submission_date),
			award_amount      = COALESCE($9, award_amount),
			updated_at        = NOW()
		WHERE id = $1
		RETURNING id, org_id, grant_id, assigned_to, status, stage, priority,
		          compatibility_score, submission_date, award_amount, award_date,
		          rejection_reason, notes, internal_deadline, created_at, updated_at`,
		p.ID, p.AssignedTo, p.Status, p.Stage, p.Priority,
		p.Notes, internalDeadline, submissionDate, p.AwardAmount,
	)
	return scanApplication(row)
}

func (r *applicationRepo) LogActivity(ctx context.Context, p repository.LogActivityParams) (*domain.ApplicationActivity, error) {
	var a domain.ApplicationActivity
	err := r.db.QueryRow(ctx, `
		INSERT INTO application_activities
			(application_id, user_id, activity_type, old_value, new_value, note)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, application_id, user_id, activity_type, old_value, new_value, note, created_at`,
		p.ApplicationID, p.UserID, p.ActivityType, p.OldValue, p.NewValue, p.Note,
	).Scan(&a.ID, &a.ApplicationID, &a.UserID, &a.ActivityType,
		&a.OldValue, &a.NewValue, &a.Note, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("log activity: %w", err)
	}
	return &a, nil
}

func (r *applicationRepo) ListActivities(ctx context.Context, applicationID uuid.UUID) ([]*domain.ApplicationActivity, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, application_id, user_id, activity_type, old_value, new_value, note, created_at
		FROM application_activities
		WHERE application_id = $1
		ORDER BY created_at DESC`, applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ApplicationActivity
	for rows.Next() {
		var a domain.ApplicationActivity
		if err := rows.Scan(&a.ID, &a.ApplicationID, &a.UserID, &a.ActivityType,
			&a.OldValue, &a.NewValue, &a.Note, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *applicationRepo) CreateNarrative(ctx context.Context, p repository.CreateNarrativeParams) (*domain.AINarrative, error) {
	var n domain.AINarrative
	err := r.db.QueryRow(ctx, `
		INSERT INTO ai_narratives
			(org_id, grant_id, application_id, section_key, prompt_used, content,
			 word_count, model_used, tokens_in, tokens_out, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			COALESCE((SELECT MAX(version)+1 FROM ai_narratives
			          WHERE org_id=$1 AND grant_id=$2 AND section_key=$4), 1))
		RETURNING id, org_id, grant_id, application_id, section_key, prompt_used,
		          content, word_count, model_used, tokens_in, tokens_out,
		          is_approved, version, created_at, updated_at`,
		p.OrgID, p.GrantID, p.ApplicationID, p.SectionKey, p.PromptUsed, p.Content,
		p.WordCount, p.ModelUsed, p.TokensIn, p.TokensOut,
	).Scan(&n.ID, &n.OrgID, &n.GrantID, &n.ApplicationID, &n.SectionKey,
		&n.PromptUsed, &n.Content, &n.WordCount, &n.ModelUsed,
		&n.TokensIn, &n.TokensOut, &n.IsApproved, &n.Version,
		&n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create narrative: %w", err)
	}
	return &n, nil
}

func (r *applicationRepo) ListNarrativesForApplication(ctx context.Context, applicationID uuid.UUID) ([]*domain.AINarrative, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, org_id, grant_id, application_id, section_key, prompt_used,
		       content, word_count, model_used, tokens_in, tokens_out,
		       is_approved, version, created_at, updated_at
		FROM ai_narratives
		WHERE application_id = $1
		ORDER BY section_key, version DESC`, applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNarratives(rows)
}

func (r *applicationRepo) GetLatestNarrativeBySection(ctx context.Context, applicationID uuid.UUID, section domain.NarrativeSection) (*domain.AINarrative, error) {
	var n domain.AINarrative
	err := r.db.QueryRow(ctx, `
		SELECT id, org_id, grant_id, application_id, section_key, prompt_used,
		       content, word_count, model_used, tokens_in, tokens_out,
		       is_approved, version, created_at, updated_at
		FROM ai_narratives
		WHERE application_id = $1 AND section_key = $2
		ORDER BY version DESC LIMIT 1`, applicationID, section,
	).Scan(&n.ID, &n.OrgID, &n.GrantID, &n.ApplicationID, &n.SectionKey,
		&n.PromptUsed, &n.Content, &n.WordCount, &n.ModelUsed,
		&n.TokensIn, &n.TokensOut, &n.IsApproved, &n.Version,
		&n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get narrative: %w", err)
	}
	return &n, nil
}

func (r *applicationRepo) Count(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM grant_applications WHERE org_id = $1`, orgID).Scan(&n)
	return n, err
}

// --- scan helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanApplication(row scanner) (*domain.GrantApplication, error) {
	var a domain.GrantApplication
	err := row.Scan(
		&a.ID, &a.OrgID, &a.GrantID, &a.AssignedTo,
		&a.Status, &a.Stage, &a.Priority,
		&a.CompatibilityScore, &a.SubmissionDate, &a.AwardAmount,
		&a.AwardDate, &a.RejectionReason, &a.Notes,
		&a.InternalDeadline, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan application: %w", err)
	}
	return &a, nil
}

func scanApplicationWithGrant(row scanner) (*domain.GrantApplication, error) {
	var a domain.GrantApplication
	err := row.Scan(
		&a.ID, &a.OrgID, &a.GrantID, &a.AssignedTo,
		&a.Status, &a.Stage, &a.Priority,
		&a.CompatibilityScore, &a.SubmissionDate, &a.AwardAmount,
		&a.AwardDate, &a.RejectionReason, &a.Notes,
		&a.InternalDeadline, &a.CreatedAt, &a.UpdatedAt,
		&a.GrantTitle, &a.FunderName, &a.GrantDeadline,
	)
	if err != nil {
		return nil, fmt.Errorf("scan application+grant: %w", err)
	}
	return &a, nil
}

func scanNarratives(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*domain.AINarrative, error) {
	var out []*domain.AINarrative
	for rows.Next() {
		var n domain.AINarrative
		if err := rows.Scan(
			&n.ID, &n.OrgID, &n.GrantID, &n.ApplicationID, &n.SectionKey,
			&n.PromptUsed, &n.Content, &n.WordCount, &n.ModelUsed,
			&n.TokensIn, &n.TokensOut, &n.IsApproved, &n.Version,
			&n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}
