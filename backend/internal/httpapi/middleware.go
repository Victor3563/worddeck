package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxRequestID
	ctxLogInfo
)

// Обработчики записывают сюда пользователя, чтобы access log мог его вывести.
type logInfo struct{ userID int64 }

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

const (
	maxRequestIDLen = 128
	requestIDBytes  = 8
)

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > maxRequestIDLen {
			b := make([]byte, requestIDBytes)
			rand.Read(b) // с Go 1.24 не возвращает ошибку
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxRequestID, id)))
	})
}

func accessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			info := &logInfo{}
			next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), ctxLogInfo, info)))
			if rec.status == 0 {
				rec.status = http.StatusOK
			}
			level := slog.LevelInfo
			switch {
			case r.URL.Path == pathHealth || r.URL.Path == pathReady:
				level = slog.LevelDebug
			case rec.status >= 500:
				level = slog.LevelError
			case rec.status >= 400:
				level = slog.LevelWarn
			}
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
				slog.String("request_id", reqID(r.Context())),
			}
			if info.userID != 0 {
				attrs = append(attrs, slog.Int64("user_id", info.userID))
			}
			log.LogAttrs(r.Context(), level, "http request", attrs...)
		})
	}
}

func recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
						panic(v)
					}
					log.Error("panic in handler", "panic", v, "stack", string(debug.Stack()), "request_id", reqID(r.Context()))
					writeJSON(w, http.StatusInternalServerError, errorBody(codeInternal, ""))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func reqID(ctx context.Context) string {
	id, _ := ctx.Value(ctxRequestID).(string)
	return id
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, err := s.accounts.Authenticate(r.Context(), sessionToken(r))
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		next(w, withUser(r, uid))
	}
}

func (s *Server) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if uid, err := s.accounts.Authenticate(r.Context(), sessionToken(r)); err == nil {
			r = withUser(r, uid)
		}
		next(w, r)
	}
}

func withUser(r *http.Request, uid int64) *http.Request {
	if info, ok := r.Context().Value(ctxLogInfo).(*logInfo); ok {
		info.userID = uid
	}
	return r.WithContext(context.WithValue(r.Context(), ctxUserID, uid))
}

func userID(r *http.Request) int64 {
	id, _ := r.Context().Value(ctxUserID).(int64)
	return id
}
