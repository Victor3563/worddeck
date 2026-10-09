package httpapi

import (
	"net/http"

	"github.com/worddeck/internal/domain"
	"github.com/worddeck/internal/service"
)

func (s *Server) startSession(w http.ResponseWriter, status int, u domain.User, sess service.Session) {
	s.setSessionCookie(w, sess.Token, sess.ExpiresAt)
	writeJSON(w, status, sessionResponse{User: toUser(u), ExpiresAt: sess.ExpiresAt})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	u, sess, err := s.accounts.Register(r.Context(), req.Email, req.DisplayName, req.Password)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.startSession(w, http.StatusCreated, u, sess)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	u, sess, err := s.accounts.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	s.startSession(w, http.StatusOK, u, sess)
}

// Успешен и без сессии, чтобы клиент всегда мог сбросить состояние.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if tok := sessionToken(r); tok != "" {
		if err := s.accounts.Logout(r.Context(), tok); err != nil {
			s.writeError(w, r, err)
			return
		}
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if err := s.accounts.LogoutEverywhere(r.Context(), userID(r)); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, err := s.accounts.Me(r.Context(), userID(r))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUser(u))
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DisplayName string `json:"display_name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := s.accounts.UpdateDisplayName(r.Context(), userID(r), req.DisplayName)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUser(u))
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.accounts.ChangePassword(r.Context(), userID(r), sessionToken(r), req.CurrentPassword, req.NewPassword); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.accounts.DeleteAccount(r.Context(), userID(r), req.Password); err != nil {
		s.writeError(w, r, err)
		return
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
