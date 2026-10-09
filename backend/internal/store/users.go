package store

import (
	"context"
	"time"

	"github.com/worddeck/internal/domain"
)

const userCols = `id, email, display_name, password_hash, created_at`

func scanUser(sc scanner) (domain.User, error) {
	var u domain.User
	err := sc.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.CreatedAt)
	return u, notFound(err)
}

func (s *Store) CreateUser(ctx context.Context, email, displayName, passwordHash string) (domain.User, error) {
	u, err := scanUser(s.q.QueryRowContext(ctx, `
		INSERT INTO users (email, display_name, password_hash) VALUES ($1, $2, $3)
		RETURNING `+userCols, email, displayName, passwordHash))
	if isUniqueViolation(err) {
		return u, ErrDuplicate
	}
	return u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(s.q.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE lower(email) = lower($1)`, email))
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(s.q.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (s *Store) UpdateUserName(ctx context.Context, id int64, displayName string) (domain.User, error) {
	return scanUser(s.q.QueryRowContext(ctx, `
		UPDATE users SET display_name = $2, updated_at = now()
		WHERE id = $1 RETURNING `+userCols, id, displayName))
}

func (s *Store) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	return affected(s.q.ExecContext(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash))
}

// Остальные данные пользователя удаляет ON DELETE CASCADE.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	return affected(s.q.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id))
}

func (s *Store) CreateSession(ctx context.Context, tokenHash []byte, userID int64, expiresAt time.Time) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO user_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt)
	return err
}

func (s *Store) SessionUser(ctx context.Context, tokenHash []byte) (int64, error) {
	var id int64
	err := s.q.QueryRowContext(ctx, `SELECT user_id FROM user_sessions WHERE token_hash = $1 AND expires_at > now()`,
		tokenHash).Scan(&id)
	return id, notFound(err)
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID)
	return err
}

func (s *Store) DeleteOtherSessions(ctx context.Context, userID int64, keepHash []byte) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = $1 AND token_hash <> $2`, userID, keepHash)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context, userID int64) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = $1 AND expires_at <= now()`, userID)
	return err
}
