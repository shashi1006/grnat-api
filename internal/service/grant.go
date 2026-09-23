package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/ai/claude"
	"github.com/readygeneration/readygeneration-backend/internal/ai/embedding"
	"github.com/readygeneration/readygeneration-backend/internal/ai/openai"
	"github.com/readygeneration/readygeneration-backend/internal/ai/rag"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// GrantService handles grant catalog management, NOFO ingestion, and semantic search.
type GrantService struct {
	grants       repository.GrantRepo
	embedSvc     *embedding.Service
	ragEngine    *rag.Engine
	claudeClient *claude.Client
	openAIClient *openai.Client
}

// NewGrantService creates a GrantService.
func NewGrantService(grants repository.GrantRepo, embedSvc *embedding.Service, ragEngine *rag.Engine, claudeClient *claude.Client, openAIClient *openai.Client) *GrantService {
	return &GrantService{grants: grants, embedSvc: embedSvc, ragEngine: ragEngine, claudeClient: claudeClient, openAIClient: openAIClient}
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

// ExtractRequirements runs the NOFO text through Claude to pull structured
// submission requirements — who may apply, the submission pathway, required
// forms, narrative sections, set-asides, and certifications — and persists
// them on the grant.
func (s *GrantService) ExtractRequirements(ctx context.Context, grantID uuid.UUID) (*domain.Grant, error) {
	if s.claudeClient == nil && s.openAIClient == nil {
		return nil, fmt.Errorf("no LLM client configured for requirement extraction")
	}

	grant, err := s.grants.GetByID(ctx, grantID)
	if err != nil {
		return nil, fmt.Errorf("grant not found: %w", err)
	}

	text := ""
	if grant.FullNOFOText != nil {
		text = *grant.FullNOFOText
	}
	if text == "" {
		chunks, err := s.grants.ListNOFOChunks(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("load NOFO chunks: %w", err)
		}
		for _, c := range chunks {
			text += c.Content + "\n\n"
		}
	}
	if text == "" {
		return nil, fmt.Errorf("no NOFO text stored for this grant — ingest the NOFO first")
	}

	var extracted *domain.ExtractedRequirements
	if s.claudeClient != nil {
		extracted, err = s.claudeClient.ExtractRequirements(ctx, grant.Title, text)
	}
	if extracted == nil && s.openAIClient != nil {
		extracted, err = s.openAIClient.ExtractRequirements(ctx, grant.Title, text)
	}
	if err != nil && extracted == nil {
		return nil, err
	}

	now := time.Now()
	var note *string
	if extracted.PassThroughNote != "" {
		note = &extracted.PassThroughNote
	}
	return s.grants.UpdateRequirements(ctx, grantID, repository.UpdateRequirementsParams{
		EligibleApplicants:      extracted.EligibleApplicants,
		SubmissionPathway:       domain.SubmissionPathway(extracted.SubmissionPathway),
		PassThroughNote:         note,
		SubmissionRequirements:  extracted.SubmissionRequirementsMap(),
		RequirementsExtractedAt: &now,
	})
}
