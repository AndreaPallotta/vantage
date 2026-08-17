package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client handles GitHub REST API operations.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient returns a new GitHub API client.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		baseURL: "https://api.github.com",
		token:   token,
	}
}

// User represents authenticated user details.
type User struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	url := c.baseURL + path
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
	req.Header.Set("User-Agent", "Vantage-Dashboard/1.0")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
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

// GetAuthenticatedUser retrieves the authenticated user details.
func (c *Client) GetAuthenticatedUser(ctx context.Context) (*User, error) {
	req, err := c.newRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}

	var user User
	_, err = c.do(req, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListRepositories lists repositories for a user/org with optional fork/archived inclusion.
func (c *Client) ListRepositories(ctx context.Context, owner string, includeForks, includeArchived bool) ([]Repository, error) {
	var allRepos []Repository
	page := 1

	for {
		path := fmt.Sprintf("/users/%s/repos?per_page=100&page=%d&sort=pushed&direction=desc", owner, page)
		req, err := c.newRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		type rawRepo struct {
			ID            int64     `json:"id"`
			Name          string    `json:"name"`
			FullName      string    `json:"full_name"`
			Owner         struct {
				Login string `json:"login"`
			} `json:"owner"`
			Description   string    `json:"description"`
			HTMLURL       string    `json:"html_url"`
			Language      string    `json:"language"`
			StargazersCount int     `json:"stargazers_count"`
			ForksCount    int       `json:"forks_count"`
			OpenIssuesCount int     `json:"open_issues_count"`
			DefaultBranch string    `json:"default_branch"`
			Private       bool      `json:"private"`
			Fork          bool      `json:"fork"`
			Archived      bool      `json:"archived"`
			UpdatedAt     time.Time `json:"updated_at"`
			PushedAt      time.Time `json:"pushed_at"`
		}

		var rawList []rawRepo
		_, err = c.do(req, &rawList)
		if err != nil {
			// Fallback: If /users/:owner fails, try /orgs/:owner
			if page == 1 {
				orgPath := fmt.Sprintf("/orgs/%s/repos?per_page=100&page=%d&sort=pushed&direction=desc", owner, page)
				orgReq, orgErr := c.newRequest(ctx, "GET", orgPath, nil)
				if orgErr == nil {
					if _, orgErr = c.do(orgReq, &rawList); orgErr != nil {
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
			allRepos = append(allRepos, Repository{
				ID:            r.ID,
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

// GetLatestCommit retrieves the latest commit for a repo on its default branch.
func (c *Client) GetLatestCommit(ctx context.Context, owner, repo, branch string) (*Commit, error) {
	if branch == "" {
		branch = "main"
	}
	path := fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, branch)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type rawCommitResp struct {
		SHA    string `json:"sha"`
		HTMLURL string `json:"html_url"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}

	var raw rawCommitResp
	_, err = c.do(req, &raw)
	if err != nil {
		return nil, err
	}

	short := raw.SHA
	if len(short) > 7 {
		short = short[:7]
	}

	return &Commit{
		SHA:         raw.SHA,
		ShortSHA:    short,
		Message:     raw.Commit.Message,
		AuthorName:  raw.Commit.Author.Name,
		AuthorEmail: raw.Commit.Author.Email,
		AuthorDate:  raw.Commit.Author.Date,
		HTMLURL:     raw.HTMLURL,
	}, nil
}

// GetLatestRelease retrieves the latest release/tag for a repo.
func (c *Client) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var rel Release
	_, err = c.do(req, &rel)
	if err != nil {
		// Fallback: check tags if no formal release published
		tagsPath := fmt.Sprintf("/repos/%s/%s/tags?per_page=1", owner, repo)
		tagsReq, tagsErr := c.newRequest(ctx, "GET", tagsPath, nil)
		if tagsErr == nil {
			type tagItem struct {
				Name string `json:"name"`
			}
			var tags []tagItem
			if _, tagErr := c.do(tagsReq, &tags); tagErr == nil && len(tags) > 0 {
				return &Release{
					TagName: tags[0].Name,
					Name:    tags[0].Name,
					HTMLURL: fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, tags[0].Name),
				}, nil
			}
		}
		return nil, nil // No release found
	}

	return &rel, nil
}

// ListWorkflows retrieves all GitHub Actions workflow definitions in a repo.
func (c *Client) ListWorkflows(ctx context.Context, owner, repo string) ([]Workflow, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows", owner, repo)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type workflowsResp struct {
		Workflows []Workflow `json:"workflows"`
	}

	var resp workflowsResp
	_, err = c.do(req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Workflows, nil
}

// ListWorkflowRuns retrieves recent workflow runs for a repo.
func (c *Client) ListWorkflowRuns(ctx context.Context, owner, repo string, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 10
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs?per_page=%d", owner, repo, limit)
	req, err := c.newRequest(ctx, "GET", path, nil)
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
	_, err = c.do(req, &resp)
	if err != nil {
		return nil, err
	}

	var runs []Run
	for _, r := range resp.WorkflowRuns {
		var dur int64
		if !r.RunStartedAt.IsZero() && !r.UpdatedAt.IsZero() {
			dur = int64(r.UpdatedAt.Sub(r.RunStartedAt).Seconds())
			if dur < 0 {
				dur = 0
			}
		}

		runs = append(runs, Run{
			ID:           r.ID,
			Name:         r.Name,
			WorkflowID:   r.WorkflowID,
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

// GetRunJobs retrieves jobs for a given workflow run.
func (c *Client) GetRunJobs(ctx context.Context, owner, repo string, runID int64) ([]Job, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/jobs", owner, repo, runID)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	type jobsResp struct {
		Jobs []Job `json:"jobs"`
	}

	var resp jobsResp
	_, err = c.do(req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Jobs, nil
}

// DispatchWorkflow triggers a workflow_dispatch event.
func (c *Client) DispatchWorkflow(ctx context.Context, owner, repo string, workflowIDOrFile string, ref string, inputs map[string]interface{}) error {
	if ref == "" {
		ref = "main"
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/dispatches", owner, repo, workflowIDOrFile)
	payload := map[string]interface{}{
		"ref": ref,
	}
	if len(inputs) > 0 {
		payload["inputs"] = inputs
	}

	req, err := c.newRequest(ctx, "POST", path, payload)
	if err != nil {
		return err
	}

	_, err = c.do(req, nil)
	return err
}

// RerunWorkflow reruns an existing workflow run.
func (c *Client) RerunWorkflow(ctx context.Context, owner, repo string, runID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/rerun", owner, repo, runID)
	req, err := c.newRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}

	_, err = c.do(req, nil)
	return err
}

// CancelWorkflowRun cancels an in-progress workflow run.
func (c *Client) CancelWorkflowRun(ctx context.Context, owner, repo string, runID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/cancel", owner, repo, runID)
	req, err := c.newRequest(ctx, "POST", path, nil)
	if err != nil {
		return err
	}

	_, err = c.do(req, nil)
	return err
}

// GetSpaceOverview enriches all repositories in the space with concurrent goroutines.
func (c *Client) GetSpaceOverview(ctx context.Context, owner string, includeForks, includeArchived bool) (*SpaceOverview, error) {
	repos, err := c.ListRepositories(ctx, owner, includeForks, includeArchived)
	if err != nil {
		return nil, err
	}

	// Concurrently enrich repositories (commits, releases, workflows, runs)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8) // Limit concurrent API calls to avoid secondary rate limits

	enrichedRepos := make([]Repository, len(repos))
	copy(enrichedRepos, repos)

	var allRecentRuns []Run
	var runsMutex sync.Mutex

	for i := range enrichedRepos {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			r := &enrichedRepos[idx]

			// Latest commit
			if commit, err := c.GetLatestCommit(ctx, r.Owner, r.Name, r.DefaultBranch); err == nil {
				r.LatestCommit = commit
			}

			// Latest release
			if rel, err := c.GetLatestRelease(ctx, r.Owner, r.Name); err == nil && rel != nil {
				r.LatestRelease = rel
			}

			// Workflows
			if wfs, err := c.ListWorkflows(ctx, r.Owner, r.Name); err == nil {
				r.Workflows = wfs
			}

			// Workflow runs
			if runs, err := c.ListWorkflowRuns(ctx, r.Owner, r.Name, 5); err == nil {
				r.WorkflowRuns = runs
				runsMutex.Lock()
				allRecentRuns = append(allRecentRuns, runs...)
				runsMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Calculate space statistics
	totalStars := 0
	activeCount := 0
	failedCount := 0
	successCount := 0

	for _, r := range enrichedRepos {
		totalStars += r.Stars
		for _, run := range r.WorkflowRuns {
			if run.Status == "in_progress" || run.Status == "queued" {
				activeCount++
			}
			if run.Conclusion == "failure" || run.Conclusion == "timed_out" {
				failedCount++
			}
			if run.Conclusion == "success" {
				successCount++
			}
		}
	}

	successRate := 100.0
	totalFinished := successCount + failedCount
	if totalFinished > 0 {
		successRate = float64(successCount) / float64(totalFinished) * 100.0
	}

	return &SpaceOverview{
		Owner:           owner,
		TotalRepos:      len(enrichedRepos),
		ActivePipelines: activeCount,
		FailedPipelines: failedCount,
		SuccessRate:     successRate,
		TotalStars:      totalStars,
		LastRefreshed:   time.Now(),
		Repositories:    enrichedRepos,
		RecentRuns:      allRecentRuns,
	}, nil
}
