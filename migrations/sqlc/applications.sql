-- name: CreateApplication :one
INSERT INTO grant_applications (org_id, grant_id, assigned_to, status, stage, priority, compatibility_score, notes, internal_deadline)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetApplication :one
SELECT ga.*, g.title AS grant_title, g.funder_name, g.deadline AS grant_deadline
FROM grant_applications ga
JOIN grants g ON g.id = ga.grant_id
WHERE ga.id = $1;

-- name: GetApplicationByOrgAndGrant :one
SELECT * FROM grant_applications WHERE org_id = $1 AND grant_id = $2;

-- name: ListApplicationsForOrg :many
SELECT ga.*, g.title AS grant_title, g.funder_name, g.category, g.deadline AS grant_deadline
FROM grant_applications ga
JOIN grants g ON g.id = ga.grant_id
WHERE ga.org_id = $1
ORDER BY ga.updated_at DESC
LIMIT $2 OFFSET $3;

-- name: ListApplicationsByStatus :many
SELECT ga.*, g.title AS grant_title, o.name AS org_name
FROM grant_applications ga
JOIN grants g ON g.id = ga.grant_id
JOIN organizations o ON o.id = ga.org_id
WHERE ga.status = $1
ORDER BY ga.updated_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateApplicationStatus :one
UPDATE grant_applications
SET status = $2, stage = COALESCE($3, stage), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateApplication :one
UPDATE grant_applications
SET assigned_to = COALESCE($2, assigned_to),
    status = COALESCE($3, status),
    stage = COALESCE($4, stage),
    priority = COALESCE($5, priority),
    notes = COALESCE($6, notes),
    internal_deadline = COALESCE($7, internal_deadline),
    submission_date = COALESCE($8, submission_date),
    award_amount = COALESCE($9, award_amount),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: LogApplicationActivity :one
INSERT INTO application_activities (application_id, user_id, activity_type, old_value, new_value, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListApplicationActivities :many
SELECT aa.*, u.first_name, u.last_name, u.email
FROM application_activities aa
LEFT JOIN users u ON u.id = aa.user_id
WHERE aa.application_id = $1
ORDER BY aa.created_at DESC;

-- name: CreateNarrative :one
INSERT INTO ai_narratives (org_id, grant_id, application_id, section_key, prompt_used, content, word_count, model_used, tokens_in, tokens_out)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListNarrativesForApplication :many
SELECT * FROM ai_narratives WHERE application_id = $1 ORDER BY section_key, version DESC;

-- name: GetLatestNarrativeBySection :one
SELECT * FROM ai_narratives
WHERE application_id = $1 AND section_key = $2
ORDER BY version DESC
LIMIT 1;

-- name: CountApplicationsByStatus :many
SELECT status, COUNT(*) AS count
FROM grant_applications
WHERE org_id = $1
GROUP BY status;

-- name: CountApplicationsByOrg :one
SELECT COUNT(*) FROM grant_applications WHERE org_id = $1;
