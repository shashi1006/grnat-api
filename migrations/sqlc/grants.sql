-- name: CreateGrant :one
INSERT INTO grants (
    slug, title, funder_name, funder_type, program_number, opportunity_number,
    agency, sub_agency, description, synopsis, full_nofo_text, category, subcategory,
    focus_areas, eligible_org_types, eligible_populations, eligible_states,
    requires_501c3, requires_audited_fin, requires_indirect_rate, requires_match,
    match_percentage, min_award_amount, max_award_amount, avg_award_amount,
    total_funding_available, num_awards_expected, application_url, status,
    deadline, open_date, period_of_performance, difficulty_level, competition_level,
    tags, metadata, created_by
)
VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,
    $18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37
)
RETURNING *;

-- name: GetGrantByID :one
SELECT * FROM grants WHERE id = $1;

-- name: GetGrantBySlug :one
SELECT * FROM grants WHERE slug = $1;

-- name: ListGrants :many
SELECT * FROM grants
WHERE status = ANY(COALESCE($3::text[], ARRAY['active']))
ORDER BY deadline ASC NULLS LAST, created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListGrantsByCategory :many
SELECT * FROM grants
WHERE category = $1 AND status = 'active'
ORDER BY deadline ASC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: SearchGrantsByTitle :many
SELECT * FROM grants
WHERE title ILIKE '%' || $1 || '%' AND status = 'active'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListGrantsForOrgType :many
SELECT * FROM grants
WHERE $1 = ANY(eligible_org_types) AND status = 'active'
ORDER BY deadline ASC NULLS LAST
LIMIT $2 OFFSET $3;

-- name: UpdateGrant :one
UPDATE grants
SET title = COALESCE($2, title),
    description = COALESCE($3, description),
    synopsis = COALESCE($4, synopsis),
    status = COALESCE($5, status),
    deadline = COALESCE($6, deadline),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateGrantEmbedding :exec
UPDATE grants SET embedding = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateGrantNOFO :exec
UPDATE grants SET full_nofo_text = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteGrant :exec
UPDATE grants SET status = 'archived', updated_at = NOW() WHERE id = $1;

-- name: CountGrants :one
SELECT COUNT(*) FROM grants WHERE status != 'archived';

-- name: CountActiveGrants :one
SELECT COUNT(*) FROM grants WHERE status = 'active';

-- name: UpsertNofoChunk :one
INSERT INTO nofo_chunks (grant_id, chunk_index, section, content, token_count, embedding)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (grant_id, chunk_index) DO UPDATE SET
    section = EXCLUDED.section,
    content = EXCLUDED.content,
    token_count = EXCLUDED.token_count,
    embedding = EXCLUDED.embedding
RETURNING *;

-- name: ListNofoChunks :many
SELECT * FROM nofo_chunks WHERE grant_id = $1 ORDER BY chunk_index ASC;

-- name: SearchNofoChunksBySimilarity :many
SELECT nc.*, (nc.embedding <=> $1::vector) AS distance
FROM nofo_chunks nc
WHERE nc.grant_id = $2
ORDER BY distance ASC
LIMIT $3;

-- name: SearchGrantsBySimilarity :many
SELECT g.*, (g.embedding <=> $1::vector) AS distance
FROM grants g
WHERE g.status = 'active'
ORDER BY distance ASC
LIMIT $2;

-- name: UpsertGrantScoringCriterion :one
INSERT INTO grant_scoring_criteria (grant_id, criterion_key, weight, is_required, disqualifying, description)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (grant_id, criterion_key) DO UPDATE SET
    weight = EXCLUDED.weight,
    is_required = EXCLUDED.is_required,
    disqualifying = EXCLUDED.disqualifying,
    description = EXCLUDED.description
RETURNING *;

-- name: ListGrantScoringCriteria :many
SELECT * FROM grant_scoring_criteria WHERE grant_id = $1 ORDER BY criterion_key;
