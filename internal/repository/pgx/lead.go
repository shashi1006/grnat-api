package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type leadRepo struct {
	db *pgxpool.Pool
}

// NewLeadRepo creates a LeadRepo backed by pgx.
func NewLeadRepo(db *pgxpool.Pool) repository.LeadRepo {
	return &leadRepo{db: db}
}

func (r *leadRepo) Create(ctx context.Context, p repository.CreateLeadParams) (*domain.Lead, error) {
	quiz, err := json.Marshal(p.QuizResponses)
	if err != nil {
		return nil, fmt.Errorf("marshal quiz_responses: %w", err)
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO leads
			(email, first_name, last_name, org_name, org_type, phone,
			 city, state, zip, source, utm_source, utm_medium, utm_campaign,
			 quiz_responses, interested_grants, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'new')
		RETURNING id, email, first_name, last_name, org_name, org_type, phone,
		          city, state, zip, source, utm_source, utm_medium, utm_campaign,
		          status, score, assigned_to, converted_org_id, converted_at,
		          last_contacted_at, notes, quiz_responses, interested_grants,
		          created_at, updated_at`,
		p.Email, p.FirstName, p.LastName, p.OrgName, p.OrgType, p.Phone,
		p.City, p.State, p.Zip, p.Source, p.UTMSource, p.UTMMedium, p.UTMCampaign,
		quiz, p.InterestedGrants,
	)
	return scanLead(row)
}

func (r *leadRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lead, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, first_name, last_name, org_name, org_type, phone,
		       city, state, zip, source, utm_source, utm_medium, utm_campaign,
		       status, score, assigned_to, converted_org_id, converted_at,
		       last_contacted_at, notes, quiz_responses, interested_grants,
		       created_at, updated_at
		FROM leads WHERE id = $1`, id)
	return scanLead(row)
}

func (r *leadRepo) GetByEmail(ctx context.Context, email string) (*domain.Lead, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, first_name, last_name, org_name, org_type, phone,
		       city, state, zip, source, utm_source, utm_medium, utm_campaign,
		       status, score, assigned_to, converted_org_id, converted_at,
		       last_contacted_at, notes, quiz_responses, interested_grants,
		       created_at, updated_at
		FROM leads WHERE email = $1`, email)
	return scanLead(row)
}

func (r *leadRepo) List(ctx context.Context, limit, offset int32) ([]*domain.Lead, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, email, first_name, last_name, org_name, org_type, phone,
		       city, state, zip, source, utm_source, utm_medium, utm_campaign,
		       status, score, assigned_to, converted_org_id, converted_at,
		       last_contacted_at, notes, quiz_responses, interested_grants,
		       created_at, updated_at
		FROM leads ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLeads(rows)
}

func (r *leadRepo) ListByStatus(ctx context.Context, status domain.LeadStatus, limit, offset int32) ([]*domain.Lead, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, email, first_name, last_name, org_name, org_type, phone,
		       city, state, zip, source, utm_source, utm_medium, utm_campaign,
		       status, score, assigned_to, converted_org_id, converted_at,
		       last_contacted_at, notes, quiz_responses, interested_grants,
		       created_at, updated_at
		FROM leads WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLeads(rows)
}

func (r *leadRepo) Update(ctx context.Context, p repository.UpdateLeadParams) (*domain.Lead, error) {
	var lastContactedAt *time.Time
	if p.LastContactedAt != nil {
		t, err := time.Parse(time.RFC3339, *p.LastContactedAt)
		if err != nil {
			return nil, fmt.Errorf("parse last_contacted_at: %w", err)
		}
		lastContactedAt = &t
	}
	row := r.db.QueryRow(ctx, `
		UPDATE leads SET
			first_name       = COALESCE($2, first_name),
			last_name        = COALESCE($3, last_name),
			org_name         = COALESCE($4, org_name),
			phone            = COALESCE($5, phone),
			status           = COALESCE($6, status),
			score            = COALESCE($7, score),
			assigned_to      = COALESCE($8, assigned_to),
			notes            = COALESCE($9, notes),
			last_contacted_at = COALESCE($10, last_contacted_at),
			updated_at       = NOW()
		WHERE id = $1
		RETURNING id, email, first_name, last_name, org_name, org_type, phone,
		          city, state, zip, source, utm_source, utm_medium, utm_campaign,
		          status, score, assigned_to, converted_org_id, converted_at,
		          last_contacted_at, notes, quiz_responses, interested_grants,
		          created_at, updated_at`,
		p.ID, p.FirstName, p.LastName, p.OrgName, p.Phone,
		p.Status, p.Score, p.AssignedTo, p.Notes, lastContactedAt,
	)
	return scanLead(row)
}

func (r *leadRepo) Convert(ctx context.Context, id, orgID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE leads SET
			status = 'converted',
			converted_org_id = $2,
			converted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`, id, orgID)
	return err
}

func (r *leadRepo) LogActivity(ctx context.Context, p repository.LogLeadActivityParams) (*domain.LeadActivity, error) {
	var scheduledAt *time.Time
	if p.ScheduledAt != nil {
		t, err := time.Parse(time.RFC3339, *p.ScheduledAt)
		if err != nil {
			return nil, fmt.Errorf("parse scheduled_at: %w", err)
		}
		scheduledAt = &t
	}
	var a domain.LeadActivity
	err := r.db.QueryRow(ctx, `
		INSERT INTO lead_activities
			(lead_id, user_id, activity_type, subject, body,
			 old_value, new_value, scheduled_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, lead_id, user_id, activity_type, subject, body,
		          old_value, new_value, scheduled_at, completed_at, created_at`,
		p.LeadID, p.UserID, p.ActivityType, p.Subject, p.Body,
		p.OldValue, p.NewValue, scheduledAt,
	).Scan(&a.ID, &a.LeadID, &a.UserID, &a.ActivityType,
		&a.Subject, &a.Body, &a.OldValue, &a.NewValue,
		&a.ScheduledAt, &a.CompletedAt, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("log lead activity: %w", err)
	}
	return &a, nil
}

func (r *leadRepo) ListActivities(ctx context.Context, leadID uuid.UUID) ([]*domain.LeadActivity, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, lead_id, user_id, activity_type, subject, body,
		       old_value, new_value, scheduled_at, completed_at, created_at
		FROM lead_activities
		WHERE lead_id = $1
		ORDER BY created_at DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.LeadActivity
	for rows.Next() {
		var a domain.LeadActivity
		if err := rows.Scan(&a.ID, &a.LeadID, &a.UserID, &a.ActivityType,
			&a.Subject, &a.Body, &a.OldValue, &a.NewValue,
			&a.ScheduledAt, &a.CompletedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *leadRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM leads`).Scan(&n)
	return n, err
}

func (r *leadRepo) Search(ctx context.Context, query string, limit, offset int32) ([]*domain.Lead, error) {
	pattern := "%" + query + "%"
	rows, err := r.db.Query(ctx, `
		SELECT id, email, first_name, last_name, org_name, org_type, phone,
		       city, state, zip, source, utm_source, utm_medium, utm_campaign,
		       status, score, assigned_to, converted_org_id, converted_at,
		       last_contacted_at, notes, quiz_responses, interested_grants,
		       created_at, updated_at
		FROM leads
		WHERE email ILIKE $1 OR first_name ILIKE $1 OR last_name ILIKE $1 OR org_name ILIKE $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		pattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLeads(rows)
}

// --- scan helpers ---

func scanLead(row interface{ Scan(dest ...any) error }) (*domain.Lead, error) {
	var l domain.Lead
	var quizJSON []byte
	err := row.Scan(
		&l.ID, &l.Email, &l.FirstName, &l.LastName, &l.OrgName, &l.OrgType, &l.Phone,
		&l.City, &l.State, &l.Zip, &l.Source, &l.UTMSource, &l.UTMMedium, &l.UTMCampaign,
		&l.Status, &l.Score, &l.AssignedTo, &l.ConvertedOrgID, &l.ConvertedAt,
		&l.LastContactedAt, &l.Notes, &quizJSON, &l.InterestedGrants,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan lead: %w", err)
	}
	if len(quizJSON) > 0 {
		_ = json.Unmarshal(quizJSON, &l.QuizResponses)
	}
	return &l, nil
}

func scanLeads(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*domain.Lead, error) {
	var out []*domain.Lead
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
