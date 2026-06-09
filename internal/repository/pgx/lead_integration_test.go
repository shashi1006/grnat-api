package pgxrepo_test

import (
	"context"
	"testing"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	pgxrepo "github.com/readygeneration/readygeneration-backend/internal/repository/pgx"
	"github.com/readygeneration/readygeneration-backend/internal/testutil"
)

func TestLeadRepo_CreateAndGet(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewLeadRepo(pool)
	ctx := context.Background()

	lead, err := repo.Create(ctx, repository.CreateLeadParams{
		Email:  "jane@example.com",
		Source: domain.LeadSourceOrganic,
		QuizResponses: map[string]interface{}{
			"org_type": "nonprofit",
		},
	})
	testutil.Must(t, err, "create lead")

	if lead.Email != "jane@example.com" {
		t.Errorf("email mismatch: got %s", lead.Email)
	}
	if lead.Status != domain.LeadStatusNew {
		t.Errorf("expected status=new, got %s", lead.Status)
	}
	if lead.QuizResponses["org_type"] != "nonprofit" {
		t.Errorf("quiz_responses not persisted correctly")
	}

	got, err := repo.GetByID(ctx, lead.ID)
	testutil.Must(t, err, "get by id")
	if got.ID != lead.ID {
		t.Error("ID mismatch")
	}
}

func TestLeadRepo_GetByEmail(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewLeadRepo(pool)
	ctx := context.Background()

	_, err := repo.Create(ctx, repository.CreateLeadParams{
		Email:  "lookup@example.com",
		Source: domain.LeadSourceReferral,
	})
	testutil.Must(t, err, "create lead")

	got, err := repo.GetByEmail(ctx, "lookup@example.com")
	testutil.Must(t, err, "get by email")
	if got.Email != "lookup@example.com" {
		t.Errorf("email mismatch: got %s", got.Email)
	}
}

func TestLeadRepo_Update(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewLeadRepo(pool)
	ctx := context.Background()

	lead, err := repo.Create(ctx, repository.CreateLeadParams{
		Email:  "update@example.com",
		Source: domain.LeadSourceOrganic,
	})
	testutil.Must(t, err, "create lead")

	status := domain.LeadStatusContacted
	updated, err := repo.Update(ctx, repository.UpdateLeadParams{
		ID:        lead.ID,
		FirstName: testutil.Ptr("Jane"),
		Status:    &status,
	})
	testutil.Must(t, err, "update lead")

	if updated.FirstName == nil || *updated.FirstName != "Jane" {
		t.Error("first_name not updated")
	}
	if updated.Status != domain.LeadStatusContacted {
		t.Errorf("status not updated: got %s", updated.Status)
	}
}

func TestLeadRepo_Search(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewLeadRepo(pool)
	ctx := context.Background()

	_, err := repo.Create(ctx, repository.CreateLeadParams{
		Email:     "search-me@example.com",
		FirstName: testutil.Ptr("SearchFirst"),
		Source:    domain.LeadSourceOrganic,
	})
	testutil.Must(t, err, "create lead")

	results, err := repo.Search(ctx, "SearchFirst", 10, 0)
	testutil.Must(t, err, "search")
	if len(results) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestLeadRepo_Convert(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	leadRepo := pgxrepo.NewLeadRepo(pool)
	orgRepo := pgxrepo.NewOrganizationRepo(pool)
	ctx := context.Background()

	lead, err := leadRepo.Create(ctx, repository.CreateLeadParams{
		Email:  "convert@example.com",
		Source: domain.LeadSourceDemo,
	})
	testutil.Must(t, err, "create lead")

	org, err := orgRepo.Create(ctx, repository.CreateOrgParams{
		Name:    "Converted Org",
		Slug:    "converted-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	testutil.Must(t, leadRepo.Convert(ctx, lead.ID, org.ID), "convert")

	got, err := leadRepo.GetByID(ctx, lead.ID)
	testutil.Must(t, err, "get converted lead")
	if got.Status != domain.LeadStatusConverted {
		t.Errorf("expected status=converted, got %s", got.Status)
	}
	if got.ConvertedOrgID == nil || *got.ConvertedOrgID != org.ID {
		t.Error("converted_org_id not set")
	}
}
