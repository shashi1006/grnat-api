// Package scoring implements the grant compatibility scoring engine.
// It produces a 0-100 score across weighted dimensions, detecting
// hard disqualifiers before computing soft dimension scores.
package scoring

import (
	"fmt"
	"math"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
)

const engineVersion = "v1"

// Engine computes compatibility between an org profile and a grant.
type Engine struct{}

// NewEngine creates a new scoring Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// dimensions and their default weights (sum to 100 when all present)
var defaultWeights = map[string]float64{
	"org_type_match":      20.0,
	"population_match":    18.0,
	"geographic_match":    15.0,
	"program_area_match":  15.0,
	"financial_readiness": 12.0,
	"org_capacity":        10.0,
	"compliance_ready":    10.0,
}

// Compute scores an org/grant pair and returns a ScoringResult.
func (e *Engine) Compute(input domain.ScoringInput) domain.ScoringResult {
	result := domain.ScoringResult{}

	// --- Hard disqualifier checks ---
	if reasons := e.checkDisqualifiers(input); len(reasons) > 0 {
		result.Disqualified = true
		result.DisqualifyReasons = reasons
		result.Tier = domain.TierNoMatch
		result.Strengths = []string{}
		result.Gaps = []string{}
		result.Recommendations = []string{}
		return result
	}

	// --- Dimension scoring ---
	dims := []domain.DimensionScore{
		e.scoreOrgType(input),
		e.scorePopulation(input),
		e.scoreGeographic(input),
		e.scoreProgramArea(input),
		e.scoreFinancialReadiness(input),
		e.scoreOrgCapacity(input),
		e.scoreComplianceReadiness(input),
	}

	// Weighted sum
	var weightedSum, totalWeight float64
	for _, d := range dims {
		weightedSum += d.Score * d.Weight
		totalWeight += d.Weight
	}
	var total float64
	if totalWeight > 0 {
		total = math.Round((weightedSum/totalWeight)*10) / 10
	}

	result.TotalScore = total
	result.Tier = domain.ScoreTierFromScore(total)
	result.DimensionScores = dims
	result.Strengths = e.deriveStrengths(dims)
	result.Gaps = e.deriveGaps(dims)
	result.Recommendations = e.deriveRecommendations(input, dims)

	return result
}

// --- Disqualifier checks ---

func (e *Engine) checkDisqualifiers(input domain.ScoringInput) []string {
	var reasons []string
	g := input.Grant
	p := input.Profile
	org := input.Org

	if g.Requires501c3 && !p.Has501c3 {
		reasons = append(reasons, "Grant requires 501(c)(3) status")
	}
	if g.RequiresAuditedFin && !p.HasAuditedFinancials {
		reasons = append(reasons, "Grant requires audited financial statements")
	}
	if g.RequiresIndirectRate && !p.HasIndirectCostRate {
		reasons = append(reasons, "Grant requires a negotiated indirect cost rate")
	}

	// Org type eligibility
	if len(g.EligibleOrgTypes) > 0 && !contains(g.EligibleOrgTypes, string(org.OrgType)) {
		reasons = append(reasons, "Organization type is not eligible for this grant")
	}

	// Geographic restriction
	if len(g.EligibleStates) > 0 && org.State != nil && !contains(g.EligibleStates, *org.State) {
		reasons = append(reasons, "Organization's state is not in the eligible geographic area")
	}

	return reasons
}

// --- Dimension scorers ---

func (e *Engine) scoreOrgType(input domain.ScoringInput) domain.DimensionScore {
	weight := defaultWeights["org_type_match"]
	g := input.Grant
	org := input.Org

	if len(g.EligibleOrgTypes) == 0 {
		return domain.DimensionScore{Key: "org_type_match", Score: 100, MaxScore: 100, Weight: weight, Explanation: "No org type restriction"}
	}
	if contains(g.EligibleOrgTypes, string(org.OrgType)) {
		return domain.DimensionScore{Key: "org_type_match", Score: 100, MaxScore: 100, Weight: weight, Explanation: "Organization type matches eligibility criteria"}
	}
	return domain.DimensionScore{Key: "org_type_match", Score: 0, MaxScore: 100, Weight: weight, Explanation: "Organization type does not match — may be disqualified"}
}

func (e *Engine) scorePopulation(input domain.ScoringInput) domain.DimensionScore {
	weight := defaultWeights["population_match"]
	g := input.Grant
	p := input.Profile

	if len(g.EligiblePopulations) == 0 {
		return domain.DimensionScore{Key: "population_match", Score: 100, MaxScore: 100, Weight: weight, Explanation: "No target population restriction"}
	}
	if len(p.PopulationsServed) == 0 {
		return domain.DimensionScore{Key: "population_match", Score: 30, MaxScore: 100, Weight: weight, Explanation: "Organization has not specified target populations"}
	}

	overlap := countOverlap(g.EligiblePopulations, p.PopulationsServed)
	score := math.Min(100, float64(overlap)/float64(len(g.EligiblePopulations))*100)
	explanation := fmt.Sprintf("%d of %d required populations served", overlap, len(g.EligiblePopulations))
	return domain.DimensionScore{Key: "population_match", Score: score, MaxScore: 100, Weight: weight, Explanation: explanation}
}

func (e *Engine) scoreGeographic(input domain.ScoringInput) domain.DimensionScore {
	weight := defaultWeights["geographic_match"]
	g := input.Grant
	org := input.Org

	if len(g.EligibleStates) == 0 {
		return domain.DimensionScore{Key: "geographic_match", Score: 100, MaxScore: 100, Weight: weight, Explanation: "Nationwide grant — no geographic restriction"}
	}
	if org.State != nil && contains(g.EligibleStates, *org.State) {
		return domain.DimensionScore{Key: "geographic_match", Score: 100, MaxScore: 100, Weight: weight, Explanation: "Organization is in an eligible state"}
	}
	return domain.DimensionScore{Key: "geographic_match", Score: 0, MaxScore: 100, Weight: weight, Explanation: "Organization state not in eligible states list"}
}

func (e *Engine) scoreProgramArea(input domain.ScoringInput) domain.DimensionScore {
	weight := defaultWeights["program_area_match"]
	g := input.Grant
	p := input.Profile

	if len(g.FocusAreas) == 0 {
		return domain.DimensionScore{Key: "program_area_match", Score: 80, MaxScore: 100, Weight: weight, Explanation: "Grant has broad focus areas"}
	}
	if len(p.ProgramAreas) == 0 {
		return domain.DimensionScore{Key: "program_area_match", Score: 20, MaxScore: 100, Weight: weight, Explanation: "Organization has not specified program areas"}
	}

	overlap := countOverlap(g.FocusAreas, p.ProgramAreas)
	score := math.Min(100, float64(overlap)/float64(len(g.FocusAreas))*100)
	if overlap == 0 {
		// Partial credit if org focus issues overlap
		issueOverlap := countOverlap(g.FocusAreas, p.FocusIssues)
		if issueOverlap > 0 {
			score = 30
		}
	}
	return domain.DimensionScore{Key: "program_area_match", Score: score, MaxScore: 100, Weight: weight, Explanation: fmt.Sprintf("%d program areas align", overlap)}
}

func (e *Engine) scoreFinancialReadiness(input domain.ScoringInput) domain.DimensionScore {
	weight := defaultWeights["financial_readiness"]
	g := input.Grant
	p := input.Profile

	score := 60.0 // base
	explanation := "Standard financial profile"

	if p.HasAuditedFinancials {
		score += 25
		explanation = "Has audited financials — strong financial readiness"
	}
	if p.HasIndirectCostRate {
		score += 15
	}

	// Budget alignment
	if g.MinAwardAmount != nil && p.AnnualBudget != nil {
		minAward := *g.MinAwardAmount
		budget := *p.AnnualBudget
		if budget < minAward/2 {
			score = math.Max(score-20, 10)
			explanation = "Annual budget may be too small relative to award amount"
		}
	}

	score = math.Min(score, 100)
	return domain.DimensionScore{Key: "financial_readiness", Score: score, MaxScore: 100, Weight: weight, Explanation: explanation}
}

func (e *Engine) scoreOrgCapacity(input domain.ScoringInput) domain.DimensionScore {
	weight := defaultWeights["org_capacity"]
	p := input.Profile

	score := 50.0
	explanation := "Unknown capacity"

	if p.NumEmployees != nil {
		n := int(*p.NumEmployees)
		switch {
		case n >= 50:
			score = 100
			explanation = "Large staff — strong capacity"
		case n >= 20:
			score = 85
			explanation = "Mid-size staff"
		case n >= 5:
			score = 70
			explanation = "Small staff"
		default:
			score = 40
			explanation = "Very small staff — may lack implementation capacity"
		}
	}

	if p.YearsOperating != nil && *p.YearsOperating >= 3 {
		score = math.Min(score+10, 100)
	}
	if p.PriorFederalGrants {
		score = math.Min(score+10, 100)
	}

	return domain.DimensionScore{Key: "org_capacity", Score: score, MaxScore: 100, Weight: weight, Explanation: explanation}
}

func (e *Engine) scoreComplianceReadiness(input domain.ScoringInput) domain.DimensionScore {
	weight := defaultWeights["compliance_ready"]
	p := input.Profile
	g := input.Grant

	score := 70.0
	explanation := "Basic compliance readiness assumed"

	if g.Requires501c3 && p.Has501c3 {
		score = math.Min(score+15, 100)
		explanation = "Has 501(c)(3) as required"
	}
	if g.RequiresMatch && p.AnnualBudget != nil && *p.AnnualBudget > 100000_00 { // > $100k
		score = math.Min(score+10, 100)
		explanation = "Budget capacity to meet match requirement"
	}

	return domain.DimensionScore{Key: "compliance_ready", Score: score, MaxScore: 100, Weight: weight, Explanation: explanation}
}

// --- Insight derivation ---

func (e *Engine) deriveStrengths(dims []domain.DimensionScore) []string {
	var s []string
	for _, d := range dims {
		if d.Score >= 80 {
			s = append(s, d.Explanation)
		}
	}
	return s
}

func (e *Engine) deriveGaps(dims []domain.DimensionScore) []string {
	var g []string
	for _, d := range dims {
		if d.Score < 50 {
			g = append(g, d.Explanation)
		}
	}
	return g
}

func (e *Engine) deriveRecommendations(input domain.ScoringInput, dims []domain.DimensionScore) []string {
	var recs []string
	p := input.Profile

	for _, d := range dims {
		if d.Score >= 50 {
			continue
		}
		switch d.Key {
		case "population_match":
			recs = append(recs, "Update your organization profile to list all target populations served")
		case "program_area_match":
			recs = append(recs, "Align your program areas with the grant's focus areas in your profile")
		case "financial_readiness":
			if !p.HasAuditedFinancials {
				recs = append(recs, "Obtain audited financial statements to strengthen future applications")
			}
		case "org_capacity":
			recs = append(recs, "Document volunteers and partnerships to demonstrate implementation capacity")
		case "compliance_ready":
			recs = append(recs, "Ensure all compliance documentation (501c3, registrations) is up to date")
		}
	}
	return recs
}

// --- Helpers ---

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func countOverlap(a, b []string) int {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	n := 0
	for _, v := range b {
		if _, ok := set[v]; ok {
			n++
		}
	}
	return n
}
