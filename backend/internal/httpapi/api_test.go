package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/worddeck/internal/auth"
	"github.com/worddeck/internal/service"
	"github.com/worddeck/internal/store"
	"github.com/worddeck/internal/testdb"
)

var pool *sql.DB

func TestMain(m *testing.M) { os.Exit(testdb.Run(m, &pool)) }

func newClient(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	testdb.Reset(t, pool)
	st, log := store.New(pool), slog.New(slog.DiscardHandler)
	acc, err := service.NewAccounts(st, log, auth.Argon2Params{Memory: 64, Time: 1, Threads: 1}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	api := New(Deps{
		Accounts: acc, Library: service.NewLibrary(st, log), Shares: service.NewShares(st), Study: service.NewStudy(st),
		DB: st, Log: log, Version: "test", MaxImportBytes: 1 << 20,
	})
	srv := httptest.NewServer(api.Handler())
	t.Cleanup(srv.Close)
	jar, _ := cookiejar.New(nil)
	return srv, &http.Client{Jar: jar}
}

type result struct {
	status  int
	cookies []*http.Cookie
	body    map[string]any
}

func call(t *testing.T, c *http.Client, srv *httptest.Server, method, path string, body any) result {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, srv.URL+path, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out := result{status: resp.StatusCode, cookies: resp.Cookies()}
	json.NewDecoder(resp.Body).Decode(&out.body) //nolint:errcheck // пустое тело допустимо
	return out
}

func errCode(r result) any {
	e, _ := r.body["error"].(map[string]any)
	return e["code"]
}

func TestSessionCookieFlow(t *testing.T) {
	srv, c := newClient(t)

	res := call(t, c, srv, "POST", "/api/auth/register",
		map[string]string{"email": "ann@example.com", "display_name": "Ann", "password": "password123"})
	if res.status != http.StatusCreated {
		t.Fatalf("register: %+v", res)
	}
	if _, leaked := res.body["token"]; leaked {
		t.Fatal("the session token must not be in the response body")
	}
	var cookie *http.Cookie
	for _, ck := range res.cookies {
		if ck.Name == sessionCookie {
			cookie = ck
		}
	}
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api" || cookie.MaxAge <= 0 {
		t.Fatalf("bad session cookie: %+v", cookie)
	}

	if res := call(t, c, srv, "GET", "/api/me", nil); res.status != http.StatusOK || res.body["email"] != "ann@example.com" {
		t.Fatalf("me: %+v", res)
	}
	if res := call(t, c, srv, "POST", "/api/auth/logout", nil); res.status != http.StatusNoContent {
		t.Fatalf("logout: %+v", res)
	}
	if res := call(t, c, srv, "GET", "/api/me", nil); res.status != http.StatusUnauthorized || errCode(res) != codeUnauthenticated {
		t.Fatalf("me after logout: %+v", res)
	}
}

func TestErrorFormat(t *testing.T) {
	srv, c := newClient(t)

	res := call(t, c, srv, "POST", "/api/auth/register",
		map[string]string{"email": "ann@example.com", "display_name": "Ann", "password": "short"})
	e, _ := res.body["error"].(map[string]any)
	params, _ := e["params"].(map[string]any)
	if res.status != http.StatusUnprocessableEntity || e["code"] != "too_short" || e["field"] != "password" || params["min"] != float64(8) {
		t.Fatalf("validation error: %+v", res)
	}

	res = call(t, c, srv, "POST", "/api/auth/login", map[string]string{"email": "x@example.com", "password": "password123"})
	if res.status != http.StatusUnauthorized || errCode(res) != codeInvalidCredentials {
		t.Fatalf("bad credentials: %+v", res)
	}

	resp, err := c.Post(srv.URL+"/api/auth/login", "application/json", strings.NewReader("{not json"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed JSON: %d", resp.StatusCode)
	}

	if res := call(t, c, srv, "GET", "/api/nope", nil); res.status != http.StatusNotFound || errCode(res) != codeNotFound {
		t.Fatalf("unknown route: %+v", res)
	}
}

func TestLimits(t *testing.T) {
	srv, c := newClient(t)
	res := call(t, c, srv, "GET", "/api/limits", nil)
	if res.status != http.StatusOK || res.body["name"] != float64(100) || res.body["password_min"] != float64(8) ||
		res.body["import_bytes"] != float64(1<<20) {
		t.Fatalf("limits: %+v", res)
	}
}
