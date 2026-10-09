// Использование:
//
//	var pool *sql.DB
//	func TestMain(m *testing.M) { os.Exit(testdb.Run(m, &pool)) }
//	func TestX(t *testing.T) { testdb.Reset(t, pool); ... }
//
// С `go test -short` контейнер не запускается, тесты с БД пропускаются.
package testdb

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/worddeck/internal/db"
)

// Та же версия, что в docker-compose.yml.
const image = "postgres:16-alpine"

func Run(m *testing.M, pool **sql.DB) int {
	flag.Parse()
	if testing.Short() {
		return m.Run()
	}
	ctx := context.Background()
	ctr, err := postgres.Run(ctx, image,
		postgres.WithDatabase("worddeck"), postgres.WithUsername("test"), postgres.WithPassword("test"),
		postgres.BasicWaitStrategies())
	if err != nil {
		fmt.Fprintln(os.Stderr, "testdb: не удалось запустить postgres (Docker запущен? -short пропускает тесты с БД):", err)
		return 1
	}
	defer testcontainers.TerminateContainer(ctr) //nolint:errcheck // по возможности

	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "testdb:", err)
		return 1
	}
	p, err := sql.Open("postgres", url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "testdb:", err)
		return 1
	}
	defer p.Close()
	if err := db.Migrate(ctx, p, db.MigrateUp, slog.New(slog.DiscardHandler)); err != nil {
		fmt.Fprintln(os.Stderr, "testdb: migrate:", err)
		return 1
	}
	*pool = p
	return m.Run()
}

func Reset(t testing.TB, pool *sql.DB) {
	t.Helper()
	if pool == nil {
		t.Skip("needs PostgreSQL: run without -short, with Docker available")
	}
	if _, err := pool.Exec(`TRUNCATE users, user_sessions, workspaces, folders, cards, share_links, study_sessions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
}
