package domain

import (
	"time"

	"github.com/google/uuid"
)

// FunderType enumerates grant funding sources.
type FunderType string

const (
	FunderFederal    FunderType = "federal"
	FunderState      FunderType = "state"
	FunderLocal      FunderType = "local"
	FunderPrivate    FunderType = "private"
	FunderFoundation FunderType = "foundation"
	FunderCorporate  FunderType = "corporate"
)

// GrantStatus enumerates the lifecycle of a grant listing.
type GrantStatus string

const (
	GrantStatusActive   GrantStatus = "active"
	GrantStatusClosed   GrantStatus = "closed"
	GrantStatusArchived GrantStatus = "archived"
	GrantStatusDraft    GrantStatus = "draft"
)

// DifficultyLevel enumerates application effort levels.
type DifficultyLevel string

const (
	DifficultyEasy     DifficultyLevel = "easy"
	DifficultyMedium   DifficultyLevel = "medium"
	DifficultyHard     DifficultyLevel = "hard"
	DifficultyVeryHard DifficultyLevel = "very_hard"
)

// CompetitionLevel enumerates how competitive a grant is.
type CompetitionLevel string

const (
	CompetitionLow      CompetitionLevel = "low"
	CompetitionMedium   CompetitionLevel = "medium"
	CompetitionHigh     CompetitionLevel = "high"
	CompetitionVeryHigh CompetitionLevel = "very_high"
)

// SubmissionPathway describes how an application reaches the funder.
//   - direct: the applicant organization submits to the funder itself
//   - pass_through: only an administering agency (e.g. a state SAA) may
//     submit; other organizations can only receive subawards
//   - mixed: both direct and pass-through tracks exist (e.g. JAG)
type SubmissionPathway string

const (
	PathwayDirect      SubmissionPathway = "direct"
	PathwayPassThrough SubmissionPathway = "pass_through"
	PathwayMixed       SubmissionPathway = "mixed"
)

// ExtractedRequirements is the structured requirement set an LLM pulls out of
// NOFO text. EligibleApplicants lists the entity types that may SUBMIT an
// application (snake_case labels, e.g. "state-administrative-agency"), as
// distinct from orgs that may only benefit via subawards.
type ExtractedRequirements struct {
	EligibleApplicants []string `json:"eligible_applicants"`
	SubmissionPathway  string   `json:"submission_pathway"` // direct | pass_through | mixed
	PassThroughNote    string   `json:"pass_through_note"`
	RequiredForms      []string `json:"required_forms"`
	NarrativeSections  []string `json:"narrative_sections"`
	SetAsides          []string `json:"set_asides"`
	Certifications     []string `json:"certifications"`
	AwardConstraints   string   `json:"award_constraints"`
	EligibleOrgTypes   []string `json:"eligible_org_types"`
	EligibleStates     []string `json:"eligible_states"`
	MinAwardDollars    *int64   `json:"min_award_dollars"`
	MaxAwardDollars    *int64   `json:"max_award_dollars"`
}

// SubmissionRequirementsMap returns the fields that belong in
// grants.submission_requirements (everything except the top-level columns).
func (r *ExtractedRequirements) SubmissionRequirementsMap() map[string]interface{} {
	return map[string]interface{}{
		"required_forms":     r.RequiredForms,
		"narrative_sections": r.NarrativeSections,
		"set_asides":         r.SetAsides,
		"certifications":     r.Certifications,
		"award_constraints":  r.AwardConstraints,
	}
}

// Grant represents a funding opportunity in the catalog.
type Grant struct {
	ID                      uuid.UUID              `json:"id"`
	Slug                    string                 `json:"slug"`
	Title                   string                 `json:"title"`
	FunderName              string                 `json:"funder_name"`
	FunderType              FunderType             `json:"funder_type"`
	ProgramNumber           *string                `json:"program_number,omitempty"`
	OpportunityNumber       *string                `json:"opportunity_number,omitempty"`
	Agency                  *string                `json:"agency,omitempty"`
	SubAgency               *string                `json:"sub_agency,omitempty"`
	Description             *string                `json:"description,omitempty"`
	Synopsis                *string                `json:"synopsis,omitempty"`
	Category                *string                `json:"category,omitempty"`
	Subcategory             *string                `json:"subcategory,omitempty"`
	FocusAreas              []string               `json:"focus_areas"`
	EligibleOrgTypes        []string               `json:"eligible_org_types"`
	EligiblePopulations     []string               `json:"eligible_populations"`
	EligibleStates          []string               `json:"eligible_states"`
	EligibleApplicants      []string               `json:"eligible_applicants"`
	SubmissionPathway       SubmissionPathway      `json:"submission_pathway"`
	PassThroughNote         *string                `json:"pass_through_note,omitempty"`
	SubmissionRequirements  map[string]interface{} `json:"submission_requirements"`
	RequirementsExtractedAt *time.Time             `json:"requirements_extracted_at,omitempty"`
	Requires501c3           bool                   `json:"requires_501c3"`
	RequiresAuditedFin      bool                   `json:"requires_audited_fin"`
	RequiresIndirectRate    bool                   `json:"requires_indirect_rate"`
	RequiresMatch           bool                   `json:"requires_match"`
	MatchPercentage         *float64               `json:"match_percentage,omitempty"`
	MinAwardAmount          *int64                 `json:"min_award_amount,omitempty"`
	MaxAwardAmount          *int64                 `json:"max_award_amount,omitempty"`
	AvgAwardAmount          *int64                 `json:"avg_award_amount,omitempty"`
	TotalFundingAvailable   *int64                 `json:"total_funding_available,omitempty"`
	NumAwardsExpected       *int32                 `json:"num_awards_expected,omitempty"`
	ApplicationURL          *string                `json:"application_url,omitempty"`
	FAQURL                  *string                `json:"faq_url,omitempty"`
	WebinarURL              *string                `json:"webinar_url,omitempty"`
	FullNOFOText            *string                `json:"-"`
	EligibleCounties        []string               `json:"eligible_counties"`
	Status                  GrantStatus            `json:"status"`
	Deadline                *time.Time             `json:"deadline,omitempty"`
	OpenDate                *time.Time             `json:"open_date,omitempty"`
	PeriodOfPerformance     *string                `json:"period_of_performance,omitempty"`
	IsRecurring             bool                   `json:"is_recurring"`
	RecurrenceNotes         *string                `json:"recurrence_notes,omitempty"`
	DifficultyLevel         DifficultyLevel        `json:"difficulty_level"`
	CompetitionLevel        CompetitionLevel       `json:"competition_level"`
	Tags                    []string               `json:"tags"`
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy               *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
}

// NOFOChunk is a text chunk from a NOFO document, stored for RAG.
type NOFOChunk struct {
	ID         uuid.UUID `json:"id"`
	GrantID    uuid.UUID `json:"grant_id"`
	ChunkIndex int32     `json:"chunk_index"`
	Section    *string   `json:"section,omitempty"`
	Content    string    `json:"content"`
	TokenCount *int32    `json:"token_count,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// GrantScoringCriterion defines a weighted scoring rule for a grant.
type GrantScoringCriterion struct {
	ID            uuid.UUID `json:"id"`
	GrantID       uuid.UUID `json:"grant_id"`
	CriterionKey  string    `json:"criterion_key"`
	Weight        float64   `json:"weight"`
	IsRequired    bool      `json:"is_required"`
	Disqualifying bool      `json:"disqualifying"`
	Description   *string   `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
