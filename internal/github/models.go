package github

import "time"

// Repository represents a GitHub repository with enriched status.
type Repository struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	FullName      string     `json:"full_name"`
	Owner         string     `json:"owner"`
	Description   string     `json:"description"`
	HTMLURL       string     `json:"html_url"`
	Language      string     `json:"language"`
	Stars         int        `json:"stars"`
	Forks         int        `json:"forks"`
	OpenIssues    int        `json:"open_issues"`
	DefaultBranch string     `json:"default_branch"`
	IsPrivate     bool       `json:"is_private"`
	IsArchived    bool       `json:"is_archived"`
	UpdatedAt     time.Time  `json:"updated_at"`
	PushedAt      time.Time  `json:"pushed_at"`
	LatestCommit  *Commit    `json:"latest_commit,omitempty"`
	LatestRelease *Release   `json:"latest_release,omitempty"`
	WorkflowRuns  []Run      `json:"workflow_runs,omitempty"`
	Workflows     []Workflow `json:"workflows,omitempty"`
}

// Commit represents the latest commit on a branch.
type Commit struct {
	SHA         string    `json:"sha"`
	ShortSHA    string    `json:"short_sha"`
	Message     string    `json:"message"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	AuthorDate  time.Time `json:"author_date"`
	HTMLURL     string    `json:"html_url"`
}

// Release represents the latest release or tag.
type Release struct {
	TagName      string    `json:"tag_name"`
	Name         string    `json:"name"`
	PublishedAt  time.Time `json:"published_at"`
	IsDraft      bool      `json:"is_draft"`
	IsPrerelease bool      `json:"is_prerelease"`
	HTMLURL      string    `json:"html_url"`
}

// Workflow represents a GitHub Actions workflow definition.
type Workflow struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
	URL   string `json:"url"`
}

// Run represents a single GitHub Actions workflow run.
type Run struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	WorkflowID   int64     `json:"workflow_id"`
	RepoName     string    `json:"repo_name"`
	RepoOwner    string    `json:"repo_owner"`
	HeadBranch   string    `json:"head_branch"`
	HeadSHA      string    `json:"head_sha"`
	Event        string    `json:"event"`
	Status       string    `json:"status"`     // queued, in_progress, completed
	Conclusion   string    `json:"conclusion"` // success, failure, cancelled, skipped, timed_out
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	RunStartedAt time.Time `json:"run_started_at"`
	DurationSec  int64     `json:"duration_sec"`
	Actor        string    `json:"actor"`
	ActorAvatar  string    `json:"actor_avatar"`
	CommitMsg    string    `json:"commit_msg"`
}

// Job represents a job inside a workflow run.
type Job struct {
	ID          int64     `json:"id"`
	RunID       int64     `json:"run_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Steps       []Step    `json:"steps"`
	HTMLURL     string    `json:"html_url"`
}

// Step represents an individual step in a workflow job.
type Step struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	Number      int       `json:"number"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// SpaceOverview summarizes entire GitHub account or organization status.
type SpaceOverview struct {
	Owner           string       `json:"owner"`
	OwnerAvatar     string       `json:"owner_avatar"`
	TotalRepos      int          `json:"total_repos"`
	ActivePipelines int          `json:"active_pipelines"`
	FailedPipelines int          `json:"failed_pipelines"`
	SuccessRate     float64      `json:"success_rate"`
	TotalStars      int          `json:"total_stars"`
	LastRefreshed   time.Time    `json:"last_refreshed"`
	Repositories    []Repository `json:"repositories"`
	RecentRuns      []Run        `json:"recent_runs"`
}
