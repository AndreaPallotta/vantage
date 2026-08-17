package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/github"
)

//go:embed web/*
var webFS embed.FS

// Server hosts the Vantage dashboard and API endpoints.
type Server struct {
	cfg    *config.Config
	client *github.Client
	mux    *http.ServeMux
	server *http.Server
}

// New creates a new Vantage server instance.
func New(cfg *config.Config, client *github.Client) *Server {
	s := &Server{
		cfg:    cfg,
		client: client,
		mux:    http.NewServeMux(),
	}

	s.routes()
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return s
}

func (s *Server) routes() {
	// Serve embedded web frontend
	subFS, err := fs.Sub(webFS, "web")
	if err == nil {
		s.mux.Handle("/", http.FileServer(http.FS(subFS)))
	}

	// API Endpoints
	s.mux.HandleFunc("/api/space", s.handleSpaceOverview)
	s.mux.HandleFunc("/api/actions/dispatch", s.handleDispatch)
	s.mux.HandleFunc("/api/actions/rerun", s.handleRerun)
	s.mux.HandleFunc("/api/actions/cancel", s.handleCancel)
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}

func (s *Server) handleSpaceOverview(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		owner = s.cfg.Space
	}
	if owner == "" {
		// Try to resolve current authenticated user
		if user, err := s.client.GetAuthenticatedUser(r.Context()); err == nil && user.Login != "" {
			owner = user.Login
		} else {
			owner = "AndreaPallotta"
		}
	}

	overview, err := s.client.GetSpaceOverview(r.Context(), owner, s.cfg.IncludeForks, s.cfg.IncludeArchived)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get space overview: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(overview)
}

func (s *Server) handleDispatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Owner    string                 `json:"owner"`
		Repo     string                 `json:"repo"`
		Workflow string                 `json:"workflow"`
		Ref      string                 `json:"ref"`
		Inputs   map[string]interface{} `json:"inputs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Owner == "" || req.Repo == "" || req.Workflow == "" {
		http.Error(w, "owner, repo, and workflow are required", http.StatusBadRequest)
		return
	}

	if err := s.client.DispatchWorkflow(r.Context(), req.Owner, req.Repo, req.Workflow, req.Ref, req.Inputs); err != nil {
		http.Error(w, fmt.Sprintf("dispatch error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "workflow dispatched successfully"})
}

func (s *Server) handleRerun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
		RunID int64  `json:"run_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.client.RerunWorkflow(r.Context(), req.Owner, req.Repo, req.RunID); err != nil {
		http.Error(w, fmt.Sprintf("rerun error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "workflow rerun initiated"})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
		RunID int64  `json:"run_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.client.CancelWorkflowRun(r.Context(), req.Owner, req.Repo, req.RunID); err != nil {
		http.Error(w, fmt.Sprintf("cancel error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "workflow run cancelled"})
}

// Start runs the HTTP server and opens the browser if enabled.
func (s *Server) Start() error {
	url := fmt.Sprintf("http://localhost:%d", s.cfg.Port)
	fmt.Printf("\n🚀 Vantage Mission Control running at: %s\n\n", url)

	if s.cfg.AutoOpen {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(url)
		}()
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
