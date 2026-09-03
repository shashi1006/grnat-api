package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/ai/embedding"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// OrgService handles organization and profile management.
type OrgService struct {
	orgs     repository.OrganizationRepo
	embedSvc *embedding.Service
}

// NewOrgService creates an OrgService.
func NewOrgService(orgs repository.OrganizationRepo, embedSvc *embedding.Service) *OrgService {
	return &OrgService{orgs: orgs, embedSvc: embedSvc}
}

// CreateOrg creates an organization and kicks off embedding generation.
func (s *OrgService) CreateOrg(ctx context.Context, params repository.CreateOrgParams) (*domain.Organization, error) {
	org, err := s.orgs.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}

	// Create a default empty profile so scoring works immediately
	emptyStr := ""
	_, _ = s.orgs.UpsertProfile(ctx, repository.UpsertProfileParams{
		OrgID:             org.ID,
		PopulationsServed: []string{},
		ServiceAreas:      []string{},
		ProgramAreas:      []string{},
		FocusIssues:       []string{},
		Narrative:         &emptyStr,
	})

	// Generate embedding best-effort in background
	go func() {
		text := org.Name
		if org.Mission != nil {
			text += " " + *org.Mission
		}
		emb, err := s.embedSvc.Embed(context.Background(), text)
		if err == nil {
			_ = s.orgs.UpdateProfileEmbedding(context.Background(), org.ID, emb)
		}
	}()

	return org, nil
}

// ListOrgs returns a paginated list of all organizations.
func (s *OrgService) ListOrgs(ctx context.Context, limit, offset int32) ([]*domain.Organization, error) {
	return s.orgs.List(ctx, limit, offset)
}

// GetOrg retrieves an org by ID.
func (s *OrgService) GetOrg(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	org, err := s.orgs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("org not found: %w", err)
	}
	return org, nil
}

// GetOrgBySlug retrieves an org by its slug.
func (s *OrgService) GetOrgBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	return s.orgs.GetBySlug(ctx, slug)
}

// UpdateOrg updates mutable org fields.
func (s *OrgService) UpdateOrg(ctx context.Context, params repository.UpdateOrgParams) (*domain.Organization, error) {
	org, err := s.orgs.Update(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("update org: %w", err)
	}

	// Re-embed after update
	go func() {
		text := org.Name
		if org.Mission != nil {
			text += " " + *org.Mission
		}
		emb, err := s.embedSvc.Embed(context.Background(), text)
		if err == nil {
			_ = s.orgs.UpdateProfileEmbedding(context.Background(), org.ID, emb)
		}
	}()

	return org, nil
}

// GetProfile retrieves the detailed org profile.
func (s *OrgService) GetProfile(ctx context.Context, orgID uuid.UUID) (*domain.OrganizationProfile, error) {
	return s.orgs.GetProfile(ctx, orgID)
}

// UpsertProfile creates or updates an org profile.
func (s *OrgService) UpsertProfile(ctx context.Context, params repository.UpsertProfileParams) (*domain.OrganizationProfile, error) {
	return s.orgs.UpsertProfile(ctx, params)
}
