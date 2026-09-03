package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

type organizationRepo struct {
	db *pgxpool.Pool
}

// NewOrganizationRepo creates a pgx-backed OrganizationRepo.
func NewOrganizationRepo(db *pgxpool.Pool) repository.OrganizationRepo {
	return &organizationRepo{db: db}
}

func (r *organizationRepo) Create(ctx context.Context, p repository.CreateOrgParams) (*domain.Organization, error) {
	const q = `
		INSERT INTO organizations (name, slug, ein, org_type, mission, city, state, zip, website, phone, plan)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, name, slug, ein, org_type, mission, address_line1, address_line2,
		          city, state, zip, county, website, phone, logo_url, plan, plan_expires_at,
		          is_active, created_at, updated_at`
	row := r.db.QueryRow(ctx, q,
		p.Name, p.Slug, p.EIN, string(p.OrgType), p.Mission,
		p.City, p.State, p.Zip, p.Website, p.Phone, string(p.Plan),
	)
	return scanOrg(row)
}

func (r *organizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	const q = `SELECT id, name, slug, ein, org_type, mission, address_line1, address_line2,
	                  city, state, zip, county, website, phone, logo_url, plan, plan_expires_at,
	                  is_active, created_at, updated_at, NULL as owner_email
	           FROM organizations WHERE id = $1 AND is_active = TRUE`
	return scanOrg(r.db.QueryRow(ctx, q, id))
}

func (r *organizationRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	const q = `SELECT id, name, slug, ein, org_type, mission, address_line1, address_line2,
	                  city, state, zip, county, website, phone, logo_url, plan, plan_expires_at,
	                  is_active, created_at, updated_at, NULL as owner_email
	           FROM organizations WHERE slug = $1 AND is_active = TRUE`
	return scanOrg(r.db.QueryRow(ctx, q, slug))
}

func (r *organizationRepo) List(ctx context.Context, limit, offset int32) ([]*domain.Organization, error) {
	const q = `SELECT o.id, o.name, o.slug, o.ein, o.org_type, o.mission, o.address_line1, o.address_line2,
	                  o.city, o.state, o.zip, o.county, o.website, o.phone, o.logo_url, o.plan, o.plan_expires_at,
	                  o.is_active, o.created_at, o.updated_at,
	                  (SELECT u.email FROM organization_members om
	                   JOIN users u ON u.id = om.user_id
	                   WHERE om.org_id = o.id AND (om.role = 'owner' OR om.role = 'admin')
	                   ORDER BY om.role, om.created_at ASC LIMIT 1) as owner_email
	           FROM organizations o WHERE o.is_active = TRUE
	           ORDER BY o.created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()
	var orgs []*domain.Organization
	for rows.Next() {
		o, err := scanOrg(rows)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (r *organizationRepo) Update(ctx context.Context, p repository.UpdateOrgParams) (*domain.Organization, error) {
	const q = `
		UPDATE organizations
		SET name       = COALESCE($2, name),
		    org_type   = COALESCE($3, org_type),
		    mission    = COALESCE($4, mission),
		    city       = COALESCE($5, city),
		    state      = COALESCE($6, state),
		    zip        = COALESCE($7, zip),
		    website    = COALESCE($8, website),
		    phone      = COALESCE($9, phone),
		    logo_url   = COALESCE($10, logo_url),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, slug, ein, org_type, mission, address_line1, address_line2,
		          city, state, zip, county, website, phone, logo_url, plan, plan_expires_at,
		          is_active, created_at, updated_at`
	return scanOrg(r.db.QueryRow(ctx, q,
		p.ID, p.Name, p.OrgType, p.Mission, p.City, p.State, p.Zip, p.Website, p.Phone, p.LogoURL,
	))
}

func (r *organizationRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE organizations SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *organizationRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM organizations WHERE is_active=TRUE`).Scan(&n)
	return n, err
}

func (r *organizationRepo) UpsertProfile(ctx context.Context, p repository.UpsertProfileParams) (*domain.OrganizationProfile, error) {
	var emb *pgvector.Vector
	if len(p.Embedding) > 0 {
		v := pgvector.NewVector(p.Embedding)
		emb = &v
	}
	const q = `
		INSERT INTO organization_profiles (
			org_id, annual_budget, num_employees, num_volunteers, years_operating,
			populations_served, service_areas, program_areas, focus_issues,
			has_501c3, has_audited_financials, has_indirect_cost_rate,
			indirect_cost_rate_pct, prior_federal_grants, narrative, embedding
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (org_id) DO UPDATE SET
			annual_budget        = EXCLUDED.annual_budget,
			num_employees        = EXCLUDED.num_employees,
			num_volunteers       = EXCLUDED.num_volunteers,
			years_operating      = EXCLUDED.years_operating,
			populations_served   = EXCLUDED.populations_served,
			service_areas        = EXCLUDED.service_areas,
			program_areas        = EXCLUDED.program_areas,
			focus_issues         = EXCLUDED.focus_issues,
			has_501c3            = EXCLUDED.has_501c3,
			has_audited_financials = EXCLUDED.has_audited_financials,
			has_indirect_cost_rate = EXCLUDED.has_indirect_cost_rate,
			indirect_cost_rate_pct = EXCLUDED.indirect_cost_rate_pct,
			prior_federal_grants = EXCLUDED.prior_federal_grants,
			narrative            = EXCLUDED.narrative,
			embedding            = EXCLUDED.embedding,
			updated_at           = NOW()
		RETURNING id, org_id, annual_budget, num_employees, num_volunteers, years_operating,
		          populations_served, service_areas, program_areas, focus_issues,
		          has_501c3, has_audited_financials, has_indirect_cost_rate,
		          indirect_cost_rate_pct, prior_federal_grants, narrative, created_at, updated_at`

	row := r.db.QueryRow(ctx, q,
		p.OrgID, p.AnnualBudget, p.NumEmployees, p.NumVolunteers, p.YearsOperating,
		p.PopulationsServed, p.ServiceAreas, p.ProgramAreas, p.FocusIssues,
		p.Has501c3, p.HasAuditedFinancials, p.HasIndirectCostRate,
		p.IndirectCostRatePct, p.PriorFederalGrants, p.Narrative, emb,
	)
	return scanProfile(row)
}

func (r *organizationRepo) GetProfile(ctx context.Context, orgID uuid.UUID) (*domain.OrganizationProfile, error) {
	const q = `SELECT id, org_id, annual_budget, num_employees, num_volunteers, years_operating,
	                  populations_served, service_areas, program_areas, focus_issues,
	                  has_501c3, has_audited_financials, has_indirect_cost_rate,
	                  indirect_cost_rate_pct, prior_federal_grants, narrative, created_at, updated_at
	           FROM organization_profiles WHERE org_id = $1`
	return scanProfile(r.db.QueryRow(ctx, q, orgID))
}

func (r *organizationRepo) UpdateProfileEmbedding(ctx context.Context, orgID uuid.UUID, embedding []float32) error {
	v := pgvector.NewVector(embedding)
	_, err := r.db.Exec(ctx,
		`UPDATE organization_profiles SET embedding=$2, updated_at=NOW() WHERE org_id=$1`,
		orgID, v,
	)
	return err
}

// scanOrg scans a pgx Row/Rows into domain.Organization.
type scannable interface {
	Scan(dest ...any) error
}

func scanOrg(row scannable) (*domain.Organization, error) {
	var o domain.Organization
	var orgType, plan string
	err := row.Scan(
		&o.ID, &o.Name, &o.Slug, &o.EIN, &orgType, &o.Mission,
		&o.AddressLine1, &o.AddressLine2, &o.City, &o.State, &o.Zip, &o.County,
		&o.Website, &o.Phone, &o.LogoURL, &plan, &o.PlanExpiresAt,
		&o.IsActive, &o.CreatedAt, &o.UpdatedAt, &o.OwnerEmail,
	)
	if err != nil {
		return nil, fmt.Errorf("scan org: %w", err)
	}
	o.OrgType = domain.OrgType(orgType)
	o.Plan = domain.Plan(plan)
	return &o, nil
}

func scanProfile(row scannable) (*domain.OrganizationProfile, error) {
	var p domain.OrganizationProfile
	var indirectRate *float64
	var narrative *string
	err := row.Scan(
		&p.ID, &p.OrgID, &p.AnnualBudget, &p.NumEmployees, &p.NumVolunteers, &p.YearsOperating,
		&p.PopulationsServed, &p.ServiceAreas, &p.ProgramAreas, &p.FocusIssues,
		&p.Has501c3, &p.HasAuditedFinancials, &p.HasIndirectCostRate,
		&indirectRate, &p.PriorFederalGrants, &narrative,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan profile: %w", err)
	}
	p.IndirectCostRatePct = indirectRate
	p.Narrative = narrative
	return &p, nil
}

// jsonMarshal is a helper for JSONB fields.
func jsonMarshal(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}
