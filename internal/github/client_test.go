package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAuthenticatedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("expected path /user, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"login": "AndreaPallotta", "name": "Andrea Pallotta"}`))
	}))
	defer server.Close()

	client := NewClient("mock-token")
	client.baseURL = server.URL

	user, err := client.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.Login != "AndreaPallotta" {
		t.Errorf("expected login AndreaPallotta, got %s", user.Login)
	}
}

func TestListRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"id": 1, "name": "civet", "full_name": "AndreaPallotta/civet", "stargazers_count": 5, "default_branch": "main", "fork": false, "archived": false},
			{"id": 2, "name": "forked-repo", "full_name": "AndreaPallotta/forked-repo", "stargazers_count": 0, "default_branch": "main", "fork": true, "archived": false}
		]`))
	}))
	defer server.Close()

	client := NewClient("mock-token")
	client.baseURL = server.URL

	repos, err := client.ListRepositories(context.Background(), "AndreaPallotta", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo (fork filtered out), got %d", len(repos))
	}

	if repos[0].Name != "civet" {
		t.Errorf("expected name civet, got %s", repos[0].Name)
	}
}
