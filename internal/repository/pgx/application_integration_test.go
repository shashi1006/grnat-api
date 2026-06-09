package pgxrepo_test

import (
	"context"
	"testing"

	"github.com/readygeneration/readygeneration-backend/internal/domain"
	"github.com/readygeneration/readygeneration-backend/internal/repository"
	pgxrepo "github.com/readygeneration/readygeneration-backend/internal/repository/pgx"
	"github.com/readygeneration/readygeneration-backend/internal/testutil"
)

func TestApplicationRepo_CreateAndGet(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	orgRepo := pgxrepo.NewOrganizationRepo(pool)
	grantRepo := pgxrepo.NewGrantRepo(pool)
	appRepo := pgxrepo.NewApplicationRepo(pool)
	ctx := context.Background()

	org, err := orgRepo.Create(ctx, repository.CreateOrgParams{
		Name:    "App Test Org",
		Slug:    "app-test-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	grant, err := grantRepo.Create(ctx, repository.CreateGrantParams{
		Slug:             "app-test-grant",
		Title:            "App Test Grant",
		FunderName:       "Test Agency",
		FunderType:       domain.FunderFederal,
		Status:           domain.GrantStatusActive,
		DifficultyLevel:  domain.DifficultyMedium,
		CompetitionLevel: domain.CompetitionMedium,
	})
	testutil.Must(t, err, "create grant")

	app, err := appRepo.Create(ctx, repository.CreateApplicationParams{
		OrgID:    org.ID,
		GrantID:  grant.ID,
		Status:   domain.AppStatusProspect,
		Stage:    domain.StagePreApplication,
		Priority: domain.PriorityMedium,
	})
	testutil.Must(t, err, "create application")

	if app.OrgID != org.ID {
		t.Error("org_id mismatch")
	}
	if app.GrantID != grant.ID {
		t.Error("grant_id mismatch")
	}
	if app.Status != domain.AppStatusProspect {
		t.Errorf("status mismatch: got %s", app.Status)
	}

	got, err := appRepo.GetByID(ctx, app.ID)
	testutil.Must(t, err, "get by id")
	if got.GrantTitle == nil {
		t.Error("expected grant_title to be joined")
	}
}

func TestApplicationRepo_UpdateStatus(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	orgRepo := pgxrepo.NewOrganizationRepo(pool)
	grantRepo := pgxrepo.NewGrantRepo(pool)
	appRepo := pgxrepo.NewApplicationRepo(pool)
	ctx := context.Background()

	org, err := orgRepo.Create(ctx, repository.CreateOrgParams{
		Name:    "Status Test Org",
		Slug:    "status-test-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	grant, err := grantRepo.Create(ctx, repository.CreateGrantParams{
		Slug:             "status-test-grant",
		Title:            "Status Test Grant",
		FunderName:       "Agency",
		FunderType:       domain.FunderFederal,
		Status:           domain.GrantStatusActive,
		DifficultyLevel:  domain.DifficultyMedium,
		CompetitionLevel: domain.CompetitionMedium,
	})
	testutil.Must(t, err, "create grant")

	app, err := appRepo.Create(ctx, repository.CreateApplicationParams{
		OrgID:    org.ID,
		GrantID:  grant.ID,
		Status:   domain.AppStatusProspect,
		Stage:    domain.StagePreApplication,
		Priority: domain.PriorityMedium,
	})
	testutil.Must(t, err, "create application")

	stage := domain.StageApplication
	updated, err := appRepo.UpdateStatus(ctx, app.ID, domain.AppStatusDrafting, &stage)
	testutil.Must(t, err, "update status")

	if updated.Status != domain.AppStatusDrafting {
		t.Errorf("expected status=drafting, got %s", updated.Status)
	}
	if updated.Stage != domain.StageApplication {
		t.Errorf("expected stage=application, got %s", updated.Stage)
	}
}

func TestApplicationRepo_LogAndListActivities(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	orgRepo := pgxrepo.NewOrganizationRepo(pool)
	grantRepo := pgxrepo.NewGrantRepo(pool)
	appRepo := pgxrepo.NewApplicationRepo(pool)
	ctx := context.Background()

	org, err := orgRepo.Create(ctx, repository.CreateOrgParams{
		Name:    "Activity Org",
		Slug:    "activity-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	grant, err := grantRepo.Create(ctx, repository.CreateGrantParams{
		Slug:             "activity-grant",
		Title:            "Activity Grant",
		FunderName:       "Agency",
		FunderType:       domain.FunderFederal,
		Status:           domain.GrantStatusActive,
		DifficultyLevel:  domain.DifficultyMedium,
		CompetitionLevel: domain.CompetitionMedium,
	})
	testutil.Must(t, err, "create grant")

	app, err := appRepo.Create(ctx, repository.CreateApplicationParams{
		OrgID:    org.ID,
		GrantID:  grant.ID,
		Status:   domain.AppStatusProspect,
		Stage:    domain.StagePreApplication,
		Priority: domain.PriorityLow,
	})
	testutil.Must(t, err, "create application")

	_, err = appRepo.LogActivity(ctx, repository.LogActivityParams{
		ApplicationID: app.ID,
		ActivityType:  "note",
		NewValue:      testutil.Ptr("added a note"),
	})
	testutil.Must(t, err, "log activity")

	activities, err := appRepo.ListActivities(ctx, app.ID)
	testutil.Must(t, err, "list activities")
	if len(activities) == 0 {
		t.Error("expected at least one activity")
	}
	if activities[0].ActivityType != "note" {
		t.Errorf("activity type mismatch: got %s", activities[0].ActivityType)
	}
}

func TestApplicationRepo_Count(t *testing.T) {
	testutil.IntegrationSkip(t)
	pool := testutil.NewTestDB(t)
	orgRepo := pgxrepo.NewOrganizationRepo(pool)
	grantRepo := pgxrepo.NewGrantRepo(pool)
	appRepo := pgxrepo.NewApplicationRepo(pool)
	ctx := context.Background()

	org, err := orgRepo.Create(ctx, repository.CreateOrgParams{
		Name:    "Count App Org",
		Slug:    "count-app-org",
		OrgType: domain.OrgTypeNonprofit,
		Plan:    domain.PlanFree,
	})
	testutil.Must(t, err, "create org")

	grant, err := grantRepo.Create(ctx, repository.CreateGrantParams{
		Slug:             "count-app-grant",
		Title:            "Count App Grant",
		FunderName:       "Agency",
		FunderType:       domain.FunderFederal,
		Status:           domain.GrantStatusActive,
		DifficultyLevel:  domain.DifficultyMedium,
		CompetitionLevel: domain.CompetitionMedium,
	})
	testutil.Must(t, err, "create grant")

	before, err := appRepo.Count(ctx, org.ID)
	testutil.Must(t, err, "count before")

	_, err = appRepo.Create(ctx, repository.CreateApplicationParams{
		OrgID:    org.ID,
		GrantID:  grant.ID,
		Status:   domain.AppStatusProspect,
		Stage:    domain.StagePreApplication,
		Priority: domain.PriorityLow,
	})
	testutil.Must(t, err, "create application")

	after, err := appRepo.Count(ctx, org.ID)
	testutil.Must(t, err, "count after")
	if after != before+1 {
		t.Errorf("expected count +1: before=%d after=%d", before, after)
	}
}
