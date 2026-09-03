package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrgType enumerates valid organization types.
type OrgType string

const (
	// Legacy generic types (kept for backward compatibility with existing data/tests).
	OrgTypeNonprofit  OrgType = "nonprofit"
	OrgTypeGovernment OrgType = "government"
	OrgTypeTribal     OrgType = "tribal"
	OrgTypeFaith      OrgType = "faith"
	OrgTypeOther      OrgType = "other"

	// Funding OS organization categories — the taxonomy used by the org-profile
	// step of the Funding OS wizard and by grant eligibility matching.
	OrgTypeK12Schools          OrgType = "k12-schools"
	OrgTypeHigherEd            OrgType = "higher-ed"
	OrgTypeMunicipalityGov     OrgType = "municipality-government"
	OrgTypePublicSafety        OrgType = "public-safety"
	OrgTypeHospitalsHealth     OrgType = "hospitals-health-systems"
	OrgTypeEMSHealthcare       OrgType = "ems-healthcare"
	OrgTypeNonprofitCommunity  OrgType = "nonprofit-community"
	OrgTypeHousesOfWorship     OrgType = "houses-of-worship"
	OrgTypeWorkplaceIndustrial OrgType = "workplace-industrial"
	OrgTypeCorporateCampuses   OrgType = "corporate-campuses"
	OrgTypeParksRecreation     OrgType = "parks-recreation"
	OrgTypeSportsEntertainment OrgType = "sports-entertainment"
	OrgTypeTransitTransport    OrgType = "transit-transportation"
)

// FundingOSOrgTypes lists the 14 org categories offered in the Funding OS
// wizard's "Start With Your Organization" step, in display order.
var FundingOSOrgTypes = []struct {
	Key         OrgType
	Label       string
	Description string
	Icon        string
}{
	{OrgTypeK12Schools, "K-12 Schools & Districts", "Public schools, charter schools, and school districts.", "🏫"},
	{OrgTypeHigherEd, "Colleges & Universities", "Higher education campuses and community colleges.", "🎓"},
	{OrgTypeMunicipalityGov, "Municipality / Government", "City, county, state, and tribal government.", "🏛️"},
	{OrgTypePublicSafety, "Public Safety Agencies", "Police, fire, emergency management, and public safety departments.", "🚨"},
	{OrgTypeHospitalsHealth, "Hospitals & Health Systems", "Hospitals, medical campuses, and healthcare systems.", "🏥"},
	{OrgTypeEMSHealthcare, "EMS / Healthcare", "EMS agencies, clinics, and community health providers.", "🚑"},
	{OrgTypeNonprofitCommunity, "Nonprofit / Community Organizations", "Community-serving nonprofits, foundations, and local organizations.", "🤝"},
	{OrgTypeHousesOfWorship, "Houses of Worship", "Churches, synagogues, mosques, and religious organizations.", "⛪"},
	{OrgTypeWorkplaceIndustrial, "Workplace / Industrial", "Manufacturing, logistics, warehouses, and industrial facilities.", "🏭"},
	{OrgTypeCorporateCampuses, "Corporate Campuses", "Office parks, headquarters, and corporate facilities.", "🏢"},
	{OrgTypeParksRecreation, "Parks & Recreation", "Parks departments, recreation centers, and outdoor venues.", "🌲"},
	{OrgTypeSportsEntertainment, "Sports & Entertainment Venues", "Arenas, stadiums, concert venues, and event facilities.", "🏟️"},
	{OrgTypeTransitTransport, "Transit & Transportation", "Transit systems, airports, rail, and transportation hubs.", "🚆"},
	{OrgTypeOther, "Other", "Custom organization type or mixed-use organization.", "📋"},
}

// PreparednessPriority enumerates the "what are you preparing for?" needs
// captured in step 2 of the Funding OS wizard. These keys double as the
// canonical focus-area tags used to match organizations against grants.
type PreparednessPriority string

const (
	PriorityBleedingControl     PreparednessPriority = "bleeding-control"
	PriorityCardiacResponse     PreparednessPriority = "cardiac-response"
	PriorityOpioidResponse      PreparednessPriority = "opioid-response"
	PriorityEventPreparedness   PreparednessPriority = "event-preparedness"
	PriorityEmergencyComms      PreparednessPriority = "emergency-communications"
	PrioritySecurityIntegration PreparednessPriority = "security-integration"
	PriorityChokingResponse     PreparednessPriority = "choking-response"
	PriorityAllergicReactions   PreparednessPriority = "allergic-reactions"
	PriorityAsthmaResponse      PreparednessPriority = "asthma-response"
)

// FundingOSPriorities lists the 9 preparedness priority cards, in display order.
var FundingOSPriorities = []struct {
	Key         PreparednessPriority
	Label       string
	Description string
	Icon        string
}{
	{PriorityBleedingControl, "Bleeding Control", "Rapid access to lifesaving hemorrhage control equipment.", "🩸"},
	{PriorityCardiacResponse, "Cardiac Response", "Expand AED access and improve response to sudden cardiac arrest.", "❤️"},
	{PriorityOpioidResponse, "Opioid Overdose Response", "Naloxone access and public health preparedness for overdose emergencies.", "💊"},
	{PriorityEventPreparedness, "Event & Community Preparedness", "Support athletics, festivals, graduations, and large gatherings.", "🎪"},
	{PriorityEmergencyComms, "Emergency Communications", "Deliver emergency alerts, instructions, and real-time awareness.", "📣"},
	{PrioritySecurityIntegration, "Security Integration", "Connect preparedness infrastructure with cameras and security systems.", "🎥"},
	{PriorityChokingResponse, "Choking Response", "Emergency airway clearance equipment and rescue readiness for choking incidents.", "🆘"},
	{PriorityAllergicReactions, "Severe Allergic Reactions", "Epinephrine access and anaphylaxis response.", "⚕️"},
	{PriorityAsthmaResponse, "Asthma Attack Response", "Emergency asthma support kits and respiratory preparedness.", "🫁"},
}

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
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	EIN           *string    `json:"ein,omitempty"`
	OrgType       OrgType    `json:"org_type"`
	Mission       *string    `json:"mission,omitempty"`
	AddressLine1  *string    `json:"address_line1,omitempty"`
	AddressLine2  *string    `json:"address_line2,omitempty"`
	City          *string    `json:"city,omitempty"`
	State         *string    `json:"state,omitempty"`
	Zip           *string    `json:"zip,omitempty"`
	County        *string    `json:"county,omitempty"`
	Website       *string    `json:"website,omitempty"`
	Phone         *string    `json:"phone,omitempty"`
	LogoURL       *string    `json:"logo_url,omitempty"`
	Plan          Plan       `json:"plan"`
	PlanExpiresAt *time.Time `json:"plan_expires_at,omitempty"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Joined / admin fields
	OwnerEmail *string `json:"owner_email,omitempty"`
}

// OrganizationProfile stores detailed org attributes for matching.
type OrganizationProfile struct {
	ID                   uuid.UUID `json:"id"`
	OrgID                uuid.UUID `json:"org_id"`
	AnnualBudget         *int64    `json:"annual_budget,omitempty"`
	NumEmployees         *int32    `json:"num_employees,omitempty"`
	NumVolunteers        *int32    `json:"num_volunteers,omitempty"`
	YearsOperating       *int32    `json:"years_operating,omitempty"`
	PopulationsServed    []string  `json:"populations_served"`
	ServiceAreas         []string  `json:"service_areas"`
	ProgramAreas         []string  `json:"program_areas"`
	FocusIssues          []string  `json:"focus_issues"`
	Has501c3             bool      `json:"has_501c3"`
	HasAuditedFinancials bool      `json:"has_audited_financials"`
	HasIndirectCostRate  bool      `json:"has_indirect_cost_rate"`
	IndirectCostRatePct  *float64  `json:"indirect_cost_rate_pct,omitempty"`
	PriorFederalGrants   bool      `json:"prior_federal_grants"`
	Narrative            *string   `json:"narrative,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
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
