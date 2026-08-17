package models

import "time"

// Platform represents the hosting provider type.
type Platform string

const (
	PlatformGitHub Platform = "github"
	PlatformGitLab Platform = "gitlab"
)

// SpaceConfig represents a configured provider workspace or namespace.
type SpaceConfig struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Platform         Platform `json:"platform"` // "github" | "gitlab"
	BaseURL          string   `json:"base_url,omitempty"`
	Namespace        string   `json:"namespace"`
	Token            string   `json:"token,omitempty"`
	TokenEnv         string   `json:"token_env,omitempty"`
	IncludeSubgroups bool     `json:"include_subgroups,omitempty"`
}

// Repository represents a repository or project with enriched metadata.
type Repository struct {
	ID            string     `json:"id"`
	Platform      Platform   `json:"platform"`
	SpaceID       string     `json:"space_id"`
	SpaceName     string     `json:"space_name"`
	Name          string     `json:"name"`
	FullName      string     `json:"full_name"`
	Owner         string     `json:"owner"`
	Description   string     `json:"description"`
	HTMLURL       string     `json:"html_url"`
	Language      string     `json:"language"`
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

// Commit represents the latest commit metadata.
type Commit struct {
	SHA         string    `json:"sha"`
	ShortSHA    string    `json:"short_sha"`
	Message     string    `json:"message"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	AuthorDate  time.Time `json:"author_date"`
	HTMLURL     string    `json:"html_url"`
}

// Release represents a release or tag.
type Release struct {
	TagName      string    `json:"tag_name"`
	Name         string    `json:"name"`
	PublishedAt  time.Time `json:"published_at"`
	IsDraft      bool      `json:"is_draft"`
	IsPrerelease bool      `json:"is_prerelease"`
	HTMLURL      string    `json:"html_url"`
}

// Workflow represents a CI/CD workflow/job definition.
type Workflow struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
	URL   string `json:"url"`
}

// Run represents a single CI/CD pipeline or workflow execution.
type Run struct {
	ID           int64     `json:"id"`
	Platform     Platform  `json:"platform"`
	SpaceID      string    `json:"space_id"`
	Name         string    `json:"name"`
	WorkflowID   string    `json:"workflow_id"`
	RepoName     string    `json:"repo_name"`
	RepoOwner    string    `json:"repo_owner"`
	HeadBranch   string    `json:"head_branch"`
	HeadSHA      string    `json:"head_sha"`
	Event        string    `json:"event"`
	Status       string    `json:"status"`     // queued, in_progress, completed, running, pending
	Conclusion   string    `json:"conclusion"` // success, failure, cancelled, skipped, timed_out
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	RunStartedAt time.Time `json:"run_started_at"`
	DurationSec  int64     `json:"duration_sec"`
	Actor        string    `json:"actor"`
	ActorAvatar  string    `json:"actor_avatar"`
	CommitMsg    string    `json:"commit_msg"`
	JobsCount    int       `json:"jobs_count,omitempty"`
}

// Job represents a job inside a pipeline run.
type Job struct {
	ID          int64     `json:"id"`
	RunID       int64     `json:"run_id"`
	Name        string    `json:"name"`
	Stage       string    `json:"stage,omitempty"`
	Status      string    `json:"status"`     // queued, in_progress, completed
	Conclusion  string    `json:"conclusion"` // success, failure, cancelled
	DurationSec int64     `json:"duration_sec"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Steps       []Step    `json:"steps"`
	HTMLURL     string    `json:"html_url"`
}

// Step represents an individual step in a workflow job.
type Step struct {
	Number      int       `json:"number"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`     // completed, in_progress, queued
	Conclusion  string    `json:"conclusion"` // success, failure, skipped
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// SpaceOverview summarizes single or multi-space status.
type SpaceOverview struct {
	SpaceID         string       `json:"space_id"`
	SpaceName       string       `json:"space_name"`
	Platform        Platform     `json:"platform"`
	Owner           string       `json:"owner"`
	OwnerAvatar     string       `json:"owner_avatar"`
	TotalRepos      int          `json:"total_repos"`
	ActivePipelines int          `json:"active_pipelines"`
	FailedPipelines int          `json:"failed_pipelines"`
	SuccessRate     float64      `json:"success_rate"`
	LastRefreshed   time.Time    `json:"last_refreshed"`
	Repositories    []Repository `json:"repositories"`
	RecentRuns      []Run        `json:"recent_runs"`
	AvailableSpaces []SpaceInfo  `json:"available_spaces"`
}

// SpaceInfo represents basic metadata for space switcher dropdown.
type SpaceInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Platform  Platform `json:"platform"`
	Namespace string   `json:"namespace"`
}
