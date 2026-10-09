package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/worddeck/internal/domain"
)

func (s *Server) handleListStudySessions(w http.ResponseWriter, r *http.Request) {
	// При ошибке разбора Atoi вернёт 0, и сервис подставит значение по умолчанию.
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	page, err := s.study.History(r.Context(), userID(r), limit, offset)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, historyResponse{
		Items: mapSlice(page.Items, toStudySession), Total: page.Total, Limit: page.Limit, Offset: page.Offset,
	})
}

func (s *Server) handleCreateStudySession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode        string `json:"mode"`
		WorkspaceID *int64 `json:"workspace_id"`
		FolderID    *int64 `json:"folder_id"`
		Title       string `json:"title"`
		TotalCards  int    `json:"total_cards"`
		Mistakes    int    `json:"mistakes"`
		Rounds      int    `json:"rounds"`
		DurationMs  int64  `json:"duration_ms"`
		Completed   *bool  `json:"completed"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	in := domain.StudySession{
		Mode: domain.StudyMode(req.Mode), WorkspaceID: req.WorkspaceID, FolderID: req.FolderID, Title: req.Title,
		TotalCards: req.TotalCards, Mistakes: req.Mistakes, Rounds: req.Rounds,
		Duration: time.Duration(req.DurationMs) * time.Millisecond, Completed: req.Completed == nil || *req.Completed,
	}
	ss, err := s.study.Record(r.Context(), userID(r), in)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toStudySession(ss))
}

func (s *Server) handleDeleteStudySession(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.study.Delete(r.Context(), userID(r), id); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleClearStudySessions(w http.ResponseWriter, r *http.Request) {
	n, err := s.study.Clear(r.Context(), userID(r))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, deletedResponse{Deleted: n})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.study.Stats(r.Context(), userID(r), r.URL.Query().Get("tz"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStats(st))
}
