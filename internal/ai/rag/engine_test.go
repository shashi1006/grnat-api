package rag

import (
	"strings"
	"testing"
)

func TestChunkText_ShortContent(t *testing.T) {
	e := NewEngine(nil, nil)
	text := "This is a short document about grant eligibility requirements."
	chunks := e.chunkText(text)
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if chunks[0].Content == "" {
		t.Error("chunk content should not be empty")
	}
}

func TestChunkText_LongContent(t *testing.T) {
	e := NewEngine(nil, nil)
	word := "grant "
	text := strings.Repeat(word, 700) // ~700 words, exceeds defaultChunkSize
	chunks := e.chunkText(text)
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks for long text, got %d", len(chunks))
	}
}

func TestChunkText_SectionDetection(t *testing.T) {
	e := NewEngine(nil, nil)
	text := `SECTION A. ELIGIBILITY REQUIREMENTS
Organizations must be 501c3 nonprofits.

SECTION B. PROGRAM REQUIREMENTS
Programs must serve low-income communities.`
	chunks := e.chunkText(text)
	foundSection := false
	for _, c := range chunks {
		if c.Section != nil && strings.Contains(*c.Section, "SECTION") {
			foundSection = true
			break
		}
	}
	if !foundSection {
		t.Error("expected section headers to be detected")
	}
}

func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		words   int
		minToks int
	}{
		{100, 100},
		{512, 500},
	}
	for _, tc := range cases {
		text := strings.Repeat("word ", tc.words)
		toks := estimateTokens(text)
		if toks < tc.minToks {
			t.Errorf("expected >= %d tokens for %d words, got %d", tc.minToks, tc.words, toks)
		}
	}
}
