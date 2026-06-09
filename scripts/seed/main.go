// Command seed populates the database with the initial ReadyGeneration grant catalog.
// Usage: go run ./scripts/seed/main.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/readygeneration/readygeneration-backend/internal/config"
	"github.com/readygeneration/readygeneration-backend/internal/db"
	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	pgxrepo "github.com/readygeneration/readygeneration-backend/internal/repository/pgx"
)

func main() {
	if err := run(); err != nil {
		log.Println("seed error:", err)
		os.Exit(1)
	}
	log.Println("seed complete")
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := db.NewPool(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	grantRepo := pgxrepo.NewGrantRepo(pool)
	ctx := context.Background()

	for _, g := range seedGrants {
		existing, _ := grantRepo.GetBySlug(ctx, g.Slug)
		if existing != nil {
			log.Printf("skip (exists): %s", g.Slug)
			continue
		}
		created, err := grantRepo.Create(ctx, g)
		if err != nil {
			log.Printf("error creating %s: %v", g.Slug, err)
			continue
		}
		log.Printf("created: %s (%s)", created.Title, created.ID)
	}
	return nil
}

var seedGrants = []repository.CreateGrantParams{
	{
		Slug:                "hrsa-rural-health",
		Title:               "HRSA Rural Health Outreach Grant",
		FunderName:          "Health Resources & Services Administration",
		FunderType:          domain.FunderFederal,
		Agency:              strPtr("HRSA"),
		Description:         strPtr("Supports rural health networks to improve access to healthcare services in rural communities through outreach programs."),
		Synopsis:            strPtr("Funds organizations that coordinate healthcare services for rural populations."),
		Category:            strPtr("Health"),
		FocusAreas:          []string{"rural health", "health access", "outreach"},
		EligibleOrgTypes:    []string{"nonprofit", "government", "tribal"},
		EligiblePopulations: []string{},
		EligibleStates:      []string{},
		Requires501c3:       false,
		RequiresAuditedFin:  true,
		RequiresMatch:       false,
		MinAwardAmount:      int64Ptr(25000000),
		MaxAwardAmount:      int64Ptr(75000000),
		ApplicationURL:      strPtr("https://grants.hrsa.gov"),
		Status:              domain.GrantStatusActive,
		DifficultyLevel:     domain.DifficultyHard,
		CompetitionLevel:    domain.CompetitionHigh,
		Tags:                []string{"federal", "health", "rural"},
	},
	{
		Slug:                "cdbg-community-development",
		Title:               "Community Development Block Grant (CDBG)",
		FunderName:          "U.S. Department of Housing and Urban Development",
		FunderType:          domain.FunderFederal,
		Agency:              strPtr("HUD"),
		Description:         strPtr("Provides communities with resources to address a wide range of unique community development needs, primarily for low- and moderate-income persons."),
		Synopsis:            strPtr("Flexible grants to local governments for community development activities benefiting low/moderate income residents."),
		Category:            strPtr("Community Development"),
		FocusAreas:          []string{"community development", "housing", "economic development", "low income"},
		EligibleOrgTypes:    []string{"government", "nonprofit"},
		EligiblePopulations: []string{},
		EligibleStates:      []string{},
		Requires501c3:       false,
		RequiresAuditedFin:  false,
		RequiresMatch:       false,
		MinAwardAmount:      int64Ptr(10000000),
		MaxAwardAmount:      int64Ptr(500000000),
		ApplicationURL:      strPtr("https://www.hud.gov/program_offices/comm_planning/cdbg"),
		Status:              domain.GrantStatusActive,
		DifficultyLevel:     domain.DifficultyMedium,
		CompetitionLevel:    domain.CompetitionMedium,
		Tags:                []string{"federal", "community", "housing", "hud"},
	},
	{
		Slug:                "21st-cclc-afterschool",
		Title:               "21st Century Community Learning Centers",
		FunderName:          "U.S. Department of Education",
		FunderType:          domain.FunderFederal,
		Agency:              strPtr("ED"),
		Description:         strPtr("Supports creation of community learning centers that provide academic enrichment opportunities for children during non-school hours."),
		Synopsis:            strPtr("After-school and summer learning programs for students in high-poverty, low-performing schools."),
		Category:            strPtr("Education"),
		FocusAreas:          []string{"education", "afterschool", "youth development", "academic enrichment"},
		EligibleOrgTypes:    []string{"nonprofit", "school", "government"},
		EligiblePopulations: []string{},
		EligibleStates:      []string{},
		Requires501c3:       true,
		RequiresAuditedFin:  false,
		RequiresMatch:       false,
		MinAwardAmount:      int64Ptr(5000000),
		MaxAwardAmount:      int64Ptr(30000000),
		ApplicationURL:      strPtr("https://oese.ed.gov/offices/office-of-formula-grants/school-support-and-accountability/21st-century-community-learning-centers/"),
		Status:              domain.GrantStatusActive,
		DifficultyLevel:     domain.DifficultyMedium,
		CompetitionLevel:    domain.CompetitionHigh,
		Tags:                []string{"federal", "education", "youth", "afterschool"},
	},
	{
		Slug:                "ssvf-homeless-veterans",
		Title:               "Supportive Services for Veteran Families (SSVF)",
		FunderName:          "U.S. Department of Veterans Affairs",
		FunderType:          domain.FunderFederal,
		Agency:              strPtr("VA"),
		Description:         strPtr("Provides supportive services to very low-income veteran families in or transitioning to permanent housing."),
		Synopsis:            strPtr("Rapid rehousing and homelessness prevention services for veteran families."),
		Category:            strPtr("Veterans Services"),
		FocusAreas:          []string{"veterans", "homelessness", "housing", "supportive services"},
		EligibleOrgTypes:    []string{"nonprofit"},
		EligiblePopulations: []string{},
		EligibleStates:      []string{},
		Requires501c3:       true,
		RequiresAuditedFin:  true,
		RequiresMatch:       false,
		MinAwardAmount:      int64Ptr(10000000),
		MaxAwardAmount:      int64Ptr(200000000),
		ApplicationURL:      strPtr("https://www.va.gov/homeless/ssvf/"),
		Status:              domain.GrantStatusActive,
		DifficultyLevel:     domain.DifficultyHard,
		CompetitionLevel:    domain.CompetitionHigh,
		Tags:                []string{"federal", "veterans", "housing", "homelessness"},
	},
	{
		Slug:                "snap-ed-nutrition",
		Title:               "SNAP-Ed (Nutrition Education and Obesity Prevention)",
		FunderName:          "USDA Food and Nutrition Service",
		FunderType:          domain.FunderFederal,
		Agency:              strPtr("USDA FNS"),
		Description:         strPtr("Supports nutrition education and obesity prevention activities for SNAP participants and low-income individuals."),
		Synopsis:            strPtr("Community-based nutrition education interventions for low-income households."),
		Category:            strPtr("Food & Nutrition"),
		FocusAreas:          []string{"nutrition", "food access", "health", "obesity prevention", "low income"},
		EligibleOrgTypes:    []string{"nonprofit", "government", "university"},
		EligiblePopulations: []string{},
		EligibleStates:      []string{},
		Requires501c3:       false,
		RequiresAuditedFin:  false,
		RequiresMatch:       false,
		MinAwardAmount:      int64Ptr(5000000),
		MaxAwardAmount:      int64Ptr(50000000),
		ApplicationURL:      strPtr("https://www.fns.usda.gov/snap/snap-ed"),
		Status:              domain.GrantStatusActive,
		DifficultyLevel:     domain.DifficultyMedium,
		CompetitionLevel:    domain.CompetitionMedium,
		Tags:                []string{"federal", "nutrition", "usda", "food"},
	},
}

func strPtr(s string) *string { return &s }
func int64Ptr(n int64) *int64 { return &n }
