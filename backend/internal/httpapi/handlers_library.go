package httpapi

import (
	"net/http"

	"github.com/worddeck/internal/domain"
)

type workspaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
}

func (q workspaceRequest) patch() domain.WorkspacePatch {
	p := domain.WorkspacePatch{Name: q.Name, Description: q.Description}
	if q.Color != nil {
		c := domain.Color(*q.Color)
		p.Color = &c
	}
	return p
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	list, err := s.library.Workspaces(r.Context(), userID(r))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(list, toWorkspace))
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req workspaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	p := req.patch()
	in := domain.WorkspaceFields{Name: deref(p.Name), Description: deref(p.Description)}
	if p.Color != nil {
		in.Color = *p.Color
	}
	ws, err := s.library.CreateWorkspace(r.Context(), userID(r), in)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toWorkspace(ws))
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	ws, err := s.library.Workspace(r.Context(), userID(r), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkspace(ws))
}

func (s *Server) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req workspaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ws, err := s.library.UpdateWorkspace(r.Context(), userID(r), id, req.patch())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkspace(ws))
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.library.DeleteWorkspace(r.Context(), userID(r), id); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListWorkspaceCards(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	cards, err := s.library.WorkspaceCards(r.Context(), userID(r), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(cards, toCard))
}

type folderRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	WorkspaceID *int64  `json:"workspace_id"`
}

func (s *Server) handleListFolders(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	list, err := s.library.Folders(r.Context(), userID(r), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(list, toFolder))
}

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	wsID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req folderRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	f, err := s.library.CreateFolder(r.Context(), userID(r), wsID,
		domain.FolderFields{Name: deref(req.Name), Description: deref(req.Description)})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toFolder(f))
}

func (s *Server) handleGetFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	f, err := s.library.Folder(r.Context(), userID(r), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFolder(f))
}

func (s *Server) handleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req folderRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	f, err := s.library.UpdateFolder(r.Context(), userID(r), id,
		domain.FolderPatch{Name: req.Name, Description: req.Description, WorkspaceID: req.WorkspaceID})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFolder(f))
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.library.DeleteFolder(r.Context(), userID(r), id); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type cardRequest struct {
	RuWord    *string `json:"ru_word"`
	RuExample *string `json:"ru_example"`
	EnWord    *string `json:"en_word"`
	EnExample *string `json:"en_example"`
	FolderID  *int64  `json:"folder_id"`
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *Server) handleListCards(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	cards, err := s.library.Cards(r.Context(), userID(r), id, r.URL.Query().Get("q"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(cards, toCard))
}

func (s *Server) handleCreateCard(w http.ResponseWriter, r *http.Request) {
	folderID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req cardFieldsJSON
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.library.CreateCard(r.Context(), userID(r), folderID, req.domain())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCard(c))
}

func (s *Server) handleBulkCreateCards(w http.ResponseWriter, r *http.Request) {
	folderID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Cards []cardFieldsJSON `json:"cards"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	in := mapSlice(req.Cards, cardFieldsJSON.domain)
	cards, err := s.library.CreateCards(r.Context(), userID(r), folderID, in)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, mapSlice(cards, toCard))
}

func (s *Server) handleBulkDeleteCards(w http.ResponseWriter, r *http.Request) {
	folderID, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	n, err := s.library.DeleteCards(r.Context(), userID(r), folderID, req.IDs)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, deletedResponse{Deleted: n})
}

func (s *Server) handleGetCard(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := s.library.Card(r.Context(), userID(r), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toCard(c))
}

func (s *Server) handleUpdateCard(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req cardRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.library.UpdateCard(r.Context(), userID(r), id, domain.CardPatch{
		RuWord: req.RuWord, RuExample: req.RuExample, EnWord: req.EnWord, EnExample: req.EnExample, FolderID: req.FolderID,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toCard(c))
}

func (s *Server) handleDeleteCard(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.library.DeleteCard(r.Context(), userID(r), id); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
