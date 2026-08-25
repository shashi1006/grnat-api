// This file seeds the Funding OS product catalog and the ACS preparedness
// grant database, both ported from the fundingos-acs-demo-public.html sales
// demo, into the real products/grants tables so the Funding OS wizard can
// serve real data and run real compatibility scoring instead of static HTML.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

//go:embed data/products.json
var productsJSON []byte

//go:embed data/grants.json
var grantsJSON []byte

type rawProductCatalog struct {
	Products []rawProduct `json:"products"`
}

type rawProduct struct {
	ID               string          `json:"id"`
	Category         string          `json:"category"`
	ProductName      string          `json:"productName"`
	Overview         string          `json:"overview"`
	SelectionType    string          `json:"selectionType"`
	FeaturedProduct  bool            `json:"featuredProduct"`
	BaseProgramValue *float64        `json:"baseProgramValue"`
	FundingAlignment []string        `json:"fundingAlignment"`
	Configurations   []rawConfig     `json:"configurations"`
	Raw              json.RawMessage `json:"-"`
}

type rawConfig struct {
	Price float64 `json:"price"`
}

type rawGrant struct {
	ID                   string   `json:"id"`
	Category             string   `json:"category"`
	ProgramName          string   `json:"programName"`
	Agency               string   `json:"agency"`
	AwardRange           string   `json:"awardRange"`
	EligibleOrgTypes     []string `json:"eligibleOrgTypes"`
	EligibleProducts     []string `json:"eligibleProducts"`
	URL                  string   `json:"url"`
	StrategyNotes        string   `json:"strategyNotes"`
	HasACSEndorsement    bool     `json:"hasAcsEndorsement"`
	HasNarrativeTemplate bool     `json:"hasNarrativeTemplate"`
	IsSupplemental       bool     `json:"isSupplemental"`
}

// orgTypeKeyMap maps the demo's short eligibleOrgTypes keys to the Funding OS
// wizard's canonical org type slugs (domain.OrgType* constants).
var orgTypeKeyMap = map[string]string{
	"school":     string(domain.OrgTypeK12Schools),
	"higher-ed":  string(domain.OrgTypeHigherEd),
	"govt":       string(domain.OrgTypeMunicipalityGov),
	"ems-health": string(domain.OrgTypeEMSHealthcare),
	"nonprofit":  string(domain.OrgTypeNonprofitCommunity),
	"religious":  string(domain.OrgTypeHousesOfWorship),
	"workplace":  string(domain.OrgTypeWorkplaceIndustrial),
	"parks-rec":  string(domain.OrgTypeParksRecreation),
}

// productKeyToPriority maps the demo's short eligibleProducts keys to the
// wizard's canonical preparedness priority keys (domain.Priority* constants).
var productKeyToPriority = map[string]string{
	"bleeding":     string(domain.PriorityBleedingControl),
	"aed":          string(domain.PriorityCardiacResponse),
	"choking":      string(domain.PriorityChokingResponse),
	"notification": string(domain.PriorityEmergencyComms),
	"camera":       string(domain.PrioritySecurityIntegration),
	"firstaid":     string(domain.PriorityEventPreparedness),
}

func seedFundingOS(ctx context.Context, productRepo repository.ProductRepo, grantRepo repository.GrantRepo) error {
	if err := seedProducts(ctx, productRepo); err != nil {
		return fmt.Errorf("seed products: %w", err)
	}
	if err := seedACSGrants(ctx, grantRepo); err != nil {
		return fmt.Errorf("seed acs grants: %w", err)
	}
	return nil
}

func seedProducts(ctx context.Context, productRepo repository.ProductRepo) error {
	var catalog rawProductCatalog
	if err := json.Unmarshal(productsJSON, &catalog); err != nil {
		return err
	}

	// Re-decode as raw per-product objects so we can store them verbatim.
	var generic struct {
		Products []json.RawMessage `json:"products"`
	}
	if err := json.Unmarshal(productsJSON, &generic); err != nil {
		return err
	}

	for i, p := range catalog.Products {
		existing, _ := productRepo.GetBySlug(ctx, p.ID)
		if existing != nil {
			log.Printf("skip product (exists): %s", p.ID)
			continue
		}

		priceCents := int64(0)
		if len(p.Configurations) > 0 {
			priceCents = int64(p.Configurations[0].Price * 100)
		} else if p.BaseProgramValue != nil {
			priceCents = int64(*p.BaseProgramValue * 100)
		}

		category := p.Category
		selectionType := p.SelectionType
		created, err := productRepo.Create(ctx, repository.CreateProductParams{
			Slug:             p.ID,
			Name:             p.ProductName,
			Category:         &category,
			ShortDesc:        strPtr(p.Overview),
			Description:      strPtr(p.Overview),
			SelectionType:    &selectionType,
			PriceCents:       priceCents,
			PriceType:        "one_time",
			Featured:         p.FeaturedProduct,
			FundingAlignment: p.FundingAlignment,
			Catalog:          []byte(generic.Products[i]),
			SortOrder:        int32(i),
		})
		if err != nil {
			log.Printf("error creating product %s: %v", p.ID, err)
			continue
		}
		log.Printf("created product: %s (%s)", created.Name, created.ID)
	}
	return nil
}

func seedACSGrants(ctx context.Context, grantRepo repository.GrantRepo) error {
	var grants []rawGrant
	if err := json.Unmarshal(grantsJSON, &grants); err != nil {
		return err
	}

	for _, g := range grants {
		existing, _ := grantRepo.GetBySlug(ctx, g.ID)
		if existing != nil {
			_, err := grantRepo.Update(ctx, repository.UpdateGrantParams{
				ID:       existing.ID,
				Metadata: map[string]interface{}{"award_range": g.AwardRange},
			})
			if err != nil {
				log.Printf("error updating grant metadata %s: %v", g.ID, err)
			} else {
				log.Printf("updated grant metadata: %s", g.ID)
			}
			continue
		}

		minAward, maxAward := parseAwardRange(g.AwardRange)
		tags := append([]string{}, g.EligibleProducts...)
		if g.HasACSEndorsement {
			tags = append(tags, "acs-endorsement")
		}
		if g.HasNarrativeTemplate {
			tags = append(tags, "template")
		}
		if g.IsSupplemental {
			tags = append(tags, "supplemental")
		}

		params := repository.CreateGrantParams{
			Slug:                g.ID,
			Title:               g.ProgramName,
			FunderName:          g.Agency,
			FunderType:          inferFunderType(g.Category),
			Agency:              strPtr(g.Agency),
			Description:         strPtr(g.StrategyNotes),
			Synopsis:            strPtr(g.StrategyNotes),
			Category:            strPtr(g.Category),
			FocusAreas:          mapKeys(g.EligibleProducts, productKeyToPriority),
			EligibleOrgTypes:    mapKeys(g.EligibleOrgTypes, orgTypeKeyMap),
			EligiblePopulations: []string{},
			EligibleStates:      []string{},
			Requires501c3:       false,
			RequiresAuditedFin:  false,
			RequiresMatch:       false,
			MinAwardAmount:      minAward,
			MaxAwardAmount:      maxAward,
			ApplicationURL:      strPtr(g.URL),
			Status:              domain.GrantStatusActive,
			DifficultyLevel:     domain.DifficultyMedium,
			CompetitionLevel:    domain.CompetitionMedium,
			Tags:                tags,
			Metadata:            map[string]interface{}{"award_range": g.AwardRange},
		}

		created, err := grantRepo.Create(ctx, params)
		if err != nil {
			log.Printf("error creating grant %s: %v", g.ID, err)
			continue
		}
		log.Printf("created grant: %s (%s)", created.Title, created.ID)
	}
	return nil
}

// mapKeys translates a list of short demo keys into canonical keys using m,
// deduplicating and dropping any keys without a mapping.
func mapKeys(keys []string, m map[string]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, k := range keys {
		v, ok := m[k]
		if !ok || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// inferFunderType makes a best-effort guess at the funder type from the
// demo's free-text category label (e.g. "Federal — Department of Justice").
func inferFunderType(category string) domain.FunderType {
	c := strings.ToLower(category)
	switch {
	case strings.Contains(c, "federal"):
		return domain.FunderFederal
	case strings.Contains(c, "state"):
		return domain.FunderState
	case strings.Contains(c, "corporate") || strings.Contains(c, "foundation"):
		return domain.FunderCorporate
	default:
		return domain.FunderPrivate
	}
}

var moneyRe = regexp.MustCompile(`\$([0-9,.]+)\s*([MmKk]?)`)

// parseAwardRange best-effort parses strings like "Up to $500K", "$40M pool",
// "Varies by state" into award cents. Returns (nil, nil) when unparseable.
func parseAwardRange(s string) (min, max *int64) {
	match := moneyRe.FindStringSubmatch(s)
	if match == nil {
		return nil, nil
	}
	numStr := strings.ReplaceAll(match[1], ",", "")
	amount, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, nil
	}
	switch strings.ToLower(match[2]) {
	case "m":
		amount *= 1_000_000
	case "k":
		amount *= 1_000
	}
	cents := int64(amount * 100)
	if strings.Contains(strings.ToLower(s), "up to") {
		return nil, &cents
	}
	return nil, &cents
}
