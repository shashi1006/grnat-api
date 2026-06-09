package pgxrepo_test

import (
	"context"
	"testing"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	pgxrepo "github.com/readygeneration/readygeneration-backend/internal/repository/pgx"
	"github.com/readygeneration/readygeneration-backend/internal/testutil"
)

func TestOrgRepo_CreateAndGet(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewOrganizationRepo(pool)
	ctx := context.Background()

	org, err := repo.Create(ctx, repository.CreateOrgParams{
		Name:    "Test Nonprofit",
		Slug:    "test-nonprofit",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	if org.ID.String() == "" {
		t.Fatal("expected non-empty ID")
	}

	got, err := repo.GetByID(ctx, org.ID)
	testutil.Must(t, err, "get by id")
	if got.Name != "Test Nonprofit" {
		t.Errorf("name mismatch: got %s", got.Name)
	}
	if got.OrgType != domain.OrgTypeNonprofit {
		t.Errorf("org_type mismatch: got %s", got.OrgType)
	}
}

func TestOrgRepo_GetBySlug(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewOrganizationRepo(pool)
	ctx := context.Background()

	_, err := repo.Create(ctx, repository.CreateOrgParams{
		Name:    "Slug Org",
		Slug:    "slug-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	got, err := repo.GetBySlug(ctx, "slug-org")
	testutil.Must(t, err, "get by slug")
	if got.Slug != "slug-org" {
		t.Errorf("slug mismatch: got %s", got.Slug)
	}
}

func TestOrgRepo_Update(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewOrganizationRepo(pool)
	ctx := context.Background()

	org, err := repo.Create(ctx, repository.CreateOrgParams{
		Name:    "Before Update",
		Slug:    "before-update",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	updated, err := repo.Update(ctx, repository.UpdateOrgParams{
		ID:   org.ID,
		Name: testutil.Ptr("After Update"),
	})
	testutil.Must(t, err, "update org")
	if updated.Name != "After Update" {
		t.Errorf("expected 'After Update', got %s", updated.Name)
	}
}

func TestOrgRepo_UpsertProfile(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewOrganizationRepo(pool)
	ctx := context.Background()

	org, err := repo.Create(ctx, repository.CreateOrgParams{
		Name:    "Profile Org",
		Slug:    "profile-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	profile, err := repo.UpsertProfile(ctx, repository.UpsertProfileParams{
		OrgID:              org.ID,
		Has501c3:           true,
		PriorFederalGrants: true,
		NumEmployees:       testutil.Ptr(int32(10)),
		YearsOperating:     testutil.Ptr(int32(5)),
		ProgramAreas:       []string{"education", "workforce"},
	})
	testutil.Must(t, err, "upsert profile")
	if !profile.Has501c3 {
		t.Error("expected Has501c3=true")
	}
	if len(profile.ProgramAreas) != 2 {
		t.Errorf("expected 2 program areas, got %d", len(profile.ProgramAreas))
	}
}

func TestOrgRepo_Count(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	repo := pgxrepo.NewOrganizationRepo(pool)
	ctx := context.Background()

	before, err := repo.Count(ctx)
	testutil.Must(t, err, "count before")

	_, err = repo.Create(ctx, repository.CreateOrgParams{
		Name:    "Count Org",
		Slug:    "count-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	after, err := repo.Count(ctx)
	testutil.Must(t, err, "count after")
	if after != before+1 {
		t.Errorf("expected count to increase by 1: before=%d after=%d", before, after)
	}
}
