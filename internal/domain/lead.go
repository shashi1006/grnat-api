package domain

import (
	"time"

	"github.com/google/uuid"
)

// LeadStatus enumerates CRM pipeline stages.
type LeadStatus string

const (
	LeadStatusNew         LeadStatus = "new"
	LeadStatusContacted   LeadStatus = "contacted"
	LeadStatusQualified   LeadStatus = "qualified"
	LeadStatusUnqualified LeadStatus = "unqualified"
	LeadStatusConverted   LeadStatus = "converted"
	LeadStatusChurned     LeadStatus = "churned"
)

// LeadSource enumerates how the lead was acquired.
type LeadSource string

const (
	LeadSourceOrganic  LeadSource = "organic"
	LeadSourceReferral LeadSource = "referral"
	LeadSourceAd       LeadSource = "ad"
	LeadSourceWebinar  LeadSource = "webinar"
	LeadSourceDemo     LeadSource = "demo"
)

// Lead represents a prospective organization/user.
type Lead struct {
	ID               uuid.UUID              `json:"id"`
	Email            string                 `json:"email"`
	FirstName        *string                `json:"first_name,omitempty"`
	LastName         *string                `json:"last_name,omitempty"`
	OrgName          *string                `json:"org_name,omitempty"`
	OrgType          *string                `json:"org_type,omitempty"`
	Phone            *string                `json:"phone,omitempty"`
	City             *string                `json:"city,omitempty"`
	State            *string                `json:"state,omitempty"`
	Zip              *string                `json:"zip,omitempty"`
	Source           LeadSource             `json:"source"`
	UTMSource        *string                `json:"utm_source,omitempty"`
	UTMMedium        *string                `json:"utm_medium,omitempty"`
	UTMCampaign      *string                `json:"utm_campaign,omitempty"`
	Status           LeadStatus             `json:"status"`
	Score            int32                  `json:"score"`
	AssignedTo       *uuid.UUID             `json:"assigned_to,omitempty"`
	ConvertedOrgID   *uuid.UUID             `json:"converted_org_id,omitempty"`
	ConvertedAt      *time.Time             `json:"converted_at,omitempty"`
	LastContactedAt  *time.Time             `json:"last_contacted_at,omitempty"`
	Notes            *string                `json:"notes,omitempty"`
	QuizResponses    map[string]interface{} `json:"quiz_responses"`
	InterestedGrants []string               `json:"interested_grants"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// LeadActivity is an event logged against a lead.
type LeadActivity struct {
	ID           uuid.UUID  `json:"id"`
	LeadID       uuid.UUID  `json:"lead_id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	ActivityType string     `json:"activity_type"`
	Subject      *string    `json:"subject,omitempty"`
	Body         *string    `json:"body,omitempty"`
	OldValue     *string    `json:"old_value,omitempty"`
	NewValue     *string    `json:"new_value,omitempty"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
