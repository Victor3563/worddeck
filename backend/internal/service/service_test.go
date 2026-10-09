package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/worddeck/internal/auth"
	"github.com/worddeck/internal/domain"
	"github.com/worddeck/internal/store"
	"github.com/worddeck/internal/testdb"
)

var pool *sql.DB

func TestMain(m *testing.M) { os.Exit(testdb.Run(m, &pool)) }

// Дешёвые параметры ускоряют тесты; в работе — auth.DefaultArgon2.
var cheapHash = auth.Argon2Params{Memory: 64, Time: 1, Threads: 1}

type env struct {
	ctx      context.Context
	accounts *Accounts
	library  *Library
	shares   *Shares
	study    *Study
}

func setup(t *testing.T) env {
	t.Helper()
	testdb.Reset(t, pool)
	st, log := store.New(pool), slog.New(slog.DiscardHandler)
	acc, err := NewAccounts(st, log, cheapHash, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return env{ctx: context.Background(), accounts: acc, library: NewLibrary(st, log), shares: NewShares(st), study: NewStudy(st)}
}

func (e env) register(t *testing.T, email string) (domain.User, Session) {
	t.Helper()
	u, s, err := e.accounts.Register(e.ctx, email, "Tester", "password123")
	if err != nil {
		t.Fatal(err)
	}
	return u, s
}

func (e env) folder(t *testing.T, userID int64) (domain.Workspace, domain.Folder) {
	t.Helper()
	ws, err := e.library.CreateWorkspace(e.ctx, userID, domain.WorkspaceFields{Name: "WS"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := e.library.CreateFolder(e.ctx, userID, ws.ID, domain.FolderFields{Name: "F"})
	if err != nil {
		t.Fatal(err)
	}
	return ws, f
}

func wantInvalid(t *testing.T, err error, field, code string) {
	t.Helper()
	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != field || ve.Code != code {
		t.Fatalf("got %v, want %s/%s", err, field, code)
	}
}

func wantIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("got %v, want %v", err, target)
	}
}

func TestRegister(t *testing.T) {
	e := setup(t)
	u, sess := e.register(t, "Ann@Example.com")

	if id, err := e.accounts.Authenticate(e.ctx, sess.Token); err != nil || id != u.ID {
		t.Fatalf("new session does not authenticate: %d, %v", id, err)
	}
	ws, err := e.library.Workspaces(e.ctx, u.ID)
	if err != nil || len(ws) != 1 || ws[0].Name != defaultWorkspaceName {
		t.Fatalf("default workspace: %+v, %v", ws, err)
	}

	_, _, err = e.accounts.Register(e.ctx, "ann@example.com", "Ann", "password123")
	wantInvalid(t, err, "email", domain.CodeTaken) // без учёта регистра
	_, _, err = e.accounts.Register(e.ctx, "bob@example.com", "Bob", "short")
	wantInvalid(t, err, "password", domain.CodeTooShort)
	_, _, err = e.accounts.Register(e.ctx, "bob@example.com", "  ", "password123")
	wantInvalid(t, err, "display_name", domain.CodeRequired)
}

func TestLogin(t *testing.T) {
	e := setup(t)
	u, _ := e.register(t, "ann@example.com")

	_, _, err := e.accounts.Login(e.ctx, "ann@example.com", "wrong-password")
	wantIs(t, err, domain.ErrInvalidCredentials)
	_, _, err = e.accounts.Login(e.ctx, "nobody@example.com", "password123")
	wantIs(t, err, domain.ErrInvalidCredentials)

	got, sess, err := e.accounts.Login(e.ctx, "ANN@example.com", "password123")
	if err != nil || got.ID != u.ID {
		t.Fatalf("login: %+v, %v", got, err)
	}
	if _, err := e.accounts.Authenticate(e.ctx, sess.Token); err != nil {
		t.Fatal(err)
	}
	_, err = e.accounts.Authenticate(e.ctx, "forged")
	wantIs(t, err, domain.ErrUnauthenticated)
}

func TestLogout(t *testing.T) {
	e := setup(t)
	u, s1 := e.register(t, "ann@example.com")
	_, s2, _ := e.accounts.Login(e.ctx, "ann@example.com", "password123")
	_, s3, _ := e.accounts.Login(e.ctx, "ann@example.com", "password123")

	if err := e.accounts.Logout(e.ctx, s1.Token); err != nil {
		t.Fatal(err)
	}
	_, err := e.accounts.Authenticate(e.ctx, s1.Token)
	wantIs(t, err, domain.ErrUnauthenticated)
	if _, err := e.accounts.Authenticate(e.ctx, s2.Token); err != nil {
		t.Fatal("logout ended another session")
	}

	if err := e.accounts.LogoutEverywhere(e.ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	for _, s := range []Session{s2, s3} {
		_, err := e.accounts.Authenticate(e.ctx, s.Token)
		wantIs(t, err, domain.ErrUnauthenticated)
	}
}

func TestChangePasswordEndsOtherSessions(t *testing.T) {
	e := setup(t)
	u, current := e.register(t, "ann@example.com")
	_, other, _ := e.accounts.Login(e.ctx, "ann@example.com", "password123")

	err := e.accounts.ChangePassword(e.ctx, u.ID, current.Token, "wrong", "new-password")
	wantInvalid(t, err, "current_password", domain.CodeWrongPassword)
	err = e.accounts.ChangePassword(e.ctx, u.ID, current.Token, "password123", "short")
	wantInvalid(t, err, "new_password", domain.CodeTooShort)

	if err := e.accounts.ChangePassword(e.ctx, u.ID, current.Token, "password123", "new-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.accounts.Authenticate(e.ctx, current.Token); err != nil {
		t.Fatal("current session was ended")
	}
	_, err = e.accounts.Authenticate(e.ctx, other.Token)
	wantIs(t, err, domain.ErrUnauthenticated)
	if _, _, err := e.accounts.Login(e.ctx, "ann@example.com", "new-password"); err != nil {
		t.Fatal("new password does not work")
	}
}

func TestDeleteAccount(t *testing.T) {
	e := setup(t)
	u, sess := e.register(t, "ann@example.com")

	wantInvalid(t, e.accounts.DeleteAccount(e.ctx, u.ID, "wrong"), "password", domain.CodeWrongPassword)
	if err := e.accounts.DeleteAccount(e.ctx, u.ID, "password123"); err != nil {
		t.Fatal(err)
	}
	_, err := e.accounts.Authenticate(e.ctx, sess.Token)
	wantIs(t, err, domain.ErrUnauthenticated)
	_, err = e.accounts.Me(e.ctx, u.ID)
	wantIs(t, err, domain.ErrUnauthenticated)
}

func TestOwnershipIsolation(t *testing.T) {
	e := setup(t)
	ann, _ := e.register(t, "ann@example.com")
	bob, _ := e.register(t, "bob@example.com")
	annWS, annFolder := e.folder(t, ann.ID)
	_, bobFolder := e.folder(t, bob.ID)
	card, err := e.library.CreateCard(e.ctx, ann.ID, annFolder.ID, domain.CardFields{RuWord: "дом", EnWord: "house"})
	if err != nil {
		t.Fatal(err)
	}
	bobCard, _ := e.library.CreateCard(e.ctx, bob.ID, bobFolder.ID, domain.CardFields{RuWord: "кот", EnWord: "cat"})

	_, err = e.library.Workspace(e.ctx, bob.ID, annWS.ID)
	wantIs(t, err, domain.ErrNotFound)
	_, err = e.library.Cards(e.ctx, bob.ID, annFolder.ID, "")
	wantIs(t, err, domain.ErrNotFound)
	_, err = e.library.Card(e.ctx, bob.ID, card.ID)
	wantIs(t, err, domain.ErrNotFound)
	_, err = e.library.CreateFolder(e.ctx, bob.ID, annWS.ID, domain.FolderFields{Name: "intruder"})
	wantIs(t, err, domain.ErrNotFound)
	wantIs(t, e.library.DeleteCard(e.ctx, bob.ID, card.ID), domain.ErrNotFound)
	// Свою карточку нельзя перенести и в чужую папку.
	_, err = e.library.UpdateCard(e.ctx, bob.ID, bobCard.ID, domain.CardPatch{FolderID: &annFolder.ID})
	wantIs(t, err, domain.ErrNotFound)
}

func TestWorkspaceValidation(t *testing.T) {
	e := setup(t)
	u, _ := e.register(t, "ann@example.com")
	ws, err := e.library.CreateWorkspace(e.ctx, u.ID, domain.WorkspaceFields{Name: "  Words  "})
	if err != nil || ws.Name != "Words" || ws.Color != domain.DefaultColor {
		t.Fatalf("create: %+v, %v", ws, err)
	}
	bad := domain.Color("black")
	_, err = e.library.UpdateWorkspace(e.ctx, u.ID, ws.ID, domain.WorkspacePatch{Color: &bad})
	wantInvalid(t, err, "color", domain.CodeInvalid)
	empty := " "
	_, err = e.library.UpdateWorkspace(e.ctx, u.ID, ws.ID, domain.WorkspacePatch{Name: &empty})
	wantInvalid(t, err, "name", domain.CodeRequired)
}

func TestCreateCards(t *testing.T) {
	e := setup(t)
	u, _ := e.register(t, "ann@example.com")
	_, f := e.folder(t, u.ID)

	in := []domain.CardFields{{RuWord: "один", EnWord: "one"}, {RuWord: "два", EnWord: "two"}, {RuWord: "три", EnWord: "three"}}
	cards, err := e.library.CreateCards(e.ctx, u.ID, f.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cards {
		if c.EnWord != in[i].EnWord {
			t.Fatalf("order not kept: %d is %q", i, c.EnWord)
		}
	}

	_, err = e.library.CreateCards(e.ctx, u.ID, f.ID, []domain.CardFields{{RuWord: "ok", EnWord: "ok"}, {RuWord: "нет"}})
	wantInvalid(t, err, "cards", domain.CodeRequired)
	var ve *domain.ValidationError
	errors.As(err, &ve)
	if ve.Params.Number != 2 || ve.Params.ItemField != "en_word" {
		t.Fatalf("params = %+v, want number 2, item_field en_word", ve.Params)
	}
	if list, _ := e.library.Cards(e.ctx, u.ID, f.ID, ""); len(list) != 3 {
		t.Fatalf("a rejected batch must not be partially saved: %d cards", len(list))
	}
	if list, _ := e.library.Cards(e.ctx, u.ID, f.ID, "  TW "); len(list) != 1 || list[0].EnWord != "two" {
		t.Fatalf("search: %+v", list)
	}
}

func TestImport(t *testing.T) {
	e := setup(t)
	u, _ := e.register(t, "ann@example.com")
	_, f := e.folder(t, u.ID)
	data := []byte("дом;house\nодно поле\nкот;cat\n")

	res, err := e.library.Import(e.ctx, u.ID, f.ID, data, true)
	if err != nil || len(res.Cards) != 2 || len(res.Errors) != 1 || res.Created != 0 {
		t.Fatalf("dry run: %+v, %v", res, err)
	}
	if list, _ := e.library.Cards(e.ctx, u.ID, f.ID, ""); len(list) != 0 {
		t.Fatal("dry run saved cards")
	}

	res, err = e.library.Import(e.ctx, u.ID, f.ID, data, false)
	if err != nil || res.Created != 2 {
		t.Fatalf("import: %+v, %v", res, err)
	}
	_, err = e.library.Import(e.ctx, u.ID, f.ID, []byte("одно поле\n"), false)
	wantInvalid(t, err, "file", domain.CodeNoValidCards)
}

func TestSharesAndCopy(t *testing.T) {
	e := setup(t)
	ann, _ := e.register(t, "ann@example.com")
	bob, _ := e.register(t, "bob@example.com")
	annWS, annFolder := e.folder(t, ann.ID)
	bobWS, _ := e.folder(t, bob.ID)
	if _, err := e.library.CreateCards(e.ctx, ann.ID, annFolder.ID, []domain.CardFields{
		{RuWord: "дом", EnWord: "house"}, {RuWord: "кот", EnWord: "cat"},
	}); err != nil {
		t.Fatal(err)
	}

	_, err := e.shares.Create(e.ctx, bob.ID, domain.ResourceFolder, annFolder.ID)
	wantIs(t, err, domain.ErrNotFound) // чужой папкой делиться нельзя
	_, err = e.shares.Create(e.ctx, ann.ID, "user", 1)
	wantInvalid(t, err, "resource_type", domain.CodeInvalid)

	link, err := e.shares.Create(e.ctx, ann.ID, domain.ResourceFolder, annFolder.ID)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := e.shares.Create(e.ctx, ann.ID, domain.ResourceFolder, annFolder.ID)
	if again.Token != link.Token {
		t.Fatal("a resource must have a single link")
	}
	ft := domain.ResourceFolder
	if list, _ := e.shares.List(e.ctx, ann.ID, &ft, annFolder.ID); len(list) != 1 {
		t.Fatalf("filtered list: %+v", list)
	}

	content, err := e.shares.Resolve(e.ctx, link.Token)
	if err != nil || content.Folder == nil || len(content.Folder.Cards) != 2 || content.OwnerName != "Tester" {
		t.Fatalf("resolve: %+v, %v", content, err)
	}

	_, err = e.shares.Copy(e.ctx, bob.ID, link.Token, 0)
	wantInvalid(t, err, "target_id", domain.CodeRequired)
	_, err = e.shares.Copy(e.ctx, bob.ID, link.Token, annWS.ID)
	wantIs(t, err, domain.ErrNotFound) // копировать в чужую область нельзя
	res, err := e.shares.Copy(e.ctx, bob.ID, link.Token, bobWS.ID)
	if err != nil || res.CardsCopied != 2 {
		t.Fatalf("copy folder: %+v, %v", res, err)
	}
	if cards, _ := e.library.Cards(e.ctx, bob.ID, res.ID, ""); len(cards) != 2 {
		t.Fatalf("copied folder has %d cards", len(cards))
	}

	wsLink, _ := e.shares.Create(e.ctx, ann.ID, domain.ResourceWorkspace, annWS.ID)
	res, err = e.shares.Copy(e.ctx, bob.ID, wsLink.Token, 0)
	if err != nil || res.CardsCopied != 2 {
		t.Fatalf("copy workspace: %+v, %v", res, err)
	}

	if err := e.shares.Delete(e.ctx, ann.ID, link.Token); err != nil {
		t.Fatal(err)
	}
	_, err = e.shares.Resolve(e.ctx, link.Token)
	wantIs(t, err, domain.ErrNotFound)
}

func TestStudy(t *testing.T) {
	e := setup(t)
	ann, _ := e.register(t, "ann@example.com")
	bob, _ := e.register(t, "bob@example.com")
	_, annFolder := e.folder(t, ann.ID)
	_, bobFolder := e.folder(t, bob.ID)

	_, err := e.study.Record(e.ctx, ann.ID, domain.StudySession{Mode: "cram"})
	wantInvalid(t, err, "mode", domain.CodeInvalid)
	_, err = e.study.Record(e.ctx, ann.ID, domain.StudySession{Mode: domain.ModeLearn, TotalCards: -1})
	wantInvalid(t, err, "total_cards", domain.CodeInvalid)

	s, err := e.study.Record(e.ctx, ann.ID, domain.StudySession{
		Mode: domain.ModeLearn, FolderID: &bobFolder.ID, TotalCards: 10, Mistakes: 2, Duration: 3 * time.Minute, Completed: true,
	})
	if err != nil || s.FolderID != nil {
		t.Fatalf("a foreign folder must be dropped: %+v, %v", s, err)
	}
	if _, err := e.study.Record(e.ctx, ann.ID, domain.StudySession{
		Mode: domain.ModeMatch, FolderID: &annFolder.ID, TotalCards: 10, Duration: time.Minute, Completed: true,
	}); err != nil {
		t.Fatal(err)
	}

	st, err := e.study.Stats(e.ctx, ann.ID, "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	if st.Sessions != 2 || st.CardsStudied != 20 || st.MinutesStudied != 4 || st.Accuracy != 0.9 ||
		st.StreakDays != 1 || st.ByMode[domain.ModeLearn] != 1 || st.ByMode[domain.ModeDictation] != 0 || len(st.LastDays) != 14 {
		t.Fatalf("stats: %+v", st)
	}

	page, err := e.study.History(e.ctx, ann.ID, 1000, -5)
	if err != nil || page.Limit != DefaultHistoryLimit || page.Offset != 0 || page.Total != 2 || page.Items[0].Mode != domain.ModeMatch {
		t.Fatalf("history: %+v, %v", page, err)
	}
	if n, err := e.study.Clear(e.ctx, ann.ID); err != nil || n != 2 {
		t.Fatalf("clear: %d, %v", n, err)
	}
}
