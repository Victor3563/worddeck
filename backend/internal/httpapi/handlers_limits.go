package httpapi

import (
	"net/http"

	"github.com/worddeck/internal/domain"
)

func (s *Server) handleLimits(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, limitsResponse{
		DisplayName:     domain.MaxDisplayNameLen,
		Email:           domain.MaxEmailLen,
		Name:            domain.MaxNameLen,
		Description:     domain.MaxDescriptionLen,
		Word:            domain.MaxWordLen,
		Example:         domain.MaxExampleLen,
		SessionTitle:    domain.MaxSessionTitleLen,
		PasswordMin:     domain.MinPasswordLen,
		PasswordMax:     domain.MaxPasswordLen,
		CardsPerRequest: domain.MaxCardsPerRequest,
		ImportBytes:     s.maxImportBytes,
	})
}
