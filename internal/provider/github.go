package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/AndreaPallotta/vantage/internal/models"
)

// GitHubProvider implements Provider for GitHub.
type GitHubProvider struct {
	config     models.SpaceConfig
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewGitHubProvider creates a new GitHub provider instance.
func NewGitHubProvider(cfg models.SpaceConfig, token string) *GitHubProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubProvider{
		config: cfg,
		token:  token,
		httpClient: &http.Client{
			Timeout: 25 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (p *GitHubProvider) ID() string             { return p.config.ID }
func (p *GitHubProvider) Name() string           { return p.config.Name }
func (p *GitHubProvider) Platform() models.Platform { return models.PlatformGitHub }
func (p *GitHubProvider) Namespace() string      { return p.config.Namespace }

func (p *GitHubProvider) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	url := p.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Vantage-Cockpit/1.0")

	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (p *GitHubProvider) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("github api error (%d %s): %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return resp, err
		}
	}

	return resp, nil
}

func (p *GitHubProvider) ListRepositories(ctx context.Context, includeForks, includeArchived bool) ([]models.Repository, error) {
	var allRepos []models.Repository
	page := 1
	owner := p.config.Namespace
	if owner == "" {
		owner = "AndreaPallotta"
	}

	for {
		path := fmt.Sprintf("/users/%s/repos?per_page=100&page=%d&sort=pushed&direction=desc", owner, page)
		req, err := p.newRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		type rawRepo struct {
			ID              int64     `json:"id"`
			Name            string    `json:"name"`
			FullName        string    `json:"full_name"`
			Owner           struct{ Login string `json:"login"` } `json:"owner"`
			Description     string    `json:"description"`
			HTMLURL         string    `json:"html_url"`
			Language        string    `json:"language"`
			StargazersCount int       `json:"stargazers_count"`
			ForksCount      int       `json:"forks_count"`
			OpenIssuesCount int       `json:"open_issues_count"`
			DefaultBranch   string    `json:"default_branch"`
			Private         bool      `json:"private"`
			Fork            bool      `json:"fork"`
			Archived        bool      `json:"archived"`
			UpdatedAt       time.Time `json:"updated_at"`
			PushedAt        time.Time `json:"pushed_at"`
		}

		var rawList []rawRepo
		_, err = p.do(req, &rawList)
		if err != nil {
			if page == 1 {
				orgPath := fmt.Sprintf("/orgs/%s/repos?per_page=100&page=%d&sort=pushed&direction=desc", owner, page)
				orgReq, orgErr := p.newRequest(ctx, "GET", orgPath, nil)
				if orgErr == nil {
					if _, orgErr = p.do(orgReq, &rawList); orgErr != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				break
			}
		}

		if len(rawList) == 0 {
			break
		}

		for _, r := range rawList {
			if !includeForks && r.Fork {
				continue
			}
			if !includeArchived && r.Archived {
				continue
			}
			allRepos = append(allRepos, models.Repository{
				ID:            fmt.Sprintf("%d", r.ID),
				Platform:      models.PlatformGitHub,
				SpaceID:       p.config.ID,
				SpaceName:     p.config.Name,
				Name:          r.Name,
				FullName:      r.FullName,
				Owner:         r.Owner.Login,
				Description:   r.Description,
				HTMLURL:       r.HTMLURL,
				Language:      r.Language,
				Stars:         r.StargazersCount,
				Forks:         r.ForksCount,
				OpenIssues:    r.OpenIssuesCount,
				DefaultBranch: r.DefaultBranch,
				IsPrivate:     r.Private,
				IsArchived:    r.Archived,
				UpdatedAt:     r.UpdatedAt,
				PushedAt:      r.PushedAt,
			})
		}

		if len(rawList) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}

func (p *GitHubProvider) getLatestCommit(ctx context.Context, owner, repo, branch string) (*models.Commit, error) {
	if branch == "" {
		branch = "main"
	}
	path := fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, branch)
	req, err := p.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type rawCommitResp struct {
		SHA     string `json:"sha"`
		HTMLURL string `json:"html_url"`
		Commit  struct {
			Message string `json:"message"`
			Author  struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}

	var raw rawCommitResp
	if _, err := p.do(req, &raw); err != nil {
		return nil, err
	}

	short := raw.SHA
	if len(short) > 7 {
		short = short[:7]
	}

	return &models.Commit{
		SHA:         raw.SHA,
		ShortSHA:    short,
		Message:     raw.Commit.Message,
		AuthorName:  raw.Commit.Author.Name,
		AuthorEmail: raw.Commit.Author.Email,
		AuthorDate:  raw.Commit.Author.Date,
		HTMLURL:     raw.HTMLURL,
	}, nil
}

func (p *GitHubProvider) getLatestRelease(ctx context.Context, owner, repo string) (*models.Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo)
	req, err := p.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var rel models.Release
	if _, err := p.do(req, &rel); err != nil {
		tagsPath := fmt.Sprintf("/repos/%s/%s/tags?per_page=1", owner, repo)
		if tagsReq, tagsErr := p.newRequest(ctx, "GET", tagsPath, nil); tagsErr == nil {
			type tagItem struct { Name string `json:"name"` }
			var tags []tagItem
			if _, tagErr := p.do(tagsReq, &tags); tagErr == nil && len(tags) > 0 {
				return &models.Release{
					TagName: tags[0].Name,
					Name:    tags[0].Name,
					HTMLURL: fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, tags[0].Name),
				}, nil
			}
		}
		return nil, nil
	}

	return &rel, nil
}

func (p *GitHubProvider) ListPipelines(ctx context.Context, repo string, limit int) ([]models.Run, error) {
	if limit <= 0 {
		limit = 10
	}
	owner := p.config.Namespace
	path := fmt.Sprintf("/repos/%s/%s/actions/runs?per_page=%d", owner, repo, limit)
	req, err := p.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type rawRun struct {
		ID           int64     `json:"id"`
		Name         string    `json:"name"`
		WorkflowID   int64     `json:"workflow_id"`
		HeadBranch   string    `json:"head_branch"`
		HeadSHA      string    `json:"head_sha"`
		Event        string    `json:"event"`
		Status       string    `json:"status"`
		Conclusion   string    `json:"conclusion"`
		HTMLURL      string    `json:"html_url"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		RunStartedAt time.Time `json:"run_started_at"`
		Actor        struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
		} `json:"actor"`
		HeadCommit struct {
			Message string `json:"message"`
		} `json:"head_commit"`
	}

	type runsResp struct {
		WorkflowRuns []rawRun `json:"workflow_runs"`
	}

	var resp runsResp
	if _, err := p.do(req, &resp); err != nil {
		return nil, err
	}

	var runs []models.Run
	for _, r := range resp.WorkflowRuns {
		var dur int64
		if !r.RunStartedAt.IsZero() && !r.UpdatedAt.IsZero() {
			dur = int64(r.UpdatedAt.Sub(r.RunStartedAt).Seconds())
			if dur < 0 {
				dur = 0
			}
		}

		runs = append(runs, models.Run{
			ID:           r.ID,
			Platform:     models.PlatformGitHub,
			SpaceID:      p.config.ID,
			Name:         r.Name,
			WorkflowID:   fmt.Sprintf("%d", r.WorkflowID),
			RepoName:     repo,
			RepoOwner:    owner,
			HeadBranch:   r.HeadBranch,
			HeadSHA:      r.HeadSHA,
			Event:        r.Event,
			Status:       r.Status,
			Conclusion:   r.Conclusion,
			HTMLURL:      r.HTMLURL,
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
			RunStartedAt: r.RunStartedAt,
			DurationSec:  dur,
			Actor:        r.Actor.Login,
			ActorAvatar:  r.Actor.AvatarURL,
			CommitMsg:    r.HeadCommit.Message,
		})
	}

	return runs, nil
}

func (p *GitHubProvider) TriggerPipeline(ctx context.Context, repo string, ref string, inputs map[string]interface{}) error {
	if ref == "" {
		ref = "main"
	}
	owner := p.config.Namespace
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/release.yml/dispatches", owner, repo)
	payload := map[string]interface{}{"ref": ref}
	if len(inputs) > 0 {
		payload["inputs"] = inputs
	}

	req, err := p.newRequest(ctx, "POST", path, payload)
	if err != nil {
		return err
	}

	_, err = p.do(req, nil)
	return err
}

func (p *GitHubProvider) RetryPipeline(ctx context.Context, repo string, runID int64) error {
	owner := p.config.Namespace
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/rerun", owner, repo, runID)
	req, err := p.newRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	_, err = p.do(req, nil)
	return err
}

func (p *GitHubProvider) CancelPipeline(ctx context.Context, repo string, runID int64) error {
	owner := p.config.Namespace
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/cancel", owner, repo, runID)
	req, err := p.newRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	_, err = p.do(req, nil)
	return err
}

func (p *GitHubProvider) GetOverview(ctx context.Context, includeForks, includeArchived bool) (*models.SpaceOverview, error) {
	repos, err := p.ListRepositories(ctx, includeForks, includeArchived)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8)
	enrichedRepos := make([]models.Repository, len(repos))
	copy(enrichedRepos, repos)

	var allRuns []models.Run
	var runsMutex sync.Mutex

	for i := range enrichedRepos {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			r := &enrichedRepos[idx]
			if commit, err := p.getLatestCommit(ctx, r.Owner, r.Name, r.DefaultBranch); err == nil {
				r.LatestCommit = commit
			}
			if rel, err := p.getLatestRelease(ctx, r.Owner, r.Name); err == nil && rel != nil {
				r.LatestRelease = rel
			}
			if runs, err := p.ListPipelines(ctx, r.Name, 5); err == nil {
				r.WorkflowRuns = runs
				runsMutex.Lock()
				allRuns = append(allRuns, runs...)
				runsMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	totalStars := 0
	activeCount := 0
	failedCount := 0
	successCount := 0

	for _, r := range enrichedRepos {
		totalStars += r.Stars
		if len(r.WorkflowRuns) > 0 {
			latestRun := r.WorkflowRuns[0]
			if latestRun.Status == "in_progress" || latestRun.Status == "queued" {
				activeCount++
			} else if latestRun.Conclusion == "failure" || latestRun.Conclusion == "timed_out" || latestRun.Conclusion == "failed" {
				failedCount++
			} else if latestRun.Conclusion == "success" {
				successCount++
			}
		}
	}

	successRate := 100.0
	if totalFinished := successCount + failedCount; totalFinished > 0 {
		successRate = float64(successCount) / float64(totalFinished) * 100.0
	}

	return &models.SpaceOverview{
		SpaceID:         p.config.ID,
		SpaceName:       p.config.Name,
		Platform:        models.PlatformGitHub,
		Owner:           p.config.Namespace,
		TotalRepos:      len(enrichedRepos),
		ActivePipelines: activeCount,
		FailedPipelines: failedCount,
		SuccessRate:     successRate,
		TotalStars:      totalStars,
		LastRefreshed:   time.Now(),
		Repositories:    enrichedRepos,
		RecentRuns:      allRuns,
	}, nil
}
