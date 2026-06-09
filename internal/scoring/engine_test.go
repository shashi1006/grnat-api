package scoring_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/scoring"
)

func TestEngine_Compute_Disqualified_No501c3(t *testing.T) {
	e := scoring.NewEngine()
	result := e.Compute(domain.ScoringInput{
		Org:     baseOrg(),
		Profile: domain.OrganizationProfile{Has501c3: false},
		Grant:   domain.Grant{ID: uuid.New(), Requires501c3: true, Title: "Test Grant"},
	})
	if !result.Disqualified {
		t.Fatal("expected disqualified=true when grant requires 501c3 and org lacks it")
	}
	if result.Tier != domain.TierNoMatch {
		t.Errorf("expected tier NoMatch, got %s", result.Tier)
	}
	if len(result.DisqualifyReasons) == 0 {
		t.Error("expected at least one disqualify reason")
	}
}

func TestEngine_Compute_Disqualified_WrongOrgType(t *testing.T) {
	e := scoring.NewEngine()
	org := baseOrg()
	org.OrgType = domain.OrgTypeNonprofit
	result := e.Compute(domain.ScoringInput{
		Org:     org,
		Profile: domain.OrganizationProfile{},
		Grant: domain.Grant{
			ID:               uuid.New(),
			EligibleOrgTypes: []string{"government"},
		},
	})
	if !result.Disqualified {
		t.Fatal("expected disqualified for ineligible org type")
	}
}

func TestEngine_Compute_PerfectMatch(t *testing.T) {
	e := scoring.NewEngine()
	state := "CA"
	budget := int64(500_000_00) // $500k in cents
	employees := int32(25)
	years := int32(5)
	result := e.Compute(domain.ScoringInput{
		Org: domain.Organization{
			ID:      uuid.New(),
			OrgType: domain.OrgTypeNonprofit,
			State:   &state,
		},
		Profile: domain.OrganizationProfile{
			Has501c3:             true,
			HasAuditedFinancials: true,
			HasIndirectCostRate:  true,
			PriorFederalGrants:   true,
			PopulationsServed:    []string{"youth", "low-income"},
			ProgramAreas:         []string{"education", "workforce"},
			FocusIssues:          []string{"education", "workforce"},
			AnnualBudget:         &budget,
			NumEmployees:         &employees,
			YearsOperating:       &years,
		},
		Grant: domain.Grant{
			ID:                  uuid.New(),
			Title:               "Perfect Match Grant",
			EligibleOrgTypes:    []string{"nonprofit"},
			EligibleStates:      []string{"CA"},
			EligiblePopulations: []string{"youth", "low-income"},
			FocusAreas:          []string{"education", "workforce"},
			Requires501c3:       true,
			RequiresAuditedFin:  true,
			MaxAwardAmount:      int64Ptr(100_000_00),
		},
	})
	if result.Disqualified {
		t.Fatalf("expected not disqualified, reasons: %v", result.DisqualifyReasons)
	}
	if result.TotalScore < 80 {
		t.Errorf("expected high score for perfect match, got %.1f", result.TotalScore)
	}
	if result.Tier != domain.TierStrongMatch && result.Tier != domain.TierGoodMatch {
		t.Errorf("expected StrongMatch or GoodMatch tier, got %s", result.Tier)
	}
}

func TestEngine_Compute_LowCapacityOrg(t *testing.T) {
	e := scoring.NewEngine()
	state := "TX"
	employees := int32(2)
	result := e.Compute(domain.ScoringInput{
		Org: domain.Organization{
			ID:      uuid.New(),
			OrgType: domain.OrgTypeNonprofit,
			State:   &state,
		},
		Profile: domain.OrganizationProfile{
			Has501c3:     true,
			NumEmployees: &employees,
		},
		Grant: domain.Grant{
			ID:               uuid.New(),
			EligibleOrgTypes: []string{"nonprofit"},
			EligibleStates:   []string{},
		},
	})
	if result.Disqualified {
		t.Fatal("should not be disqualified")
	}
	// Capacity score should be low
	var capScore float64 = -1
	for _, d := range result.DimensionScores {
		if d.Key == "org_capacity" {
			capScore = d.Score
		}
	}
	if capScore < 0 {
		t.Fatal("org_capacity dimension not found")
	}
	if capScore > 50 {
		t.Errorf("expected low capacity score for 2-person org, got %.1f", capScore)
	}
}

func TestEngine_Compute_GeographicMismatch(t *testing.T) {
	e := scoring.NewEngine()
	state := "FL"
	result := e.Compute(domain.ScoringInput{
		Org: domain.Organization{
			ID:      uuid.New(),
			OrgType: domain.OrgTypeNonprofit,
			State:   &state,
		},
		Profile: domain.OrganizationProfile{Has501c3: true},
		Grant: domain.Grant{
			ID:               uuid.New(),
			EligibleOrgTypes: []string{"nonprofit"},
			EligibleStates:   []string{"CA", "NY"},
		},
	})
	// Geographic mismatch is a disqualifier in checkDisqualifiers
	if !result.Disqualified {
		t.Fatal("expected disqualified for out-of-state org")
	}
}

func TestScoreTierFromScore(t *testing.T) {
	cases := []struct {
		score float64
		tier  domain.CompatibilityTier
	}{
		{95, domain.TierStrongMatch},
		{80, domain.TierStrongMatch},
		{65, domain.TierGoodMatch},
		{50, domain.TierPartialMatch},
		{30, domain.TierLowMatch},
		{0, domain.TierNoMatch},
	}
	for _, tc := range cases {
		got := domain.ScoreTierFromScore(tc.score)
		if got != tc.tier {
			t.Errorf("score %.0f: expected tier %s, got %s", tc.score, tc.tier, got)
		}
	}
}

func baseOrg() domain.Organization {
	state := "CA"
	return domain.Organization{
		ID:      uuid.New(),
		OrgType: domain.OrgTypeNonprofit,
		State:   &state,
	}
}

func int64Ptr(n int64) *int64 { return &n }
