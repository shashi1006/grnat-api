package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type analyticsRepo struct {
	db *pgxpool.Pool
}

// NewAnalyticsRepo creates an AnalyticsRepo backed by pgx.
func NewAnalyticsRepo(db *pgxpool.Pool) repository.AnalyticsRepo {
	return &analyticsRepo{db: db}
}

func (r *analyticsRepo) LogEvent(ctx context.Context, p repository.LogEventParams) error {
	props, err := json.Marshal(p.Properties)
	if err != nil {
		return fmt.Errorf("marshal event properties: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO analytics_events
			(org_id, user_id, lead_id, event_type, entity_type, entity_id,
			 properties, ip_address, user_agent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.OrgID, p.UserID, p.LeadID, p.EventType, p.EntityType, p.EntityID,
		props, p.IPAddress, p.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("log event: %w", err)
	}
	return nil
}

func (r *analyticsRepo) LogAIUsage(ctx context.Context, p repository.LogAIUsageParams) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_usage_logs
			(org_id, user_id, operation, model, tokens_in, tokens_out,
			 cost_usd_cents, latency_ms, success, error_message)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		p.OrgID, p.UserID, p.Operation, p.Model, p.TokensIn, p.TokensOut,
		p.CostUSDCents, p.LatencyMS, p.Success, p.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("log ai usage: %w", err)
	}
	return nil
}

func (r *analyticsRepo) GetPlatformStats(ctx context.Context) (*repository.PlatformStats, error) {
	var s repository.PlatformStats
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM organizations)    AS total_orgs,
			(SELECT COUNT(*) FROM users)            AS total_users,
			(SELECT COUNT(*) FROM grants WHERE status = 'active') AS active_grants,
			(SELECT COUNT(*) FROM leads)            AS total_leads,
			(SELECT COUNT(*) FROM grant_applications) AS total_applications
	`).Scan(&s.TotalOrgs, &s.TotalUsers, &s.ActiveGrants, &s.TotalLeads, &s.TotalApplications)
	if err != nil {
		return nil, fmt.Errorf("get platform stats: %w", err)
	}
	return &s, nil
}
