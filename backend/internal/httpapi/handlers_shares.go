package httpapi

import (
	"net/http"
	"strconv"

	"github.com/worddeck/internal/domain"
)

func (s *Server) handleListShares(w http.ResponseWriter, r *http.Request) {
	var (
		only *domain.ResourceType
		id   int64
	)
	if t := r.URL.Query().Get("type"); t != "" {
		rt := domain.ResourceType(t)
		var err error
		if id, err = strconv.ParseInt(r.URL.Query().Get("id"), 10, 64); err != nil || !rt.Valid() {
			writeJSON(w, http.StatusBadRequest, errorBody(codeBadRequest, "id"))
			return
		}
		only = &rt
	}
	list, err := s.shares.List(r.Context(), userID(r), only, id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(list, toShare))
}

func (s *Server) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string `json:"resource_type"`
		ResourceID   int64  `json:"resource_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	l, err := s.shares.Create(r.Context(), userID(r), domain.ResourceType(req.ResourceType), req.ResourceID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toShare(l))
}

func (s *Server) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	if err := s.shares.Delete(r.Context(), userID(r), r.PathValue("token")); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePublicShare(w http.ResponseWriter, r *http.Request) {
	c, err := s.shares.Resolve(r.Context(), r.PathValue("token"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSharedContent(c, userID(r)))
}

func (s *Server) handleCopyShare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetID int64 `json:"target_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := s.shares.Copy(r.Context(), userID(r), r.PathValue("token"), req.TargetID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, copyResponse{ResourceType: string(res.ResourceType), ID: res.ID, CardsCopied: res.CardsCopied})
}
