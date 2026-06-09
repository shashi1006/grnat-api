package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrgType enumerates valid organization types.
type OrgType string

const (
	OrgTypeNonprofit  OrgType = "nonprofit"
	OrgTypeGovernment OrgType = "government"
	OrgTypeTribal     OrgType = "tribal"
	OrgTypeFaith      OrgType = "faith"
	OrgTypeOther      OrgType = "other"
)

// Plan enumerates subscription plans.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanStarter    Plan = "starter"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

// Organization is the core tenant entity.
type Organization struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	EIN          *string    `json:"ein,omitempty"`
	OrgType      OrgType    `json:"org_type"`
	Mission      *string    `json:"mission,omitempty"`
	AddressLine1 *string    `json:"address_line1,omitempty"`
	AddressLine2 *string    `json:"address_line2,omitempty"`
	City         *string    `json:"city,omitempty"`
	State        *string    `json:"state,omitempty"`
	Zip          *string    `json:"zip,omitempty"`
	County       *string    `json:"county,omitempty"`
	Website      *string    `json:"website,omitempty"`
	Phone        *string    `json:"phone,omitempty"`
	LogoURL      *string    `json:"logo_url,omitempty"`
	Plan         Plan       `json:"plan"`
	PlanExpiresAt *time.Time `json:"plan_expires_at,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// OrganizationProfile stores detailed org attributes for matching.
type OrganizationProfile struct {
	ID                   uuid.UUID  `json:"id"`
	OrgID                uuid.UUID  `json:"org_id"`
	AnnualBudget         *int64     `json:"annual_budget,omitempty"`
	NumEmployees         *int32     `json:"num_employees,omitempty"`
	NumVolunteers        *int32     `json:"num_volunteers,omitempty"`
	YearsOperating       *int32     `json:"years_operating,omitempty"`
	PopulationsServed    []string   `json:"populations_served"`
	ServiceAreas         []string   `json:"service_areas"`
	ProgramAreas         []string   `json:"program_areas"`
	FocusIssues          []string   `json:"focus_issues"`
	Has501c3             bool       `json:"has_501c3"`
	HasAuditedFinancials bool       `json:"has_audited_financials"`
	HasIndirectCostRate  bool       `json:"has_indirect_cost_rate"`
	IndirectCostRatePct  *float64   `json:"indirect_cost_rate_pct,omitempty"`
	PriorFederalGrants   bool       `json:"prior_federal_grants"`
	Narrative            *string    `json:"narrative,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// OrgMemberRole enumerates roles within an org.
type OrgMemberRole string

const (
	RoleOwner  OrgMemberRole = "owner"
	RoleAdmin  OrgMemberRole = "admin"
	RoleMember OrgMemberRole = "member"
	RoleViewer OrgMemberRole = "viewer"
)

// OrganizationMember links a user to an org with a role.
type OrganizationMember struct {
	ID        uuid.UUID     `json:"id"`
	OrgID     uuid.UUID     `json:"org_id"`
	UserID    uuid.UUID     `json:"user_id"`
	Role      OrgMemberRole `json:"role"`
	IsActive  bool          `json:"is_active"`
	InvitedBy *uuid.UUID    `json:"invited_by,omitempty"`
	JoinedAt  *time.Time    `json:"joined_at,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
