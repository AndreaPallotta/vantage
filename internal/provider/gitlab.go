package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/AndreaPallotta/vantage/internal/models"
)

// GitLabProvider implements Provider for public and self-hosted GitLab instances.
type GitLabProvider struct {
	config     models.SpaceConfig
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewGitLabProvider creates a new GitLab provider.
func NewGitLabProvider(cfg models.SpaceConfig, token string) *GitLabProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://gitlab.com/api/v4"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api/v4") && !strings.Contains(baseURL, "/api/v4") {
		baseURL = baseURL + "/api/v4"
	}

	return &GitLabProvider{
		config: cfg,
		token:  token,
		httpClient: &http.Client{
			Timeout: 25 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (p *GitLabProvider) ID() string             { return p.config.ID }
func (p *GitLabProvider) Name() string           { return p.config.Name }
func (p *GitLabProvider) Platform() models.Platform { return models.PlatformGitLab }
func (p *GitLabProvider) Namespace() string      { return p.config.Namespace }

func (p *GitLabProvider) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	fullURL := p.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Vantage-Cockpit/1.0")

	if p.token != "" {
		req.Header.Set("PRIVATE-TOKEN", p.token)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (p *GitLabProvider) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("gitlab api error (%d %s): %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return resp, err
		}
	}

	return resp, nil
}

type glProject struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	PathWithNamespace string    `json:"path_with_namespace"`
	Description       string    `json:"description"`
	WebURL            string    `json:"web_url"`
	DefaultBranch     string    `json:"default_branch"`
	StarCount         int       `json:"star_count"`
	ForksCount        int       `json:"forks_count"`
	OpenIssuesCount   int       `json:"open_issues_count"`
	Visibility        string    `json:"visibility"`
	Archived          bool      `json:"archived"`
	LastActivityAt    time.Time `json:"last_activity_at"`
	CreatedAt         time.Time `json:"created_at"`
	Namespace         struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"namespace"`
}

func (p *GitLabProvider) ListRepositories(ctx context.Context, includeForks, includeArchived bool) ([]models.Repository, error) {
	var allRepos []models.Repository
	namespace := p.config.Namespace
	page := 1

	encodedNamespace := url.PathEscape(namespace)

	for {
		// Attempt 1: Group projects (with subgroups if requested)
		subgroupsParam := "true"
		if !p.config.IncludeSubgroups {
			subgroupsParam = "false"
		}
		path := fmt.Sprintf("/groups/%s/projects?include_subgroups=%s&per_page=100&page=%d&order_by=last_activity_at", encodedNamespace, subgroupsParam, page)
		req, err := p.newRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var projects []glProject
		_, err = p.do(req, &projects)
		if err != nil {
			// Attempt 2: User projects if group fails
			if page == 1 {
				userPath := fmt.Sprintf("/users/%s/projects?per_page=100&page=%d&order_by=last_activity_at", encodedNamespace, page)
				userReq, userErr := p.newRequest(ctx, "GET", userPath, nil)
				if userErr == nil {
					if _, userErr = p.do(userReq, &projects); userErr != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				break
			}
		}

		if len(projects) == 0 {
			break
		}

		for _, prj := range projects {
			if !includeArchived && prj.Archived {
				continue
			}

			branch := prj.DefaultBranch
			if branch == "" {
				branch = "main"
			}

			allRepos = append(allRepos, models.Repository{
				ID:            fmt.Sprintf("%d", prj.ID),
				Platform:      models.PlatformGitLab,
				SpaceID:       p.config.ID,
				SpaceName:     p.config.Name,
				Name:          prj.Name,
				FullName:      prj.PathWithNamespace,
				Owner:         prj.Namespace.Name,
				Description:   prj.Description,
				HTMLURL:       prj.WebURL,
				Stars:         prj.StarCount,
				Forks:         prj.ForksCount,
				OpenIssues:    prj.OpenIssuesCount,
				DefaultBranch: branch,
				IsPrivate:     prj.Visibility == "private",
				IsArchived:    prj.Archived,
				UpdatedAt:     prj.LastActivityAt,
				PushedAt:      prj.LastActivityAt,
			})
		}

		if len(projects) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}

func (p *GitLabProvider) getLatestCommit(ctx context.Context, projectID string, branch string) (*models.Commit, error) {
	path := fmt.Sprintf("/projects/%s/repository/commits?ref_name=%s&per_page=1", projectID, branch)
	req, err := p.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type glCommit struct {
		ID             string    `json:"id"`
		ShortID        string    `json:"short_id"`
		Title          string    `json:"title"`
		Message        string    `json:"message"`
		AuthorName     string    `json:"author_name"`
		AuthorEmail    string    `json:"author_email"`
		AuthoredDate   time.Time `json:"authored_date"`
		WebURL         string    `json:"web_url"`
	}

	var commits []glCommit
	if _, err := p.do(req, &commits); err != nil || len(commits) == 0 {
		return nil, err
	}

	c := commits[0]
	return &models.Commit{
		SHA:         c.ID,
		ShortSHA:    c.ShortID,
		Message:     c.Title,
		AuthorName:  c.AuthorName,
		AuthorEmail: c.AuthorEmail,
		AuthorDate:  c.AuthoredDate,
		HTMLURL:     c.WebURL,
	}, nil
}

func (p *GitLabProvider) getLatestRelease(ctx context.Context, projectID string) (*models.Release, error) {
	path := fmt.Sprintf("/projects/%s/releases?per_page=1", projectID)
	req, err := p.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type glRelease struct {
		TagName     string    `json:"tag_name"`
		Name        string    `json:"name"`
		ReleasedAt  time.Time `json:"released_at"`
		UpcomingRelease bool  `json:"upcoming_release"`
		Links struct {
			Self string `json:"self"`
		} `json:"_links"`
	}

	var releases []glRelease
	if _, err := p.do(req, &releases); err == nil && len(releases) > 0 {
		r := releases[0]
		return &models.Release{
			TagName:     r.TagName,
			Name:        r.Name,
			PublishedAt: r.ReleasedAt,
			HTMLURL:     r.Links.Self,
		}, nil
	}

	// Fallback to tags
	tagsPath := fmt.Sprintf("/projects/%s/repository/tags?per_page=1", projectID)
	if tagReq, err := p.newRequest(ctx, "GET", tagsPath, nil); err == nil {
		type glTag struct {
			Name string `json:"name"`
		}
		var tags []glTag
		if _, err := p.do(tagReq, &tags); err == nil && len(tags) > 0 {
			return &models.Release{
				TagName: tags[0].Name,
				Name:    tags[0].Name,
			}, nil
		}
	}

	return nil, nil
}

func (p *GitLabProvider) ListPipelines(ctx context.Context, projectID string, limit int) ([]models.Run, error) {
	if limit <= 0 {
		limit = 10
	}
	path := fmt.Sprintf("/projects/%s/pipelines?per_page=%d", projectID, limit)
	req, err := p.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type glPipeline struct {
		ID        int64     `json:"id"`
		IID       int64     `json:"iid"`
		ProjectID int64     `json:"project_id"`
		SHA       string    `json:"sha"`
		Ref       string    `json:"ref"`
		Status    string    `json:"status"` // running, pending, success, failed, canceled, skipped
		Source    string    `json:"source"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		WebURL    string    `json:"web_url"`
		User      struct {
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
	}

	var pipelines []glPipeline
	if _, err := p.do(req, &pipelines); err != nil {
		return nil, err
	}

	var runs []models.Run
	for _, pipe := range pipelines {
		conclusion := pipe.Status
		status := "completed"

		if pipe.Status == "running" || pipe.Status == "pending" || pipe.Status == "created" {
			status = "in_progress"
		} else if pipe.Status == "failed" {
			conclusion = "failure"
		} else if pipe.Status == "canceled" {
			conclusion = "cancelled"
		}

		var dur int64
		if !pipe.CreatedAt.IsZero() && !pipe.UpdatedAt.IsZero() {
			dur = int64(pipe.UpdatedAt.Sub(pipe.CreatedAt).Seconds())
			if dur < 0 {
				dur = 0
			}
		}

		runs = append(runs, models.Run{
			ID:           pipe.ID,
			Platform:     models.PlatformGitLab,
			SpaceID:      p.config.ID,
			Name:         fmt.Sprintf("Pipeline #%d", pipe.ID),
			WorkflowID:   fmt.Sprintf("%d", pipe.ID),
			RepoName:     p.config.Namespace,
			HeadBranch:   pipe.Ref,
			HeadSHA:      pipe.SHA,
			Event:        pipe.Source,
			Status:       status,
			Conclusion:   conclusion,
			HTMLURL:      pipe.WebURL,
			CreatedAt:    pipe.CreatedAt,
			UpdatedAt:    pipe.UpdatedAt,
			RunStartedAt: pipe.CreatedAt,
			DurationSec:  dur,
			Actor:        pipe.User.Name,
			ActorAvatar:  pipe.User.AvatarURL,
			CommitMsg:    fmt.Sprintf("Triggered via %s", pipe.Source),
		})
	}

	return runs, nil
}

func (p *GitLabProvider) TriggerPipeline(ctx context.Context, projectID string, ref string, inputs map[string]interface{}) error {
	if ref == "" {
		ref = "main"
	}
	path := fmt.Sprintf("/projects/%s/pipeline", projectID)
	payload := map[string]interface{}{
		"ref": ref,
	}

	req, err := p.newRequest(ctx, "POST", path, payload)
	if err != nil {
		return err
	}

	_, err = p.do(req, nil)
	return err
}

func (p *GitLabProvider) RetryPipeline(ctx context.Context, projectID string, runID int64) error {
	path := fmt.Sprintf("/projects/%s/pipelines/%d/retry", projectID, runID)
	req, err := p.newRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	_, err = p.do(req, nil)
	return err
}

func (p *GitLabProvider) CancelPipeline(ctx context.Context, projectID string, runID int64) error {
	path := fmt.Sprintf("/projects/%s/pipelines/%d/cancel", projectID, runID)
	req, err := p.newRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}
	_, err = p.do(req, nil)
	return err
}

func (p *GitLabProvider) GetOverview(ctx context.Context, includeForks, includeArchived bool) (*models.SpaceOverview, error) {
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
			if commit, err := p.getLatestCommit(ctx, r.ID, r.DefaultBranch); err == nil {
				r.LatestCommit = commit
			}
			if rel, err := p.getLatestRelease(ctx, r.ID); err == nil && rel != nil {
				r.LatestRelease = rel
			}
			if runs, err := p.ListPipelines(ctx, r.ID, 5); err == nil {
				for j := range runs {
					runs[j].RepoName = r.Name
					runs[j].RepoOwner = r.Owner
				}
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
			} else if latestRun.Conclusion == "failure" || latestRun.Conclusion == "failed" {
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
		Platform:        models.PlatformGitLab,
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
