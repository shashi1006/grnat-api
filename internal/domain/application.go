package domain

import (
	"time"

	"github.com/google/uuid"
)

// ApplicationStatus enumerates grant application pipeline stages.
type ApplicationStatus string

const (
	AppStatusProspect       ApplicationStatus = "prospect"
	AppStatusResearching    ApplicationStatus = "researching"
	AppStatusDrafting       ApplicationStatus = "drafting"
	AppStatusInternalReview ApplicationStatus = "internal_review"
	AppStatusSubmitted      ApplicationStatus = "submitted"
	AppStatusAwarded        ApplicationStatus = "awarded"
	AppStatusRejected       ApplicationStatus = "rejected"
	AppStatusWithdrawn      ApplicationStatus = "withdrawn"
)

// ApplicationStage enumerates broad phases.
type ApplicationStage string

const (
	StagePreApplication ApplicationStage = "pre_application"
	StageApplication    ApplicationStage = "application"
	StagePostSubmission ApplicationStage = "post_submission"
)

// ApplicationPriority enumerates urgency levels.
type ApplicationPriority string

const (
	PriorityLow      ApplicationPriority = "low"
	PriorityMedium   ApplicationPriority = "medium"
	PriorityHigh     ApplicationPriority = "high"
	PriorityCritical ApplicationPriority = "critical"
)

// GrantApplication tracks an org's pursuit of a specific grant.
type GrantApplication struct {
	ID                 uuid.UUID           `json:"id"`
	OrgID              uuid.UUID           `json:"org_id"`
	GrantID            uuid.UUID           `json:"grant_id"`
	AssignedTo         *uuid.UUID          `json:"assigned_to,omitempty"`
	CreatedBy          *uuid.UUID          `json:"created_by,omitempty"`
	Status             ApplicationStatus   `json:"status"`
	Stage              ApplicationStage    `json:"stage"`
	Priority           ApplicationPriority `json:"priority"`
	CompatibilityScore *float64            `json:"compatibility_score,omitempty"`
	SubmissionDate     *time.Time          `json:"submission_date,omitempty"`
	AwardAmount        *int64              `json:"award_amount,omitempty"`
	AwardDate          *time.Time          `json:"award_date,omitempty"`
	RejectionReason    *string             `json:"rejection_reason,omitempty"`
	Notes              *string             `json:"notes,omitempty"`
	InternalDeadline   *time.Time          `json:"internal_deadline,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`

	// Joined fields
	GrantTitle     *string    `json:"grant_title,omitempty"`
	FunderName     *string    `json:"funder_name,omitempty"`
	GrantDeadline  *time.Time `json:"grant_deadline,omitempty"`
	OrgName        *string    `json:"org_name,omitempty"`
	CreatedByEmail *string    `json:"created_by_email,omitempty"`
}

// ApplicationActivity is an audit log entry for an application.
type ApplicationActivity struct {
	ID            uuid.UUID  `json:"id"`
	ApplicationID uuid.UUID  `json:"application_id"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`
	ActivityType  string     `json:"activity_type"`
	OldValue      *string    `json:"old_value,omitempty"`
	NewValue      *string    `json:"new_value,omitempty"`
	Note          *string    `json:"note,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// NarrativeSection enumerates the sections Claude can generate.
type NarrativeSection string

const (
	SectionNeedStatement      NarrativeSection = "need_statement"
	SectionProjectDescription NarrativeSection = "project_description"
	SectionGoalsObjectives    NarrativeSection = "goals_objectives"
	SectionEvalPlan           NarrativeSection = "evaluation_plan"
	SectionOrgCapacity        NarrativeSection = "org_capacity"
	SectionBudgetNarrative    NarrativeSection = "budget_narrative"
	SectionExecutiveSummary   NarrativeSection = "executive_summary"
)

// AINarrative stores a Claude-generated narrative section.
type AINarrative struct {
	ID            uuid.UUID        `json:"id"`
	OrgID         uuid.UUID        `json:"org_id"`
	GrantID       uuid.UUID        `json:"grant_id"`
	ApplicationID *uuid.UUID       `json:"application_id,omitempty"`
	SectionKey    NarrativeSection `json:"section_key"`
	PromptUsed    *string          `json:"prompt_used,omitempty"`
	Content       string           `json:"content"`
	WordCount     *int32           `json:"word_count,omitempty"`
	ModelUsed     string           `json:"model_used"`
	TokensIn      *int32           `json:"tokens_in,omitempty"`
	TokensOut     *int32           `json:"tokens_out,omitempty"`
	IsApproved    bool             `json:"is_approved"`
	Version       int32            `json:"version"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}
