package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AndreaPallotta/vantage/internal/models"
)

func TestGitLabListRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "test-gl-token" {
			t.Errorf("expected PRIVATE-TOKEN header test-gl-token, got %s", r.Header.Get("PRIVATE-TOKEN"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"id": 101,
				"name": "core-service",
				"path_with_namespace": "my-group/core-service",
				"description": "Core microservice",
				"web_url": "https://gitlab.com/my-group/core-service",
				"default_branch": "main",
				"star_count": 12,
				"archived": false,
				"namespace": { "name": "my-group", "path": "my-group" }
			}
		]`))
	}))
	defer server.Close()

	cfg := models.SpaceConfig{
		ID:        "gitlab-test",
		Name:      "GitLab Test",
		Platform:  models.PlatformGitLab,
		BaseURL:   server.URL,
		Namespace: "my-group",
	}

	p := NewGitLabProvider(cfg, "test-gl-token")
	repos, err := p.ListRepositories(context.Background(), false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	if repos[0].Name != "core-service" {
		t.Errorf("expected repo name core-service, got %s", repos[0].Name)
	}
	if repos[0].Platform != models.PlatformGitLab {
		t.Errorf("expected platform gitlab, got %s", repos[0].Platform)
	}
}

func TestGitLabListPipelines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"id": 5050,
				"project_id": 101,
				"sha": "abcdef123456",
				"ref": "main",
				"status": "success",
				"source": "push",
				"web_url": "https://gitlab.com/my-group/core-service/-/pipelines/5050",
				"user": { "name": "Andrea Pallotta", "avatar_url": "" }
			}
		]`))
	}))
	defer server.Close()

	cfg := models.SpaceConfig{
		ID:        "gitlab-test",
		Platform:  models.PlatformGitLab,
		BaseURL:   server.URL,
		Namespace: "my-group",
	}

	p := NewGitLabProvider(cfg, "test-gl-token")
	runs, err := p.ListPipelines(context.Background(), "101", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	if runs[0].Conclusion != "success" {
		t.Errorf("expected conclusion success, got %s", runs[0].Conclusion)
	}
}
