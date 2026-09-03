package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// ApplicationService manages the grant application pipeline.
type ApplicationService struct {
	apps    repository.ApplicationRepo
	grants  repository.GrantRepo
	scoring repository.ScoringRepo
}

// NewApplicationService creates an ApplicationService.
func NewApplicationService(
	apps repository.ApplicationRepo,
	grants repository.GrantRepo,
	scoring repository.ScoringRepo,
) *ApplicationService {
	return &ApplicationService{apps: apps, grants: grants, scoring: scoring}
}

// Create starts tracking a grant application. Pulls the latest compatibility
// score if one exists and attaches it.
func (s *ApplicationService) Create(ctx context.Context, params repository.CreateApplicationParams) (*domain.GrantApplication, error) {
	// Attach latest compatibility score if available
	if params.CompatibilityScore == nil {
		score, err := s.scoring.Get(ctx, params.OrgID, params.GrantID)
		if err == nil && score != nil {
			params.CompatibilityScore = &score.TotalScore
		}
	}
	app, err := s.apps.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}

	// Log creation activity
	_, _ = s.apps.LogActivity(ctx, repository.LogActivityParams{
		ApplicationID: app.ID,
		ActivityType:  "created",
		NewValue:      strPtr(string(app.Status)),
	})

	return app, nil
}

// GetByID retrieves an application with joined grant fields.
func (s *ApplicationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.GrantApplication, error) {
	app, err := s.apps.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}
	return app, nil
}

// List returns all applications across all organizations.
func (s *ApplicationService) List(ctx context.Context, limit, offset int32) ([]*domain.GrantApplication, error) {
	return s.apps.List(ctx, limit, offset)
}

// ListForOrg returns all applications for an organization.
func (s *ApplicationService) ListForOrg(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]*domain.GrantApplication, error) {
	return s.apps.ListForOrg(ctx, orgID, limit, offset)
}

// UpdateStatus advances the pipeline status and logs an activity entry.
func (s *ApplicationService) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.ApplicationStatus,
	stage *domain.ApplicationStage,
	userID *uuid.UUID,
) (*domain.GrantApplication, error) {
	// Fetch old status for activity log
	old, _ := s.apps.GetByID(ctx, id)

	app, err := s.apps.UpdateStatus(ctx, id, status, stage)
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	var oldVal *string
	if old != nil {
		oldVal = strPtr(string(old.Status))
	}
	_, _ = s.apps.LogActivity(ctx, repository.LogActivityParams{
		ApplicationID: id,
		UserID:        userID,
		ActivityType:  "status_change",
		OldValue:      oldVal,
		NewValue:      strPtr(string(status)),
	})

	return app, nil
}

// Update modifies application fields.
func (s *ApplicationService) Update(ctx context.Context, params repository.UpdateApplicationParams) (*domain.GrantApplication, error) {
	app, err := s.apps.Update(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}
	return app, nil
}

// ListActivities returns the audit trail for an application.
func (s *ApplicationService) ListActivities(ctx context.Context, applicationID uuid.UUID) ([]*domain.ApplicationActivity, error) {
	return s.apps.ListActivities(ctx, applicationID)
}

// ListNarratives returns all AI narratives for an application.
func (s *ApplicationService) ListNarratives(ctx context.Context, applicationID uuid.UUID) ([]*domain.AINarrative, error) {
	return s.apps.ListNarrativesForApplication(ctx, applicationID)
}

// Count returns the total number of applications for an org.
func (s *ApplicationService) Count(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return s.apps.Count(ctx, orgID)
}

func strPtr(s string) *string { return &s }
