package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/ai/embedding"
	"github.com/readygeneration/readygeneration-backend/internal/ai/rag"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// GrantService handles grant catalog management, NOFO ingestion, and semantic search.
type GrantService struct {
	grants    repository.GrantRepo
	embedSvc  *embedding.Service
	ragEngine *rag.Engine
}

// NewGrantService creates a GrantService.
func NewGrantService(grants repository.GrantRepo, embedSvc *embedding.Service, ragEngine *rag.Engine) *GrantService {
	return &GrantService{grants: grants, embedSvc: embedSvc, ragEngine: ragEngine}
}

// ListGrants returns a paginated list of active grants.
func (s *GrantService) ListGrants(ctx context.Context, statuses []string, limit, offset int32) ([]*domain.Grant, error) {
	return s.grants.List(ctx, statuses, limit, offset)
}

// GetGrant returns a single grant by ID.
func (s *GrantService) GetGrant(ctx context.Context, id uuid.UUID) (*domain.Grant, error) {
	g, err := s.grants.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("grant not found: %w", err)
	}
	return g, nil
}

// SearchGrants performs a title/keyword search.
func (s *GrantService) SearchGrants(ctx context.Context, query string, limit, offset int32) ([]*domain.Grant, error) {
	return s.grants.Search(ctx, query, limit, offset)
}

// SemanticSearchGrants finds grants by semantic similarity to a free-text description.
func (s *GrantService) SemanticSearchGrants(ctx context.Context, query string, topK int32) ([]*repository.GrantWithDistance, error) {
	emb, err := s.embedSvc.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	return s.grants.SearchBySimilarity(ctx, emb, topK)
}

// CreateGrant adds a new grant to the catalog.
func (s *GrantService) CreateGrant(ctx context.Context, params repository.CreateGrantParams) (*domain.Grant, error) {
	g, err := s.grants.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create grant: %w", err)
	}

	// Generate and store embedding asynchronously (best-effort).
	go func() {
		text := g.Title
		if g.Description != nil {
			text += " " + *g.Description
		}
		if g.Synopsis != nil {
			text += " " + *g.Synopsis
		}
		emb, err := s.embedSvc.Embed(context.Background(), text)
		if err == nil {
			_ = s.grants.UpdateEmbedding(context.Background(), g.ID, emb)
		}
	}()

	return g, nil
}

// UpdateGrant updates mutable grant fields.
func (s *GrantService) UpdateGrant(ctx context.Context, params repository.UpdateGrantParams) (*domain.Grant, error) {
	return s.grants.Update(ctx, params)
}

// IngestNOFO stores and chunks a raw NOFO document, generating pgvector embeddings.
func (s *GrantService) IngestNOFO(ctx context.Context, grantID uuid.UUID, nofoText string) error {
	return s.ragEngine.IngestNOFO(ctx, grantID, nofoText)
}

// QueryNOFO retrieves relevant NOFO passages for RAG context.
func (s *GrantService) QueryNOFO(ctx context.Context, grantID uuid.UUID, query string, topK int) (string, error) {
	return s.ragEngine.Query(ctx, grantID, query, topK)
}

// ListByCategory returns grants filtered by category.
func (s *GrantService) ListByCategory(ctx context.Context, category string, limit, offset int32) ([]*domain.Grant, error) {
	return s.grants.ListByCategory(ctx, category, limit, offset)
}

// ArchiveGrant marks a grant as archived.
func (s *GrantService) ArchiveGrant(ctx context.Context, id uuid.UUID) error {
	return s.grants.Archive(ctx, id)
}
