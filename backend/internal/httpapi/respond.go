package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/worddeck/internal/domain"
)

const maxJSONBody = 1 << 20

const (
	codeBadRequest         = "bad_request"
	codeTooLarge           = "too_large"
	codeNotFound           = "not_found"
	codeUnauthenticated    = "unauthenticated"
	codeInvalidCredentials = "invalid_credentials"
	codeInternal           = "internal"
)

// Для ошибок валидации (статус 422) code — одна из констант domain.Code*.
type apiError struct {
	Code   string           `json:"code"`
	Field  string           `json:"field,omitempty"`
	Params *errorParamsJSON `json:"params,omitempty"`
}

type errorParamsJSON struct {
	Min       int    `json:"min,omitempty"`
	Max       int    `json:"max,omitempty"`
	Found     int    `json:"found,omitempty"`
	Number    int    `json:"number,omitempty"`
	ItemField string `json:"item_field,omitempty"`
	MaxBytes  int64  `json:"max_bytes,omitempty"`
}

func toErrorParams(p domain.ValidationParams) *errorParamsJSON {
	if p == (domain.ValidationParams{}) {
		return nil
	}
	return &errorParamsJSON{Min: p.Min, Max: p.Max, Found: p.Found, Number: p.Number, ItemField: p.ItemField}
}

type errorResponse struct {
	Error apiError `json:"error"`
}

func errorBody(code, field string) errorResponse {
	return errorResponse{Error: apiError{Code: code, Field: field}}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v) //nolint:errcheck // клиент уже отключился
	}
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: apiError{
			Code: ve.Code, Field: ve.Field, Params: toErrorParams(ve.Params),
		}})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorBody(codeNotFound, ""))
	case errors.Is(err, domain.ErrUnauthenticated):
		writeJSON(w, http.StatusUnauthorized, errorBody(codeUnauthenticated, ""))
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeJSON(w, http.StatusUnauthorized, errorBody(codeInvalidCredentials, ""))
	case r.Context().Err() != nil:
		// Клиент отключился — отвечать некому.
	default:
		s.log.Error("request failed", "error", err.Error(), "path", r.URL.Path, "request_id", reqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, errorBody(codeInternal, ""))
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorBody(codeTooLarge, ""))
		} else {
			writeJSON(w, http.StatusBadRequest, errorBody(codeBadRequest, ""))
		}
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusNotFound, errorBody(codeNotFound, ""))
		return 0, false
	}
	return id, true
}
