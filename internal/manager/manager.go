package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/models"
	"github.com/AndreaPallotta/vantage/internal/provider"
)

// Manager coordinates multiple space providers (GitHub, GitLab public/self-hosted).
type Manager struct {
	cfg       *config.Config
	providers map[string]provider.Provider
	mu        sync.RWMutex
}

// New creates a new multi-space manager.
func New(cfg *config.Config) *Manager {
	m := &Manager{
		cfg:       cfg,
		providers: make(map[string]provider.Provider),
	}
	m.initProviders()
	return m
}

func (m *Manager) initProviders() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers = make(map[string]provider.Provider)
	for _, sc := range m.cfg.Spaces {
		token := config.ResolveToken(sc)
		if sc.Platform == models.PlatformGitLab {
			m.providers[sc.ID] = provider.NewGitLabProvider(sc, token)
		} else {
			m.providers[sc.ID] = provider.NewGitHubProvider(sc, token)
		}
	}
}

// ListSpaces returns all configured spaces.
func (m *Manager) ListSpaces() []models.SpaceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []models.SpaceInfo
	for _, sc := range m.cfg.Spaces {
		list = append(list, models.SpaceInfo{
			ID:        sc.ID,
			Name:      sc.Name,
			Platform:  sc.Platform,
			Namespace: sc.Namespace,
		})
	}
	return list
}

// GetProvider retrieves a provider by space ID.
func (m *Manager) GetProvider(spaceID string) (provider.Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.providers[spaceID]
	if !ok {
		return nil, fmt.Errorf("space provider not found: %s", spaceID)
	}
	return p, nil
}

// GetOverview aggregates space data across single or all configured providers.
func (m *Manager) GetOverview(ctx context.Context, spaceID string, includeForks, includeArchived bool) (*models.SpaceOverview, error) {
	spacesList := m.ListSpaces()

	// Single Space
	if spaceID != "" && spaceID != "all" {
		p, err := m.GetProvider(spaceID)
		if err != nil {
			return nil, err
		}
		overview, err := p.GetOverview(ctx, includeForks, includeArchived)
		if err != nil {
			return nil, err
		}
		overview.AvailableSpaces = spacesList
		return overview, nil
	}

	// Multi-Space Aggregation ("all")
	m.mu.RLock()
	allProviders := make([]provider.Provider, 0, len(m.providers))
	for _, p := range m.providers {
		allProviders = append(allProviders, p)
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allRepos []models.Repository
	var allRuns []models.Run

	totalStars := 0
	activeCount := 0
	failedCount := 0
	successCount := 0

	for _, p := range allProviders {
		wg.Add(1)
		go func(prov provider.Provider) {
			defer wg.Done()
			ov, err := prov.GetOverview(ctx, includeForks, includeArchived)
			if err != nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			allRepos = append(allRepos, ov.Repositories...)
			allRuns = append(allRuns, ov.RecentRuns...)
			totalStars += ov.TotalStars
			activeCount += ov.ActivePipelines
			failedCount += ov.FailedPipelines
		}(p)
	}

	wg.Wait()

	for _, r := range allRuns {
		if r.Conclusion == "success" {
			successCount++
		}
	}

	successRate := 100.0
	if totalFinished := successCount + failedCount; totalFinished > 0 {
		successRate = float64(successCount) / float64(totalFinished) * 100.0
	}

	return &models.SpaceOverview{
		SpaceID:         "all",
		SpaceName:       "All Spaces (Unified Fleet)",
		Platform:        "all",
		Owner:           "All Configured Spaces",
		TotalRepos:      len(allRepos),
		ActivePipelines: activeCount,
		FailedPipelines: failedCount,
		SuccessRate:     successRate,
		TotalStars:      totalStars,
		LastRefreshed:   time.Now(),
		Repositories:    allRepos,
		RecentRuns:      allRuns,
		AvailableSpaces: spacesList,
	}, nil
}
