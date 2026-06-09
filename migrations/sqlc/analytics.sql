-- name: LogAnalyticsEvent :one
INSERT INTO analytics_events (org_id, user_id, lead_id, event_type, entity_type, entity_id, properties, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListAnalyticsEvents :many
SELECT * FROM analytics_events
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: LogAIUsage :one
INSERT INTO ai_usage_logs (org_id, user_id, operation, model, tokens_in, tokens_out, cost_usd_cents, latency_ms, success, error_message)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: SumAIUsageForOrg :one
SELECT
    SUM(tokens_in)       AS total_tokens_in,
    SUM(tokens_out)      AS total_tokens_out,
    SUM(cost_usd_cents)  AS total_cost_cents,
    COUNT(*)             AS total_requests
FROM ai_usage_logs
WHERE org_id = $1 AND created_at >= $2;

-- name: UpsertDailyMetric :exec
INSERT INTO daily_metrics (metric_date, org_id, metric_key, metric_value, dimensions)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (metric_date, org_id, metric_key)
DO UPDATE SET metric_value = EXCLUDED.metric_value, dimensions = EXCLUDED.dimensions;

-- name: GetDailyMetrics :many
SELECT * FROM daily_metrics
WHERE org_id = $1 AND metric_date >= $2 AND metric_date <= $3
ORDER BY metric_date ASC;

-- name: GetPlatformStats :one
SELECT
    (SELECT COUNT(*) FROM organizations WHERE is_active = TRUE) AS total_orgs,
    (SELECT COUNT(*) FROM users WHERE is_active = TRUE)         AS total_users,
    (SELECT COUNT(*) FROM grants WHERE status = 'active')       AS active_grants,
    (SELECT COUNT(*) FROM leads)                                AS total_leads,
    (SELECT COUNT(*) FROM grant_applications)                   AS total_applications;
