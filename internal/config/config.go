package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	App       AppConfig
	DB        DBConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Claude    ClaudeConfig
	Embedding EmbeddingConfig
	Firebase  FirebaseConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
	Email     EmailConfig
}

type FirebaseConfig struct {
	WebAPIKey string
}

type AppConfig struct {
	Name        string
	Env         string // development, staging, production
	Port        int
	Debug       bool
	BaseURL     string
	FrontendURL string
}

type DBConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	MigrationsDir   string
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret        string
	Issuer        string
	TTLSeconds    int
	RefreshTTLDay int
}

type ClaudeConfig struct {
	APIKey              string
	DefaultModel        string
	MaxTokens           int
	EmbeddingModel      string
	EmbeddingDimensions int
}

type EmbeddingConfig struct {
	OpenAIKey  string
	Model      string
	Dimensions int
	BatchSize  int
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	RequestsPerMinute int64
	BurstSize         int64
}

type EmailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	Enabled  bool
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// App defaults
	v.SetDefault("app.name", "readygeneration-backend")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.debug", false)
	v.SetDefault("app.base_url", "http://localhost:8080")
	v.SetDefault("app.frontend_url", "http://127.0.0.1:4200")

	// DB defaults
	v.SetDefault("db.max_open_conns", 25)
	v.SetDefault("db.max_idle_conns", 5)
	v.SetDefault("db.conn_max_lifetime", "15m")
	v.SetDefault("db.migrations_dir", "migrations")

	// Redis defaults
	v.SetDefault("redis.db", 0)

	// JWT defaults
	v.SetDefault("jwt.issuer", "readygeneration")
	v.SetDefault("jwt.ttl_seconds", 3600)
	v.SetDefault("jwt.refresh_ttl_days", 30)

	// Claude defaults
	v.SetDefault("claude.default_model", "claude-3-5-sonnet-20241022")
	v.SetDefault("claude.max_tokens", 4096)
	v.SetDefault("claude.embedding_model", "text-embedding-3-small")
	v.SetDefault("claude.embedding_dimensions", 1536)

	// Embedding defaults
	v.SetDefault("embedding.dimensions", 1536)
	v.SetDefault("embedding.batch_size", 100)

	// CORS defaults
	v.SetDefault("cors.allowed_origins", "http://localhost:3000,https://readygeneration.com")

	// Firebase defaults
	v.SetDefault("firebase.web_api_key", "")

	// Rate limit defaults
	v.SetDefault("rate_limit.requests_per_minute", 60)
	v.SetDefault("rate_limit.burst_size", 10)

	// Email defaults
	v.SetDefault("email.host", "smtp.gmail.com")
	v.SetDefault("email.port", 587)
	v.SetDefault("email.user", "")
	v.SetDefault("email.password", "")
	v.SetDefault("email.from", "responseiot@gmail.com")
	v.SetDefault("email.enabled", false)

	connMaxLifetime, err := time.ParseDuration(v.GetString("db.conn_max_lifetime"))
	if err != nil {
		connMaxLifetime = 15 * time.Minute
	}

	dbURL := v.GetString("DATABASE_URL")
	if dbURL == "" {
		dbURL = v.GetString("db.url")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := v.GetString("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = v.GetString("jwt.secret")
	}
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	claudeKey := v.GetString("ANTHROPIC_API_KEY")
	if claudeKey == "" {
		claudeKey = v.GetString("claude.api_key")
	}

	redisURL := v.GetString("REDIS_URL")
	if redisURL == "" {
		redisURL = v.GetString("redis.url")
		if redisURL == "" {
			redisURL = "redis://localhost:6379"
		}
	}

	originsRaw := v.GetString("CORS_ALLOWED_ORIGINS")
	if originsRaw == "" {
		originsRaw = v.GetString("cors.allowed_origins")
	}
	origins := strings.Split(originsRaw, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &Config{
		App: AppConfig{
			Name:        v.GetString("app.name"),
			Env:         v.GetString("APP_ENV"),
			Port:        v.GetInt("PORT"),
			Debug:       v.GetBool("app.debug"),
			BaseURL:     v.GetString("app.base_url"),
			FrontendURL: v.GetString("app.frontend_url"),
		},
		DB: DBConfig{
			URL:             dbURL,
			MaxOpenConns:    v.GetInt("db.max_open_conns"),
			MaxIdleConns:    v.GetInt("db.max_idle_conns"),
			ConnMaxLifetime: connMaxLifetime,
			MigrationsDir:   v.GetString("MIGRATIONS_DIR"),
		},
		Redis: RedisConfig{
			URL:      redisURL,
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
		},
		JWT: JWTConfig{
			Secret:        jwtSecret,
			Issuer:        v.GetString("jwt.issuer"),
			TTLSeconds:    v.GetInt("JWT_TTL_SECONDS"),
			RefreshTTLDay: v.GetInt("jwt.refresh_ttl_days"),
		},
		Claude: ClaudeConfig{
			APIKey:              claudeKey,
			DefaultModel:        v.GetString("claude.default_model"),
			MaxTokens:           v.GetInt("claude.max_tokens"),
			EmbeddingModel:      v.GetString("claude.embedding_model"),
			EmbeddingDimensions: v.GetInt("claude.embedding_dimensions"),
		},
		Embedding: EmbeddingConfig{
			OpenAIKey:  v.GetString("OPENAI_API_KEY"),
			Model:      v.GetString("claude.embedding_model"),
			Dimensions: v.GetInt("embedding.dimensions"),
			BatchSize:  v.GetInt("embedding.batch_size"),
		},
		Firebase: FirebaseConfig{
			WebAPIKey: v.GetString("FIREBASE_API_KEY"),
		},
		CORS: CORSConfig{
			AllowedOrigins: origins,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: v.GetInt64("rate_limit.requests_per_minute"),
			BurstSize:         v.GetInt64("rate_limit.burst_size"),
		},
		Email: EmailConfig{
			Host:     v.GetString("email.host"),
			Port:     v.GetInt("email.port"),
			User:     v.GetString("email.user"),
			Password: v.GetString("email.password"),
			From:     v.GetString("email.from"),
			Enabled:  v.GetBool("email.enabled"),
		},
	}, nil
}

// IsProd returns true when running in production.
func (c *Config) IsProd() bool {
	return c.App.Env == "production"
}
