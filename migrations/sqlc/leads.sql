-- name: CreateLead :one
INSERT INTO leads (email, first_name, last_name, org_name, org_type, phone, city, state, zip, source, utm_source, utm_medium, utm_campaign, quiz_responses, interested_grants, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: GetLeadByID :one
SELECT * FROM leads WHERE id = $1;

-- name: GetLeadByEmail :one
SELECT * FROM leads WHERE email = $1;

-- name: ListLeads :many
SELECT * FROM leads
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListLeadsByStatus :many
SELECT * FROM leads
WHERE status = $1
ORDER BY score DESC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateLead :one
UPDATE leads
SET first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    org_name = COALESCE($4, org_name),
    phone = COALESCE($5, phone),
    status = COALESCE($6, status),
    score = COALESCE($7, score),
    assigned_to = COALESCE($8, assigned_to),
    notes = COALESCE($9, notes),
    last_contacted_at = COALESCE($10, last_contacted_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ConvertLead :exec
UPDATE leads
SET status = 'converted',
    converted_org_id = $2,
    converted_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: LogLeadActivity :one
INSERT INTO lead_activities (lead_id, user_id, activity_type, subject, body, old_value, new_value, scheduled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListLeadActivities :many
SELECT la.*, u.first_name, u.last_name
FROM lead_activities la
LEFT JOIN users u ON u.id = la.user_id
WHERE la.lead_id = $1
ORDER BY la.created_at DESC;

-- name: CountLeadsByStatus :many
SELECT status, COUNT(*) AS count
FROM leads
GROUP BY status;

-- name: CountLeads :one
SELECT COUNT(*) FROM leads;

-- name: SearchLeads :many
SELECT * FROM leads
WHERE email ILIKE '%' || $1 || '%'
   OR org_name ILIKE '%' || $1 || '%'
   OR first_name ILIKE '%' || $1 || '%'
   OR last_name ILIKE '%' || $1 || '%'
ORDER BY score DESC
LIMIT $2 OFFSET $3;
