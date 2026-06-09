-- name: CreateOrganization :one
INSERT INTO organizations (name, slug, ein, org_type, mission, address_line1, address_line2, city, state, zip, county, website, phone, plan, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetOrganizationByID :one
SELECT * FROM organizations WHERE id = $1 AND is_active = TRUE;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1 AND is_active = TRUE;

-- name: ListOrganizations :many
SELECT * FROM organizations
WHERE is_active = TRUE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateOrganization :one
UPDATE organizations
SET name = COALESCE($2, name),
    mission = COALESCE($3, mission),
    address_line1 = COALESCE($4, address_line1),
    city = COALESCE($5, city),
    state = COALESCE($6, state),
    zip = COALESCE($7, zip),
    website = COALESCE($8, website),
    phone = COALESCE($9, phone),
    logo_url = COALESCE($10, logo_url),
    metadata = COALESCE($11, metadata),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateOrganization :exec
UPDATE organizations SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: CountOrganizations :one
SELECT COUNT(*) FROM organizations WHERE is_active = TRUE;

-- name: UpsertOrganizationProfile :one
INSERT INTO organization_profiles (
    org_id, annual_budget, num_employees, num_volunteers, years_operating,
    populations_served, service_areas, program_areas, focus_issues,
    has_501c3, has_audited_financials, has_indirect_cost_rate, indirect_cost_rate_pct,
    prior_federal_grants, narrative, embedding
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (org_id) DO UPDATE SET
    annual_budget = EXCLUDED.annual_budget,
    num_employees = EXCLUDED.num_employees,
    num_volunteers = EXCLUDED.num_volunteers,
    years_operating = EXCLUDED.years_operating,
    populations_served = EXCLUDED.populations_served,
    service_areas = EXCLUDED.service_areas,
    program_areas = EXCLUDED.program_areas,
    focus_issues = EXCLUDED.focus_issues,
    has_501c3 = EXCLUDED.has_501c3,
    has_audited_financials = EXCLUDED.has_audited_financials,
    has_indirect_cost_rate = EXCLUDED.has_indirect_cost_rate,
    indirect_cost_rate_pct = EXCLUDED.indirect_cost_rate_pct,
    prior_federal_grants = EXCLUDED.prior_federal_grants,
    narrative = EXCLUDED.narrative,
    embedding = EXCLUDED.embedding,
    updated_at = NOW()
RETURNING *;

-- name: GetOrganizationProfile :one
SELECT * FROM organization_profiles WHERE org_id = $1;

-- name: UpdateOrgProfileEmbedding :exec
UPDATE organization_profiles SET embedding = $2, updated_at = NOW() WHERE org_id = $1;

-- name: GetOrgsWithProfile :many
SELECT o.*, op.program_areas, op.populations_served, op.has_501c3
FROM organizations o
JOIN organization_profiles op ON op.org_id = o.id
WHERE o.is_active = TRUE
ORDER BY o.created_at DESC
LIMIT $1 OFFSET $2;
