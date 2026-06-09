-- name: UpsertCompatibilityScore :one
INSERT INTO compatibility_scores (
    org_id, grant_id, total_score, tier, dimension_scores,
    disqualified, disqualify_reasons, strengths, gaps, recommendations,
    semantic_score, engine_version
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (org_id, grant_id) DO UPDATE SET
    total_score = EXCLUDED.total_score,
    tier = EXCLUDED.tier,
    dimension_scores = EXCLUDED.dimension_scores,
    disqualified = EXCLUDED.disqualified,
    disqualify_reasons = EXCLUDED.disqualify_reasons,
    strengths = EXCLUDED.strengths,
    gaps = EXCLUDED.gaps,
    recommendations = EXCLUDED.recommendations,
    semantic_score = EXCLUDED.semantic_score,
    engine_version = EXCLUDED.engine_version,
    computed_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetCompatibilityScore :one
SELECT * FROM compatibility_scores WHERE org_id = $1 AND grant_id = $2;

-- name: ListTopGrantsForOrg :many
SELECT cs.*, g.title, g.funder_name, g.category, g.deadline,
       g.min_award_amount, g.max_award_amount, g.status
FROM compatibility_scores cs
JOIN grants g ON g.id = cs.grant_id
WHERE cs.org_id = $1 AND cs.disqualified = FALSE AND g.status = 'active'
ORDER BY cs.total_score DESC
LIMIT $2 OFFSET $3;

-- name: ListOrgsForGrant :many
SELECT cs.*, o.name AS org_name, o.state, o.org_type
FROM compatibility_scores cs
JOIN organizations o ON o.id = cs.org_id
WHERE cs.grant_id = $1 AND cs.disqualified = FALSE
ORDER BY cs.total_score DESC
LIMIT $2 OFFSET $3;

-- name: DeleteCompatibilityScoresForOrg :exec
DELETE FROM compatibility_scores WHERE org_id = $1;

-- name: DeleteCompatibilityScoresForGrant :exec
DELETE FROM compatibility_scores WHERE grant_id = $1;
