package service

import (
	"context"

	"github.com/readygeneration/readygeneration-backend/internal/repository"
)

// AnalyticsService wraps analytics event and usage logging.
type AnalyticsService struct {
	analytics repository.AnalyticsRepo
}

// NewAnalyticsService creates an AnalyticsService.
func NewAnalyticsService(analytics repository.AnalyticsRepo) *AnalyticsService {
	return &AnalyticsService{analytics: analytics}
}

// LogEvent records a platform event (fire-and-forget; errors are non-fatal).
func (s *AnalyticsService) LogEvent(ctx context.Context, params repository.LogEventParams) {
	_ = s.analytics.LogEvent(ctx, params)
}

// LogAIUsage records an AI model invocation for billing and monitoring.
func (s *AnalyticsService) LogAIUsage(ctx context.Context, params repository.LogAIUsageParams) {
	_ = s.analytics.LogAIUsage(ctx, params)
}

// GetPlatformStats returns aggregate counts for the admin dashboard.
func (s *AnalyticsService) GetPlatformStats(ctx context.Context) (*repository.PlatformStats, error) {
	return s.analytics.GetPlatformStats(ctx)
}
