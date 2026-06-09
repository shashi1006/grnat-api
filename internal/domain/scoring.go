package domain

import (
	"time"

	"github.com/google/uuid"
)

// CompatibilityTier classifies score ranges.
type CompatibilityTier string

const (
	TierStrongMatch  CompatibilityTier = "strong_match"
	TierGoodMatch    CompatibilityTier = "good_match"
	TierPartialMatch CompatibilityTier = "partial_match"
	TierLowMatch     CompatibilityTier = "low_match"
	TierNoMatch      CompatibilityTier = "no_match"
	TierUnknown      CompatibilityTier = "unknown"
)

// ScoreTierFromScore converts a numeric score to a tier.
func ScoreTierFromScore(score float64) CompatibilityTier {
	switch {
	case score >= 80:
		return TierStrongMatch
	case score >= 65:
		return TierGoodMatch
	case score >= 45:
		return TierPartialMatch
	case score >= 25:
		return TierLowMatch
	default:
		return TierNoMatch
	}
}

// DimensionScore holds a score for a single scoring dimension.
type DimensionScore struct {
	Key         string  `json:"key"`
	Score       float64 `json:"score"`
	MaxScore    float64 `json:"max_score"`
	Weight      float64 `json:"weight"`
	Explanation string  `json:"explanation"`
}

// CompatibilityScore stores the computed match between an org and a grant.
type CompatibilityScore struct {
	ID                uuid.UUID         `json:"id"`
	OrgID             uuid.UUID         `json:"org_id"`
	GrantID           uuid.UUID         `json:"grant_id"`
	TotalScore        float64           `json:"total_score"`
	Tier              CompatibilityTier `json:"tier"`
	DimensionScores   []DimensionScore  `json:"dimension_scores"`
	Disqualified      bool              `json:"disqualified"`
	DisqualifyReasons []string          `json:"disqualify_reasons"`
	Strengths         []string          `json:"strengths"`
	Gaps              []string          `json:"gaps"`
	Recommendations   []string          `json:"recommendations"`
	SemanticScore     *float64          `json:"semantic_score,omitempty"`
	EngineVersion     string            `json:"engine_version"`
	ComputedAt        time.Time         `json:"computed_at"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// ScoringInput bundles the org and grant data needed by the scoring engine.
type ScoringInput struct {
	Org     Organization
	Profile OrganizationProfile
	Grant   Grant
}

// ScoringResult is the raw output from the scoring engine before persistence.
type ScoringResult struct {
	TotalScore        float64
	Tier              CompatibilityTier
	DimensionScores   []DimensionScore
	Disqualified      bool
	DisqualifyReasons []string
	Strengths         []string
	Gaps              []string
	Recommendations   []string
	SemanticScore     *float64
}
