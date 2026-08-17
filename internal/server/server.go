package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/AndreaPallotta/vantage/internal/config"
	"github.com/AndreaPallotta/vantage/internal/manager"
)

//go:embed web/*
var webFS embed.FS

// Server hosts the Vantage dashboard and API endpoints.
type Server struct {
	cfg     *config.Config
	manager *manager.Manager
	mux     *http.ServeMux
	server  *http.Server
}

// New creates a new Vantage server instance.
func New(cfg *config.Config, mgr *manager.Manager) *Server {
	s := &Server{
		cfg:     cfg,
		manager: mgr,
		mux:     http.NewServeMux(),
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
	subFS, err := fs.Sub(webFS, "web")
	if err == nil {
		s.mux.Handle("/", http.FileServer(http.FS(subFS)))
	}

	s.mux.HandleFunc("/api/spaces", s.handleListSpaces)
	s.mux.HandleFunc("/api/space", s.handleSpaceOverview)
	s.mux.HandleFunc("/api/actions/dispatch", s.handleDispatch)
	s.mux.HandleFunc("/api/actions/rerun", s.handleRerun)
	s.mux.HandleFunc("/api/actions/cancel", s.handleCancel)
	s.mux.HandleFunc("/api/actions/jobs", s.handleGetJobs)
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}

func (s *Server) handleGetJobs(w http.ResponseWriter, r *http.Request) {
	spaceID := r.URL.Query().Get("space_id")
	repo := r.URL.Query().Get("repo")
	runIDStr := r.URL.Query().Get("run_id")

	var runID int64
	fmt.Sscanf(runIDStr, "%d", &runID)

	if spaceID == "" && len(s.cfg.Spaces) > 0 {
		spaceID = s.cfg.Spaces[0].ID
	}

	prov, err := s.manager.GetProvider(spaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobs, err := prov.GetRunJobs(r.Context(), repo, runID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get jobs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jobs)
}

func (s *Server) handleListSpaces(w http.ResponseWriter, r *http.Request) {
	spaces := s.manager.ListSpaces()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(spaces)
}

func (s *Server) handleSpaceOverview(w http.ResponseWriter, r *http.Request) {
	spaceID := r.URL.Query().Get("id")
	if spaceID == "" {
		spaceID = s.cfg.ActiveSpace
	}

	overview, err := s.manager.GetOverview(r.Context(), spaceID, s.cfg.IncludeForks, s.cfg.IncludeArchived)
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
		SpaceID  string                 `json:"space_id"`
		Repo     string                 `json:"repo"`
		Workflow string                 `json:"workflow"`
		Ref      string                 `json:"ref"`
		Inputs   map[string]interface{} `json:"inputs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SpaceID == "" && len(s.cfg.Spaces) > 0 {
		req.SpaceID = s.cfg.Spaces[0].ID
	}

	prov, err := s.manager.GetProvider(req.SpaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := prov.TriggerPipeline(r.Context(), req.Repo, req.Ref, req.Inputs); err != nil {
		http.Error(w, fmt.Sprintf("dispatch error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "pipeline triggered successfully"})
}

func (s *Server) handleRerun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SpaceID string `json:"space_id"`
		Repo    string `json:"repo"`
		RunID   int64  `json:"run_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SpaceID == "" && len(s.cfg.Spaces) > 0 {
		req.SpaceID = s.cfg.Spaces[0].ID
	}

	prov, err := s.manager.GetProvider(req.SpaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := prov.RetryPipeline(r.Context(), req.Repo, req.RunID); err != nil {
		http.Error(w, fmt.Sprintf("rerun error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "pipeline rerun initiated"})
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SpaceID string `json:"space_id"`
		Repo    string `json:"repo"`
		RunID   int64  `json:"run_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SpaceID == "" && len(s.cfg.Spaces) > 0 {
		req.SpaceID = s.cfg.Spaces[0].ID
	}

	prov, err := s.manager.GetProvider(req.SpaceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := prov.CancelPipeline(r.Context(), req.Repo, req.RunID); err != nil {
		http.Error(w, fmt.Sprintf("cancel error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "pipeline cancelled"})
}

// Start runs the HTTP server.
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
