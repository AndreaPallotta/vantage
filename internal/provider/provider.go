package provider

import (
	"context"

	"github.com/AndreaPallotta/vantage/internal/models"
)

// Provider defines operations for a Git & CI/CD hosting platform (GitHub, GitLab public/self-hosted).
type Provider interface {
	ID() string
	Name() string
	Platform() models.Platform
	Namespace() string
	GetOverview(ctx context.Context, includeForks, includeArchived bool) (*models.SpaceOverview, error)
	ListRepositories(ctx context.Context, includeForks, includeArchived bool) ([]models.Repository, error)
	ListPipelines(ctx context.Context, repo string, limit int) ([]models.Run, error)
	GetRunJobs(ctx context.Context, repo string, runID int64) ([]models.Job, error)
	TriggerPipeline(ctx context.Context, repo string, ref string, inputs map[string]interface{}) error
	RetryPipeline(ctx context.Context, repo string, runID int64) error
	CancelPipeline(ctx context.Context, repo string, runID int64) error
}
