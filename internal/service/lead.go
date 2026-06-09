package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// LeadService manages the CRM lead pipeline.
type LeadService struct {
	leads repository.LeadRepo
}

// NewLeadService creates a LeadService.
func NewLeadService(leads repository.LeadRepo) *LeadService {
	return &LeadService{leads: leads}
}

// Create captures a new lead (e.g. from quiz or landing page).
func (s *LeadService) Create(ctx context.Context, params repository.CreateLeadParams) (*domain.Lead, error) {
	// Deduplicate by email
	existing, _ := s.leads.GetByEmail(ctx, params.Email)
	if existing != nil {
		return existing, nil
	}
	lead, err := s.leads.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create lead: %w", err)
	}
	return lead, nil
}

// GetByID retrieves a lead.
func (s *LeadService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lead, error) {
	lead, err := s.leads.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("lead not found: %w", err)
	}
	return lead, nil
}

// List returns all leads with pagination.
func (s *LeadService) List(ctx context.Context, limit, offset int32) ([]*domain.Lead, error) {
	return s.leads.List(ctx, limit, offset)
}

// ListByStatus filters leads by CRM status.
func (s *LeadService) ListByStatus(ctx context.Context, status domain.LeadStatus, limit, offset int32) ([]*domain.Lead, error) {
	return s.leads.ListByStatus(ctx, status, limit, offset)
}

// Update modifies a lead record.
func (s *LeadService) Update(ctx context.Context, params repository.UpdateLeadParams) (*domain.Lead, error) {
	lead, err := s.leads.Update(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("update lead: %w", err)
	}
	return lead, nil
}

// Convert marks a lead as converted and links it to the new organization.
func (s *LeadService) Convert(ctx context.Context, id, orgID uuid.UUID) error {
	if err := s.leads.Convert(ctx, id, orgID); err != nil {
		return fmt.Errorf("convert lead: %w", err)
	}
	_, _ = s.leads.LogActivity(ctx, repository.LogLeadActivityParams{
		LeadID:       id,
		ActivityType: "converted",
		NewValue:     func() *string { v := orgID.String(); return &v }(),
	})
	return nil
}

// LogActivity records a CRM touch on a lead.
func (s *LeadService) LogActivity(ctx context.Context, params repository.LogLeadActivityParams) (*domain.LeadActivity, error) {
	return s.leads.LogActivity(ctx, params)
}

// ListActivities returns the activity timeline for a lead.
func (s *LeadService) ListActivities(ctx context.Context, leadID uuid.UUID) ([]*domain.LeadActivity, error) {
	return s.leads.ListActivities(ctx, leadID)
}

// Search finds leads matching a query string.
func (s *LeadService) Search(ctx context.Context, query string, limit, offset int32) ([]*domain.Lead, error) {
	return s.leads.Search(ctx, query, limit, offset)
}

// Count returns the total number of leads.
func (s *LeadService) Count(ctx context.Context) (int64, error) {
	return s.leads.Count(ctx)
}
