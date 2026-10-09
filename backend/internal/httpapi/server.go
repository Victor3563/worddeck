package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/worddeck/internal/service"
)

const (
	pathHealth = "/healthz"
	pathReady  = "/readyz"
)

const readyCheckTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	Accounts       *service.Accounts
	Library        *service.Library
	Shares         *service.Shares
	Study          *service.Study
	DB             Pinger
	Log            *slog.Logger
	Version        string
	MaxImportBytes int64
	SecureCookies  bool
}

type Server struct {
	accounts       *service.Accounts
	library        *service.Library
	shares         *service.Shares
	study          *service.Study
	db             Pinger
	log            *slog.Logger
	version        string
	maxImportBytes int64
	secureCookies  bool

	draining atomic.Bool // выставляется по SIGTERM, чтобы /readyz отвечал 503 ещё до остановки
	started  time.Time
}

func New(d Deps) *Server {
	return &Server{
		accounts: d.Accounts, library: d.Library, shares: d.Shares, study: d.Study,
		db: d.DB, log: d.Log, version: d.Version,
		maxImportBytes: d.MaxImportBytes, secureCookies: d.SecureCookies,
		started: time.Now(),
	}
}

func (s *Server) SetDraining() { s.draining.Store(true) }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	a := s.requireAuth

	mux.HandleFunc("GET "+pathHealth, s.handleHealth)
	mux.HandleFunc("GET "+pathReady, s.handleReady)

	mux.HandleFunc("GET /api/limits", s.handleLimits)

	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("POST /api/auth/logout-all", a(s.handleLogoutAll))
	mux.HandleFunc("GET /api/me", a(s.handleMe))
	mux.HandleFunc("PATCH /api/me", a(s.handleUpdateMe))
	mux.HandleFunc("PUT /api/me/password", a(s.handleChangePassword))
	mux.HandleFunc("DELETE /api/me", a(s.handleDeleteMe))

	mux.HandleFunc("GET /api/workspaces", a(s.handleListWorkspaces))
	mux.HandleFunc("POST /api/workspaces", a(s.handleCreateWorkspace))
	mux.HandleFunc("GET /api/workspaces/{id}", a(s.handleGetWorkspace))
	mux.HandleFunc("PATCH /api/workspaces/{id}", a(s.handleUpdateWorkspace))
	mux.HandleFunc("DELETE /api/workspaces/{id}", a(s.handleDeleteWorkspace))
	mux.HandleFunc("GET /api/workspaces/{id}/cards", a(s.handleListWorkspaceCards))

	mux.HandleFunc("GET /api/workspaces/{id}/folders", a(s.handleListFolders))
	mux.HandleFunc("POST /api/workspaces/{id}/folders", a(s.handleCreateFolder))
	mux.HandleFunc("GET /api/folders/{id}", a(s.handleGetFolder))
	mux.HandleFunc("PATCH /api/folders/{id}", a(s.handleUpdateFolder))
	mux.HandleFunc("DELETE /api/folders/{id}", a(s.handleDeleteFolder))

	mux.HandleFunc("GET /api/folders/{id}/cards", a(s.handleListCards))
	mux.HandleFunc("POST /api/folders/{id}/cards", a(s.handleCreateCard))
	mux.HandleFunc("POST /api/folders/{id}/cards/bulk", a(s.handleBulkCreateCards))
	mux.HandleFunc("POST /api/folders/{id}/cards/delete", a(s.handleBulkDeleteCards))
	mux.HandleFunc("POST /api/folders/{id}/import", a(s.handleImport))
	mux.HandleFunc("GET /api/folders/{id}/export", a(s.handleExport))
	mux.HandleFunc("GET /api/cards/{id}", a(s.handleGetCard))
	mux.HandleFunc("PATCH /api/cards/{id}", a(s.handleUpdateCard))
	mux.HandleFunc("DELETE /api/cards/{id}", a(s.handleDeleteCard))

	mux.HandleFunc("GET /api/shares", a(s.handleListShares))
	mux.HandleFunc("POST /api/shares", a(s.handleCreateShare))
	mux.HandleFunc("DELETE /api/shares/{token}", a(s.handleDeleteShare))
	mux.HandleFunc("POST /api/shares/{token}/copy", a(s.handleCopyShare))
	mux.HandleFunc("GET /api/public/shares/{token}", s.optionalAuth(s.handlePublicShare))

	mux.HandleFunc("GET /api/study-sessions", a(s.handleListStudySessions))
	mux.HandleFunc("POST /api/study-sessions", a(s.handleCreateStudySession))
	mux.HandleFunc("DELETE /api/study-sessions", a(s.handleClearStudySessions))
	mux.HandleFunc("DELETE /api/study-sessions/{id}", a(s.handleDeleteStudySession))
	mux.HandleFunc("GET /api/stats", a(s.handleStats))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, errorBody(codeNotFound, ""))
	})

	var h http.Handler = mux
	h = securityHeaders(h)
	h = recoverer(s.log)(h)
	h = accessLog(s.log)(h)
	h = requestID(h)
	return h
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: s.version,
		Uptime:  time.Since(s.started).Round(time.Second).String(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.draining.Load() {
		writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "draining"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), readyCheckTimeout)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "db_unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
}
