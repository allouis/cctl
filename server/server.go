package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/allouis/cctl/config"
	"github.com/allouis/cctl/session"
	"github.com/allouis/cctl/web"
)

type Server struct {
	svc *session.Service
	cfg *config.Config
	hub *Hub
	mux *http.ServeMux
}

func New(svc *session.Service, cfg *config.Config, notify <-chan struct{}) *Server {
	s := &Server{
		svc: svc,
		cfg: cfg,
		hub: NewHub(svc, notify),
		mux: http.NewServeMux(),
	}

	s.mux.Handle("/", spaHandler(web.App()))

	// API routes
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/sessions/", s.handleSession)
	s.mux.HandleFunc("/api/resume/", s.handleResume)
	s.mux.HandleFunc("/api/transcript/", s.handleTranscript)
	s.mux.HandleFunc("/api/send/", s.handleSend)
	s.mux.HandleFunc("/api/takeover/", s.handleTakeover)
	s.mux.HandleFunc("/api/projects", s.handleProjects)
	s.mux.HandleFunc("/api/projects/", s.handleProject)
	s.mux.HandleFunc("/api/repos", s.handleRepos)
	s.mux.HandleFunc("/api/events", s.handleEvents)

	return s
}

func (s *Server) ListenAndServe(addr string) error {
	go s.hub.Run()
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessions, err := s.svc.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, sessions)

	case http.MethodPost:
		var req struct {
			Name      string  `json:"name"`
			Dir       string  `json:"dir"`
			Prompt    string  `json:"prompt"`
			Safe      bool    `json:"safe"`
			ParentID  *string `json:"parent_id"`
			ProjectID *string `json:"project_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		dir := req.Dir
		if dir == "" {
			dir = "."
		}
		sessionID, err := s.svc.Create(session.CreateOpts{
			Name:      req.Name,
			Dir:       dir,
			Prompt:    req.Prompt,
			Safe:      req.Safe,
			ParentID:  req.ParentID,
			ProjectID: req.ProjectID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]string{"status": "created", "name": req.Name, "session_id": sessionID})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/api/sessions/"):]
	if name == "" {
		http.Error(w, "missing session name", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		sess, err := s.svc.Get(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if sess == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, sess)

	case http.MethodPatch:
		var req struct {
			ProjectID *string `json:"project_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := s.svc.SetSessionProject(name, req.ProjectID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "updated"})

	case http.MethodDelete:
		if err := s.svc.Kill(name); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Path[len("/api/resume/"):]
	if name == "" {
		http.Error(w, "missing session name", http.StatusBadRequest)
		return
	}

	if err := s.svc.Resume(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "resumed"})
}

func (s *Server) handleTranscript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/transcript/")
	if name == "" {
		http.Error(w, "missing session name", http.StatusBadRequest)
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	entries, err := s.svc.Transcript(name, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, entries)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Path[len("/api/send/"):]
	if name == "" {
		http.Error(w, "missing session name", http.StatusBadRequest)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	result, err := s.svc.Send(name, req.Text)
	if errors.Is(err, session.ErrSessionAttached) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("send failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleTakeover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Path[len("/api/takeover/"):]
	if name == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	if err := s.svc.Takeover(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.hub.ServeSSE(w, r)
}

func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	dirs, err := s.svc.ListRepoDirs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if dirs == nil {
		dirs = []string{}
	}
	writeJSON(w, dirs)
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		projects, err := s.svc.ListProjects()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, projects)

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		p, err := s.svc.CreateProject(req.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, p)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/projects/"):]
	if id == "" {
		http.Error(w, "missing project id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := s.svc.DeleteProject(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Hashed assets are immutable — cache forever.
		if strings.HasPrefix(path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
			return
		}

		// index.html must never be cached so new builds are picked up.
		setNoCache := func() {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		if path == "/" {
			setNoCache()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if the file exists
		f, err := fsys.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for non-file routes
		setNoCache()
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
