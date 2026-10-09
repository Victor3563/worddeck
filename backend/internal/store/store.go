package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/worddeck/internal/domain"
)

var ErrDuplicate = errors.New("duplicate")

const pgUniqueViolation = "23505"

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Store struct {
	db *sql.DB
	q  queryer
}

func New(db *sql.DB) *Store { return &Store{db: db, q: db} }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Вложенный вызов использует уже открытую транзакцию.
func (s *Store) InTx(ctx context.Context, fn func(tx *Store) error) error {
	if _, ok := s.q.(*sql.Tx); ok {
		return fn(s)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(&Store{db: s.db, q: tx}); err != nil {
		tx.Rollback() //nolint:errcheck // важнее исходная ошибка
		return err
	}
	return tx.Commit()
}

func isUniqueViolation(err error) bool {
	var pe *pq.Error
	return errors.As(err, &pe) && pe.Code == pgUniqueViolation
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func affected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type scanner interface{ Scan(dest ...any) error }

func collect[T any](rows *sql.Rows, err error, scan func(scanner) (T, error)) ([]T, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func nullInt(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}
