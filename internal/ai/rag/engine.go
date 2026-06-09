// Package rag implements Retrieval-Augmented Generation for NOFO documents.
// It chunks NOFO text, generates embeddings, stores them via pgvector,
// and retrieves the most relevant passages for a given query.
package rag

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

const (
	defaultChunkSize    = 512  // tokens (approx words × 1.3)
	defaultChunkOverlap = 64
	defaultTopK         = 5
)

// EmbedFunc is a function that converts text to a vector embedding.
type EmbedFunc func(ctx context.Context, text string) ([]float32, error)

// Engine handles NOFO ingestion and semantic retrieval.
type Engine struct {
	grantRepo repository.GrantRepo
	embed     EmbedFunc
	chunkSize int
	overlap   int
}

// NewEngine creates a RAG engine.
func NewEngine(grantRepo repository.GrantRepo, embedFn EmbedFunc) *Engine {
	return &Engine{
		grantRepo: grantRepo,
		embed:     embedFn,
		chunkSize: defaultChunkSize,
		overlap:   defaultChunkOverlap,
	}
}

// IngestNOFO chunks the NOFO text for a grant, generates embeddings, and stores them.
func (e *Engine) IngestNOFO(ctx context.Context, grantID uuid.UUID, nofoText string) error {
	if err := e.grantRepo.UpdateNOFO(ctx, grantID, nofoText); err != nil {
		return fmt.Errorf("store nofo text: %w", err)
	}

	chunks := e.chunkText(nofoText)

	for i, chunk := range chunks {
		emb, err := e.embed(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("embed chunk %d: %w", i, err)
		}

		tokenCount := int32(estimateTokens(chunk.Content))
		idx := int32(i)
		if _, err := e.grantRepo.UpsertNOFOChunk(ctx, repository.UpsertChunkParams{
			GrantID:    grantID,
			ChunkIndex: idx,
			Section:    chunk.Section,
			Content:    chunk.Content,
			TokenCount: &tokenCount,
			Embedding:  emb,
		}); err != nil {
			return fmt.Errorf("upsert chunk %d: %w", i, err)
		}
	}

	return nil
}

// Query retrieves the top-K most relevant NOFO chunks for a given query.
func (e *Engine) Query(ctx context.Context, grantID uuid.UUID, query string, topK int) (string, error) {
	if topK <= 0 {
		topK = defaultTopK
	}

	emb, err := e.embed(ctx, query)
	if err != nil {
		return "", fmt.Errorf("embed query: %w", err)
	}

	chunks, err := e.grantRepo.SearchChunksBySimilarity(ctx, grantID, emb, int32(topK))
	if err != nil {
		return "", fmt.Errorf("search chunks: %w", err)
	}

	if len(chunks) == 0 {
		return "", nil
	}

	var sb strings.Builder
	for i, c := range chunks {
		if i > 0 {
			sb.WriteString("\n\n---\n\n")
		}
		if c.Section != nil {
			sb.WriteString(fmt.Sprintf("[%s]\n", *c.Section))
		}
		sb.WriteString(c.Content)
	}

	return sb.String(), nil
}

// QueryGrants finds the most semantically similar grants to a free-text description.
func (e *Engine) QueryGrants(ctx context.Context, query string, topK int) ([]*repository.GrantWithDistance, error) {
	if topK <= 0 {
		topK = defaultTopK
	}

	emb, err := e.embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed grant query: %w", err)
	}

	return e.grantRepo.SearchBySimilarity(ctx, emb, int32(topK))
}

// textChunk holds a piece of text and its detected section label.
type textChunk struct {
	Section *string
	Content string
}

// chunkText splits long NOFO text into overlapping chunks by section boundaries.
func (e *Engine) chunkText(text string) []textChunk {
	var chunks []textChunk
	sections := splitIntoSections(text)

	for _, sec := range sections {
		words := strings.Fields(sec.Content)
		if len(words) == 0 {
			continue
		}

		// If section fits in one chunk, store it whole
		if len(words) <= e.chunkSize {
			chunks = append(chunks, textChunk{Section: sec.Section, Content: strings.Join(words, " ")})
			continue
		}

		// Sliding window chunking
		for start := 0; start < len(words); start += e.chunkSize - e.overlap {
			end := start + e.chunkSize
			if end > len(words) {
				end = len(words)
			}
			content := strings.Join(words[start:end], " ")
			chunks = append(chunks, textChunk{Section: sec.Section, Content: content})
			if end == len(words) {
				break
			}
		}
	}

	return chunks
}

type sectionText struct {
	Section *string
	Content string
}

// splitIntoSections detects section headers and groups text under them.
func splitIntoSections(text string) []sectionText {
	lines := strings.Split(text, "\n")
	var sections []sectionText
	var currentSection *string
	var currentLines []string

	flush := func() {
		content := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if content != "" {
			sections = append(sections, sectionText{Section: currentSection, Content: content})
		}
		currentLines = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isSectionHeader(trimmed) {
			flush()
			s := trimmed
			currentSection = &s
		} else {
			currentLines = append(currentLines, line)
		}
	}
	flush()

	if len(sections) == 0 {
		sections = []sectionText{{Section: nil, Content: text}}
	}
	return sections
}

// isSectionHeader detects all-caps lines or lines that look like NOFO headers.
func isSectionHeader(line string) bool {
	if len(line) < 4 || len(line) > 120 {
		return false
	}
	upperCount := 0
	for _, r := range line {
		if unicode.IsUpper(r) {
			upperCount++
		}
	}
	// Roman numeral or numbered section patterns
	commonPrefixes := []string{"SECTION ", "PART ", "A. ", "B. ", "C. ", "D. ", "I. ", "II. ", "III. ", "IV. ", "V. "}
	for _, p := range commonPrefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return float64(upperCount)/float64(len(line)) > 0.6
}

// estimateTokens provides a rough token count (~1.3 tokens per word).
func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	return int(float64(words) * 1.3)
}
