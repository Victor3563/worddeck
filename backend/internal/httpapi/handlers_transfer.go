package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/worddeck/internal/cardio"
)

// Запас на заголовки и разделители multipart сверх размера файла.
const multipartOverhead = 64 << 10

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	folderID, ok := pathID(w, r)
	if !ok {
		return
	}
	dryRun, _ := strconv.ParseBool(r.URL.Query().Get("dry_run"))
	data, ok := s.readImport(w, r)
	if !ok {
		return
	}
	res, err := s.library.Import(r.Context(), userID(r), folderID, data, dryRun)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	status := http.StatusCreated
	if dryRun {
		status = http.StatusOK
	}
	writeJSON(w, status, toImport(res))
}

func (s *Server) readImport(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	tooLarge := func() {
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: apiError{
			Code: codeTooLarge, Field: "file", Params: &errorParamsJSON{MaxBytes: s.maxImportBytes},
		}})
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxImportBytes+multipartOverhead)
	if err := r.ParseMultipartForm(s.maxImportBytes); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			tooLarge()
		} else {
			writeJSON(w, http.StatusBadRequest, errorBody(codeBadRequest, "file"))
		}
		return nil, false
	}
	defer r.MultipartForm.RemoveAll() //nolint:errcheck // только временные файлы

	f, _, err := r.FormFile("file")
	if err != nil {
		return []byte(r.FormValue("text")), true
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, s.maxImportBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(codeBadRequest, "file"))
		return nil, false
	}
	if int64(len(data)) > s.maxImportBytes {
		tooLarge()
		return nil, false
	}
	return data, true
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	folderID, ok := pathID(w, r)
	if !ok {
		return
	}
	f, cards, err := s.library.Export(r.Context(), userID(r), folderID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	var buf bytes.Buffer
	var ctype, ext string
	switch r.URL.Query().Get("format") {
	case "json":
		ctype, ext = "application/json; charset=utf-8", "json"
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		err = enc.Encode(struct {
			Cards []cardFieldsJSON `json:"cards"`
		}{mapSlice(cards, toCardFields)})
	default:
		ctype, ext = "text/csv; charset=utf-8", "csv"
		buf.WriteString(cardio.UTF8BOM) // без BOM Excel не распознаёт UTF-8
		err = cardio.WriteCSV(&buf, cards)
	}
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	name := strings.NewReplacer("/", "_", "\\", "_", "\"", "").Replace(f.Name) + "." + ext
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", "attachment; filename=\"cards."+ext+"\"; filename*=UTF-8''"+url.PathEscape(name))
	w.Write(buf.Bytes()) //nolint:errcheck // клиент уже отключился
}
