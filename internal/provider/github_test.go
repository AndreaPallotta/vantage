package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AndreaPallotta/vantage/internal/models"
)

func TestGitHubListRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"id": 1, "name": "civet", "full_name": "AndreaPallotta/civet", "stargazers_count": 5, "default_branch": "main", "fork": false, "archived": false, "owner": {"login": "AndreaPallotta"}}
		]`))
	}))
	defer server.Close()

	cfg := models.SpaceConfig{
		ID:        "gh-test",
		Name:      "GitHub Test",
		Platform:  models.PlatformGitHub,
		BaseURL:   server.URL,
		Namespace: "AndreaPallotta",
	}

	p := NewGitHubProvider(cfg, "test-token")
	repos, err := p.ListRepositories(context.Background(), false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	if repos[0].Name != "civet" {
		t.Errorf("expected name civet, got %s", repos[0].Name)
	}
}
