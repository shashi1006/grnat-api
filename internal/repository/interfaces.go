package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
)

// ErrNotFound indicates a requested record does not exist.
var ErrNotFound = errors.New("not found")

// --- Organization ---

type OrganizationRepo interface {
	Create(ctx context.Context, params CreateOrgParams) (*domain.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	List(ctx context.Context, limit, offset int32) ([]*domain.Organization, error)
	Update(ctx context.Context, params UpdateOrgParams) (*domain.Organization, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
	UpsertProfile(ctx context.Context, params UpsertProfileParams) (*domain.OrganizationProfile, error)
	GetProfile(ctx context.Context, orgID uuid.UUID) (*domain.OrganizationProfile, error)
	UpdateProfileEmbedding(ctx context.Context, orgID uuid.UUID, embedding []float32) error
}

type CreateOrgParams struct {
	Name    string
	Slug    string
	EIN     *string
	OrgType domain.OrgType
	Mission *string
	City    *string
	State   *string
	Zip     *string
	Website *string
	Phone   *string
	Plan    domain.Plan
}

type UpdateOrgParams struct {
	ID      uuid.UUID
	Name    *string
	Mission *string
	City    *string
	State   *string
	Zip     *string
	Website *string
	Phone   *string
	LogoURL *string
}

type UpsertProfileParams struct {
	OrgID                uuid.UUID
	AnnualBudget         *int64
	NumEmployees         *int32
	NumVolunteers        *int32
	YearsOperating       *int32
	PopulationsServed    []string
	ServiceAreas         []string
	ProgramAreas         []string
	FocusIssues          []string
	Has501c3             bool
	HasAuditedFinancials bool
	HasIndirectCostRate  bool
	IndirectCostRatePct  *float64
	PriorFederalGrants   bool
	Narrative            *string
	Embedding            []float32
}

// --- User ---

type UserRepo interface {
	Create(ctx context.Context, params CreateUserParams) (*domain.User, error)
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByGoogleSub(ctx context.Context, sub string) (*domain.User, error)
	Update(ctx context.Context, params UpdateUserParams) (*domain.User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role domain.UserRole) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
	AttachGoogleSub(ctx context.Context, id uuid.UUID, sub string) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) error
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, ttl int) error
	GetPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error
	AddOrgMember(ctx context.Context, orgID, userID uuid.UUID, role domain.OrgMemberRole, invitedBy *uuid.UUID) (*domain.OrganizationMember, error)
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (*domain.OrganizationMember, error)
	ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]*OrgMemberWithUser, error)
	ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*OrgWithRole, error)
	RemoveOrgMember(ctx context.Context, orgID, userID uuid.UUID) error
	SaveQuizDraft(ctx context.Context, userID uuid.UUID, data []byte) error
	GetQuizDraft(ctx context.Context, userID uuid.UUID) ([]byte, error)
	DeleteQuizDraft(ctx context.Context, userID uuid.UUID) error
}

type CreateUserParams struct {
	Email        string
	PasswordHash *string
	FirstName    *string
	LastName     *string
	Phone        *string
	GoogleSub    *string
	Role         domain.UserRole
	AuthProvider domain.AuthProvider
}

type UpdateUserParams struct {
	ID        uuid.UUID
	FirstName *string
	LastName  *string
	Phone     *string
	AvatarURL *string
}

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
}

type OrgMemberWithUser struct {
	domain.OrganizationMember
	Email     string
	FirstName *string
	LastName  *string
	AvatarURL *string
}

type OrgWithRole struct {
	domain.Organization
	Role domain.OrgMemberRole
}

// --- Grant ---

type GrantRepo interface {
	Create(ctx context.Context, params CreateGrantParams) (*domain.Grant, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Grant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Grant, error)
	List(ctx context.Context, statuses []string, limit, offset int32) ([]*domain.Grant, error)
	ListByCategory(ctx context.Context, category string, limit, offset int32) ([]*domain.Grant, error)
	Search(ctx context.Context, query string, limit, offset int32) ([]*domain.Grant, error)
	Update(ctx context.Context, params UpdateGrantParams) (*domain.Grant, error)
	UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float32) error
	UpdateNOFO(ctx context.Context, id uuid.UUID, text string) error
	Archive(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
	SearchBySimilarity(ctx context.Context, embedding []float32, limit int32) ([]*GrantWithDistance, error)
	UpsertNOFOChunk(ctx context.Context, params UpsertChunkParams) (*domain.NOFOChunk, error)
	ListNOFOChunks(ctx context.Context, grantID uuid.UUID) ([]*domain.NOFOChunk, error)
	SearchChunksBySimilarity(ctx context.Context, grantID uuid.UUID, embedding []float32, limit int32) ([]*ChunkWithDistance, error)
	UpsertScoringCriterion(ctx context.Context, params UpsertCriterionParams) (*domain.GrantScoringCriterion, error)
	ListScoringCriteria(ctx context.Context, grantID uuid.UUID) ([]*domain.GrantScoringCriterion, error)
}

type CreateGrantParams struct {
	Slug                  string
	Title                 string
	FunderName            string
	FunderType            domain.FunderType
	ProgramNumber         *string
	OpportunityNumber     *string
	Agency                *string
	Description           *string
	Synopsis              *string
	Category              *string
	FocusAreas            []string
	EligibleOrgTypes      []string
	EligiblePopulations   []string
	EligibleStates        []string
	Requires501c3         bool
	RequiresAuditedFin    bool
	RequiresMatch         bool
	MatchPercentage       *float64
	MinAwardAmount        *int64
	MaxAwardAmount        *int64
	TotalFundingAvailable *int64
	ApplicationURL        *string
	Status                domain.GrantStatus
	Deadline              *string
	OpenDate              *string
	DifficultyLevel       domain.DifficultyLevel
	CompetitionLevel      domain.CompetitionLevel
	Tags                  []string
	Metadata              map[string]interface{}
	CreatedBy             *uuid.UUID
}

type UpdateGrantParams struct {
	ID                    uuid.UUID
	Title                 *string
	FunderName            *string
	FunderType            *domain.FunderType
	Agency                *string
	Description           *string
	Synopsis              *string
	Category              *string
	FocusAreas            []string
	EligibleOrgTypes      []string
	EligibleStates        []string
	MinAwardAmount        *int64
	MaxAwardAmount        *int64
	TotalFundingAvailable *int64
	ApplicationURL        *string
	Status                *domain.GrantStatus
	Deadline              *string
	OpenDate              *string
	DifficultyLevel       *domain.DifficultyLevel
	CompetitionLevel      *domain.CompetitionLevel
	Tags                  []string
	Metadata              map[string]interface{}
}

type GrantWithDistance struct {
	domain.Grant
	Distance float64
}

type UpsertChunkParams struct {
	GrantID    uuid.UUID
	ChunkIndex int32
	Section    *string
	Content    string
	TokenCount *int32
	Embedding  []float32
}

type ChunkWithDistance struct {
	domain.NOFOChunk
	Distance float64
}

type UpsertCriterionParams struct {
	GrantID       uuid.UUID
	CriterionKey  string
	Weight        float64
	IsRequired    bool
	Disqualifying bool
	Description   *string
}

// --- Scoring ---

type ScoringRepo interface {
	Upsert(ctx context.Context, params UpsertScoreParams) (*domain.CompatibilityScore, error)
	Get(ctx context.Context, orgID, grantID uuid.UUID) (*domain.CompatibilityScore, error)
	ListTopGrantsForOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]*ScoredGrant, error)
	ListOrgsForGrant(ctx context.Context, grantID uuid.UUID, limit, offset int32) ([]*ScoredOrg, error)
	DeleteForOrg(ctx context.Context, orgID uuid.UUID) error
	DeleteForGrant(ctx context.Context, grantID uuid.UUID) error
}

type UpsertScoreParams struct {
	OrgID             uuid.UUID
	GrantID           uuid.UUID
	TotalScore        float64
	Tier              domain.CompatibilityTier
	DimensionScores   []domain.DimensionScore
	Disqualified      bool
	DisqualifyReasons []string
	Strengths         []string
	Gaps              []string
	Recommendations   []string
	SemanticScore     *float64
	EngineVersion     string
}

type ScoredGrant struct {
	domain.CompatibilityScore
	GrantTitle  string
	FunderName  string
	Category    *string
	Deadline    *string
	MinAward    *int64
	MaxAward    *int64
	GrantStatus string
}

type ScoredOrg struct {
	domain.CompatibilityScore
	OrgName string
	State   *string
	OrgType string
}

// --- Application ---

type ApplicationRepo interface {
	Create(ctx context.Context, params CreateApplicationParams) (*domain.GrantApplication, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.GrantApplication, error)
	GetByOrgAndGrant(ctx context.Context, orgID, grantID uuid.UUID) (*domain.GrantApplication, error)
	ListForOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]*domain.GrantApplication, error)
	ListByStatus(ctx context.Context, status string, limit, offset int32) ([]*domain.GrantApplication, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ApplicationStatus, stage *domain.ApplicationStage) (*domain.GrantApplication, error)
	Update(ctx context.Context, params UpdateApplicationParams) (*domain.GrantApplication, error)
	LogActivity(ctx context.Context, params LogActivityParams) (*domain.ApplicationActivity, error)
	ListActivities(ctx context.Context, applicationID uuid.UUID) ([]*domain.ApplicationActivity, error)
	CreateNarrative(ctx context.Context, params CreateNarrativeParams) (*domain.AINarrative, error)
	ListNarrativesForApplication(ctx context.Context, applicationID uuid.UUID) ([]*domain.AINarrative, error)
	GetLatestNarrativeBySection(ctx context.Context, applicationID uuid.UUID, section domain.NarrativeSection) (*domain.AINarrative, error)
	Count(ctx context.Context, orgID uuid.UUID) (int64, error)
}

type CreateApplicationParams struct {
	OrgID              uuid.UUID
	GrantID            uuid.UUID
	AssignedTo         *uuid.UUID
	Status             domain.ApplicationStatus
	Stage              domain.ApplicationStage
	Priority           domain.ApplicationPriority
	CompatibilityScore *float64
	Notes              *string
	InternalDeadline   *string
}

type UpdateApplicationParams struct {
	ID               uuid.UUID
	AssignedTo       *uuid.UUID
	Status           *domain.ApplicationStatus
	Stage            *domain.ApplicationStage
	Priority         *domain.ApplicationPriority
	Notes            *string
	InternalDeadline *string
	SubmissionDate   *string
	AwardAmount      *int64
}

type LogActivityParams struct {
	ApplicationID uuid.UUID
	UserID        *uuid.UUID
	ActivityType  string
	OldValue      *string
	NewValue      *string
	Note          *string
}

type CreateNarrativeParams struct {
	OrgID         uuid.UUID
	GrantID       uuid.UUID
	ApplicationID *uuid.UUID
	SectionKey    domain.NarrativeSection
	PromptUsed    *string
	Content       string
	WordCount     *int32
	ModelUsed     string
	TokensIn      *int32
	TokensOut     *int32
}

// --- Lead ---

type LeadRepo interface {
	Create(ctx context.Context, params CreateLeadParams) (*domain.Lead, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Lead, error)
	GetByEmail(ctx context.Context, email string) (*domain.Lead, error)
	List(ctx context.Context, limit, offset int32) ([]*domain.Lead, error)
	ListByStatus(ctx context.Context, status domain.LeadStatus, limit, offset int32) ([]*domain.Lead, error)
	Update(ctx context.Context, params UpdateLeadParams) (*domain.Lead, error)
	Convert(ctx context.Context, id, orgID uuid.UUID) error
	LogActivity(ctx context.Context, params LogLeadActivityParams) (*domain.LeadActivity, error)
	ListActivities(ctx context.Context, leadID uuid.UUID) ([]*domain.LeadActivity, error)
	Count(ctx context.Context) (int64, error)
	Search(ctx context.Context, query string, limit, offset int32) ([]*domain.Lead, error)
}

type CreateLeadParams struct {
	Email            string
	FirstName        *string
	LastName         *string
	OrgName          *string
	OrgType          *string
	Phone            *string
	City             *string
	State            *string
	Zip              *string
	Source           domain.LeadSource
	UTMSource        *string
	UTMMedium        *string
	UTMCampaign      *string
	QuizResponses    map[string]interface{}
	InterestedGrants []string
}

type UpdateLeadParams struct {
	ID              uuid.UUID
	FirstName       *string
	LastName        *string
	OrgName         *string
	Phone           *string
	Status          *domain.LeadStatus
	Score           *int32
	AssignedTo      *uuid.UUID
	Notes           *string
	LastContactedAt *string
}

type LogLeadActivityParams struct {
	LeadID       uuid.UUID
	UserID       *uuid.UUID
	ActivityType string
	Subject      *string
	Body         *string
	OldValue     *string
	NewValue     *string
	ScheduledAt  *string
}

// --- Analytics ---

type AnalyticsRepo interface {
	LogEvent(ctx context.Context, params LogEventParams) error
	LogAIUsage(ctx context.Context, params LogAIUsageParams) error
	GetPlatformStats(ctx context.Context) (*PlatformStats, error)
}

type LogEventParams struct {
	OrgID      *uuid.UUID
	UserID     *uuid.UUID
	LeadID     *uuid.UUID
	EventType  string
	EntityType *string
	EntityID   *uuid.UUID
	Properties map[string]interface{}
	IPAddress  *string
	UserAgent  *string
}

type LogAIUsageParams struct {
	OrgID        *uuid.UUID
	UserID       *uuid.UUID
	Operation    string
	Model        string
	TokensIn     int32
	TokensOut    int32
	CostUSDCents int32
	LatencyMS    *int32
	Success      bool
	ErrorMessage *string
}

type PlatformStats struct {
	TotalOrgs         int64 `json:"total_orgs"`
	TotalUsers        int64 `json:"total_users"`
	ActiveGrants      int64 `json:"active_grants"`
	TotalLeads        int64 `json:"total_leads"`
	TotalApplications int64 `json:"total_applications"`
}

// --- Products (Funding OS solutions catalog) ---

type ProductRepo interface {
	List(ctx context.Context) ([]*domain.Product, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	Create(ctx context.Context, params CreateProductParams) (*domain.Product, error)

	SaveSelection(ctx context.Context, params SaveSelectionParams) (*domain.OrgProductSelection, error)
	ListSelections(ctx context.Context, orgID uuid.UUID) ([]*domain.OrgProductSelection, error)
	DeleteSelection(ctx context.Context, orgID, productID uuid.UUID) error
}

type CreateProductParams struct {
	Slug             string
	Name             string
	Category         *string
	ShortDesc        *string
	Description      *string
	SelectionType    *string
	PriceCents       int64
	PriceType        string
	Featured         bool
	FundingAlignment []string
	Catalog          []byte
	SortOrder        int32
}

type SaveSelectionParams struct {
	OrgID           uuid.UUID
	ProductID       uuid.UUID
	ConfigurationID *string
	SelectedAddons  []string
	Quantity        int32
	UnitPriceCents  int64
	SubtotalCents   int64
}
