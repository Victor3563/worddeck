package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/worddeck/internal/auth"
	"github.com/worddeck/internal/domain"
	"github.com/worddeck/internal/store"
)

const (
	defaultWorkspaceName        = "Моя область"
	defaultWorkspaceDescription = "Ваша первая область. Создайте в ней папки со словами."
)

type Session struct {
	Token     string // отдаётся клиенту; в БД хранится только хэш
	ExpiresAt time.Time
}

type Accounts struct {
	store      *store.Store
	log        *slog.Logger
	hashParams auth.Argon2Params
	sessionTTL time.Duration
	// Проверяется при неизвестном email, чтобы время ответа не выдавало,
	// зарегистрирован ли адрес.
	dummyHash string
}

func NewAccounts(st *store.Store, log *slog.Logger, hashParams auth.Argon2Params, sessionTTL time.Duration) (*Accounts, error) {
	dummy, err := auth.HashPassword("dummy password", hashParams)
	if err != nil {
		return nil, err
	}
	return &Accounts{store: st, log: log, hashParams: hashParams, sessionTTL: sessionTTL, dummyHash: dummy}, nil
}

func (a *Accounts) Register(ctx context.Context, email, displayName, password string) (domain.User, Session, error) {
	email, err := domain.Email("email", email)
	if err != nil {
		return domain.User{}, Session{}, err
	}
	if displayName, err = domain.Text("display_name", displayName, true, domain.MaxDisplayNameLen); err != nil {
		return domain.User{}, Session{}, err
	}
	if err := domain.Password("password", password); err != nil {
		return domain.User{}, Session{}, err
	}
	hash, err := auth.HashPassword(password, a.hashParams)
	if err != nil {
		return domain.User{}, Session{}, err
	}

	var (
		u    domain.User
		sess Session
	)
	err = a.store.InTx(ctx, func(tx *store.Store) error {
		var err error
		if u, err = tx.CreateUser(ctx, email, displayName, hash); err != nil {
			if errors.Is(err, store.ErrDuplicate) {
				return domain.Invalid("email", domain.CodeTaken)
			}
			return err
		}
		if _, err = tx.CreateWorkspace(ctx, u.ID, domain.WorkspaceFields{
			Name: defaultWorkspaceName, Description: defaultWorkspaceDescription, Color: domain.DefaultColor,
		}); err != nil {
			return err
		}
		sess, err = a.startSession(ctx, tx, u.ID)
		return err
	})
	if err != nil {
		return domain.User{}, Session{}, err
	}
	a.log.Info("user registered", "user_id", u.ID)
	return u, sess, nil
}

func (a *Accounts) Login(ctx context.Context, email, password string) (domain.User, Session, error) {
	u, err := a.store.GetUserByEmail(ctx, email)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		auth.CheckPassword(password, a.dummyHash) //nolint:errcheck // только для выравнивания времени
		return domain.User{}, Session{}, domain.ErrInvalidCredentials
	case err != nil:
		return domain.User{}, Session{}, err
	}
	if ok, err := auth.CheckPassword(password, u.PasswordHash); err != nil || !ok {
		return domain.User{}, Session{}, domain.ErrInvalidCredentials
	}
	if err := a.store.DeleteExpiredSessions(ctx, u.ID); err != nil {
		return domain.User{}, Session{}, err
	}
	sess, err := a.startSession(ctx, a.store, u.ID)
	return u, sess, err
}

func (a *Accounts) startSession(ctx context.Context, st *store.Store, userID int64) (Session, error) {
	token, hash, err := auth.NewSessionToken()
	if err != nil {
		return Session{}, err
	}
	exp := time.Now().Add(a.sessionTTL)
	if err := st.CreateSession(ctx, hash, userID, exp); err != nil {
		return Session{}, err
	}
	return Session{Token: token, ExpiresAt: exp}, nil
}

func (a *Accounts) Authenticate(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, domain.ErrUnauthenticated
	}
	id, err := a.store.SessionUser(ctx, auth.HashToken(token))
	if errors.Is(err, domain.ErrNotFound) {
		return 0, domain.ErrUnauthenticated
	}
	return id, err
}

func (a *Accounts) Logout(ctx context.Context, token string) error {
	return a.store.DeleteSession(ctx, auth.HashToken(token))
}

func (a *Accounts) LogoutEverywhere(ctx context.Context, userID int64) error {
	return a.store.DeleteUserSessions(ctx, userID)
}

// Сессия удалённого аккаунта считается отсутствующей.
func (a *Accounts) Me(ctx context.Context, userID int64) (domain.User, error) {
	u, err := a.store.GetUserByID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return u, domain.ErrUnauthenticated
	}
	return u, err
}

func (a *Accounts) UpdateDisplayName(ctx context.Context, userID int64, name string) (domain.User, error) {
	name, err := domain.Text("display_name", name, true, domain.MaxDisplayNameLen)
	if err != nil {
		return domain.User{}, err
	}
	return a.store.UpdateUserName(ctx, userID, name)
}

// Остальные сессии завершаются, чтобы украденная сессия не пережила смену пароля.
func (a *Accounts) ChangePassword(ctx context.Context, userID int64, currentToken, current, next string) error {
	if err := a.confirmPassword(ctx, userID, "current_password", current); err != nil {
		return err
	}
	if err := domain.Password("new_password", next); err != nil {
		return err
	}
	hash, err := auth.HashPassword(next, a.hashParams)
	if err != nil {
		return err
	}
	return a.store.InTx(ctx, func(tx *store.Store) error {
		if err := tx.UpdateUserPassword(ctx, userID, hash); err != nil {
			return err
		}
		return tx.DeleteOtherSessions(ctx, userID, auth.HashToken(currentToken))
	})
}

func (a *Accounts) DeleteAccount(ctx context.Context, userID int64, password string) error {
	if err := a.confirmPassword(ctx, userID, "password", password); err != nil {
		return err
	}
	if err := a.store.DeleteUser(ctx, userID); err != nil {
		return err
	}
	a.log.Info("user deleted", "user_id", userID)
	return nil
}

func (a *Accounts) confirmPassword(ctx context.Context, userID int64, field, password string) error {
	u, err := a.Me(ctx, userID)
	if err != nil {
		return err
	}
	if ok, err := auth.CheckPassword(password, u.PasswordHash); err != nil || !ok {
		return domain.Invalid(field, domain.CodeWrongPassword)
	}
	return nil
}
