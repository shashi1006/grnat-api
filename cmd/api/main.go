// @title           ReadyGeneration API
// @version         1.0
// @description     Grant intelligence platform API
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/readygeneration/readygeneration-backend/docs" // Swagger generated docs

	"github.com/readygeneration/readygeneration-backend/internal/ai/claude"
	"github.com/readygeneration/readygeneration-backend/internal/ai/embedding"
	"github.com/readygeneration/readygeneration-backend/internal/ai/rag"
	"github.com/readygeneration/readygeneration-backend/internal/config"
	"github.com/readygeneration/readygeneration-backend/internal/db"
	"github.com/readygeneration/readygeneration-backend/internal/handler"
	"github.com/readygeneration/readygeneration-backend/internal/migrate"
	pgxrepo "github.com/readygeneration/readygeneration-backend/internal/repository/pgx"
	"github.com/readygeneration/readygeneration-backend/internal/router"
	"github.com/readygeneration/readygeneration-backend/internal/scoring"
	"github.com/readygeneration/readygeneration-backend/internal/service"
	"github.com/readygeneration/readygeneration-backend/pkg/jwt"
)

func main() {
	// --- Config ---
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// --- Database (auto-create if missing) ---
	if err := db.EnsureDB(context.Background(), cfg.DB.URL); err != nil {
		log.Fatalf("ensure db: %v", err)
	}

	pool, err := db.NewPool(context.Background(), cfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	// --- Migrations ---
	migrationsDir := cfg.DB.MigrationsDir
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}
	if err := migrate.Up(cfg.DB.URL, migrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	// Allow deploy scripts to run migrations separately before service start.
	if os.Getenv("MIGRATE_ONLY") == "true" {
		log.Println("migrations complete")
		return
	}

	// --- JWT ---
	jwtMgr := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.TTLSeconds)

	// --- Repositories ---
	orgRepo := pgxrepo.NewOrganizationRepo(pool)
	userRepo := pgxrepo.NewUserRepo(pool)
	grantRepo := pgxrepo.NewGrantRepo(pool)
	scoreRepo := pgxrepo.NewScoringRepo(pool)
	appRepo := pgxrepo.NewApplicationRepo(pool)
	leadRepo := pgxrepo.NewLeadRepo(pool)
	analyticsRepo := pgxrepo.NewAnalyticsRepo(pool)
	productRepo := pgxrepo.NewProductRepo(pool)

	// --- AI ---
	embedSvc := embedding.NewService(cfg.Embedding.OpenAIKey, cfg.Embedding.Model, cfg.Embedding.Dimensions)
	ragEngine := rag.NewEngine(grantRepo, embedSvc.Embed)

	var claudeClient *claude.Client
	if cfg.Claude.APIKey != "" {
		claudeClient, err = claude.NewClient(cfg)
		if err != nil {
			log.Printf("warn: claude client unavailable: %v", err)
		}
	}

	// --- Scoring Engine ---
	scoringEngine := scoring.NewEngine()

	// --- Services ---
	emailSvc := service.NewEmailService(cfg.Email)
	authSvc := service.NewAuthService(userRepo, jwtMgr, cfg.Firebase.WebAPIKey, cfg.App.FrontendURL, emailSvc)
	grantSvc := service.NewGrantService(grantRepo, embedSvc, ragEngine)
	scoringSvc := service.NewScoringService(orgRepo, grantRepo, scoreRepo, scoringEngine)
	orgSvc := service.NewOrgService(orgRepo, embedSvc)
	appSvc := service.NewApplicationService(appRepo, grantRepo, scoreRepo)
	leadSvc := service.NewLeadService(leadRepo)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo)
	productSvc := service.NewProductService(productRepo)

	var narrativeSvc *service.NarrativeService
	if claudeClient != nil {
		narrativeSvc = service.NewNarrativeService(orgRepo, grantRepo, scoreRepo, appRepo, claudeClient, grantSvc)
	}

	// --- Handlers ---
	r := router.New(router.Deps{
		Config:      cfg,
		JWTManager:  jwtMgr,
		Auth:        handler.NewAuthHandler(authSvc, userRepo),
		Grant:       handler.NewGrantHandler(grantSvc),
		Score:       handler.NewScoringHandler(scoringSvc),
		Narr:        handler.NewNarrativeHandler(narrativeSvc),
		Org:         handler.NewOrgHandler(orgSvc, userRepo),
		Application: handler.NewApplicationHandler(appSvc),
		Lead:        handler.NewLeadHandler(leadSvc),
		Analytics:   handler.NewAnalyticsHandler(analyticsSvc),
		QuizDraft:   handler.NewQuizDraftHandler(userRepo),
		Product:     handler.NewProductHandler(productSvc),
	})

	// --- HTTP Server ---
	port := cfg.App.Port
	if port == 0 {
		port = 8080
	}
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("readygeneration-api listening on :%d", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
