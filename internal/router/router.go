// Package router wires all Gin routes and middleware.
package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/readygeneration/readygeneration-backend/internal/config"
	"github.com/readygeneration/readygeneration-backend/internal/handler"
	"github.com/readygeneration/readygeneration-backend/internal/middleware"
	"github.com/readygeneration/readygeneration-backend/pkg/jwt"
)

// Deps bundles all handler dependencies.
type Deps struct {
	Config      *config.Config
	JWTManager  *jwt.Manager
	Auth        *handler.AuthHandler
	Grant       *handler.GrantHandler
	Score       *handler.ScoringHandler
	Narr        *handler.NarrativeHandler
	Org         *handler.OrgHandler
	Application *handler.ApplicationHandler
	Lead        *handler.LeadHandler
	Analytics   *handler.AnalyticsHandler
	QuizDraft   *handler.QuizDraftHandler
	Product     *handler.ProductHandler
}

// New builds and returns the configured Gin engine.
func New(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS
	corsOrigins := deps.Config.CORS.AllowedOrigins
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"*"}
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Root
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Application up and running"})
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Public routes (no auth) ---
	pub := r.Group("/api/v1")
	{
		pub.POST("/auth/signup", deps.Auth.Signup)
		pub.POST("/auth/login", deps.Auth.Login)
		pub.POST("/auth/google", deps.Auth.Google)
		pub.POST("/auth/reset-password", deps.Auth.ResetPassword)
		pub.POST("/leads", deps.Lead.CaptureLead) // quiz / landing page capture
	}

	// --- Authenticated routes ---
	api := r.Group("/api/v1")
	api.Use(middleware.Auth(deps.JWTManager))
	{
		// Auth
		api.GET("/auth/me", deps.Auth.Me)
		api.POST("/auth/change-password", deps.Auth.ChangePassword)

		// Grants (read)
		grants := api.Group("/grants")
		{
			grants.GET("", deps.Grant.ListGrants)
			grants.GET("/search", deps.Grant.SearchGrants)
			grants.GET("/semantic-search", deps.Grant.SemanticSearchGrants)
			grants.GET("/:id", deps.Grant.GetGrant)
		}

		// Organizations
		orgs := api.Group("/orgs")
		{
			orgs.POST("", deps.Org.CreateOrg)
			orgs.GET("/:id", deps.Org.GetOrg)
			orgs.PATCH("/:id", deps.Org.UpdateOrg)
			orgs.GET("/:id/profile", deps.Org.GetProfile)
			orgs.PUT("/:id/profile", deps.Org.UpsertProfile)

			// Preparedness solutions selection (Funding OS wizard)
			orgs.GET("/:id/product-selection", deps.Product.ListProductSelections)
			orgs.PUT("/:id/product-selection", deps.Product.SaveProductSelection)

			// Applications scoped to org
			orgs.POST("/:id/applications", deps.Application.CreateApplication)
			orgs.GET("/:id/applications", deps.Application.ListApplications)

			// Scoring & narratives scoped to org
			orgs.GET("/:id/top-grants", deps.Score.ListTopGrants)
			orgs.POST("/:id/score-all", deps.Score.ScoreAllGrants)
			orgs.GET("/:id/grants/:grant_id/score", deps.Score.GetScore)
			orgs.POST("/:id/grants/:grant_id/score", deps.Score.ComputeScore)
			orgs.POST("/:id/grants/:grant_id/narratives", deps.Narr.GenerateNarrative)
		}

		// Preparedness solutions catalog (Funding OS wizard)
		products := api.Group("/products")
		{
			products.GET("", deps.Product.ListProducts)
			products.GET("/:id", deps.Product.GetProduct)
		}

		// Quiz draft
		quiz := api.Group("/quiz")
		{
			quiz.GET("/draft", deps.QuizDraft.GetDraft)
			quiz.PUT("/draft", deps.QuizDraft.SaveDraft)
			quiz.DELETE("/draft", deps.QuizDraft.DeleteDraft)
		}

		// Applications (standalone — for direct lookup)
		apps := api.Group("/applications")
		{
			apps.GET("/:id", deps.Application.GetApplication)
			apps.PATCH("/:id", deps.Application.UpdateApplication)
			apps.PATCH("/:id/status", deps.Application.UpdateStatus)
			apps.GET("/:id/activities", deps.Application.ListActivities)
			apps.GET("/:id/narratives", deps.Application.ListNarratives)
		}
	}

	// --- Admin routes (admin/super_admin role required) ---
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.Auth(deps.JWTManager))
	admin.Use(middleware.RequireRole("admin", "super_admin"))
	{
		// Grant management
		admin.POST("/grants", deps.Grant.CreateGrant)
		admin.PATCH("/grants/:id", deps.Grant.UpdateGrant)
		admin.DELETE("/grants/:id", deps.Grant.ArchiveGrant)
		admin.POST("/grants/:id/nofo", deps.Grant.IngestNOFO)

		// Lead management
		admin.GET("/leads", deps.Lead.ListLeads)
		admin.GET("/leads/:id", deps.Lead.GetLead)
		admin.PATCH("/leads/:id", deps.Lead.UpdateLead)
		admin.POST("/leads/:id/convert", deps.Lead.ConvertLead)
		admin.GET("/leads/:id/activities", deps.Lead.ListLeadActivities)

		// Platform analytics
		admin.GET("/stats", deps.Analytics.GetPlatformStats)

		// User management
		admin.GET("/users", deps.Auth.ListUsers)
		admin.PATCH("/users/:id/role", deps.Auth.UpdateUserRole)
	}

	return r
}
