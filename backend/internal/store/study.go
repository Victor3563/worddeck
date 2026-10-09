package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/worddeck/internal/domain"
)

const studyCols = `id, mode, workspace_id, folder_id, title, total_cards, mistakes, rounds, duration_ms, completed, created_at`

func scanStudySession(sc scanner) (domain.StudySession, error) {
	var (
		s          domain.StudySession
		ws, folder sql.NullInt64
		ms         int64
	)
	err := sc.Scan(&s.ID, &s.Mode, &ws, &folder, &s.Title, &s.TotalCards, &s.Mistakes, &s.Rounds, &ms, &s.Completed, &s.CreatedAt)
	s.WorkspaceID, s.FolderID, s.Duration = nullInt(ws), nullInt(folder), time.Duration(ms)*time.Millisecond
	return s, notFound(err)
}

func (s *Store) CreateStudySession(ctx context.Context, userID int64, in domain.StudySession) (domain.StudySession, error) {
	return scanStudySession(s.q.QueryRowContext(ctx, `
		INSERT INTO study_sessions (user_id, mode, workspace_id, folder_id, title, total_cards, mistakes, rounds, duration_ms, completed)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+studyCols,
		userID, in.Mode, in.WorkspaceID, in.FolderID, in.Title, in.TotalCards, in.Mistakes, in.Rounds,
		in.Duration.Milliseconds(), in.Completed))
}

func (s *Store) ListStudySessions(ctx context.Context, userID int64, limit, offset int) ([]domain.StudySession, int, error) {
	var total int
	if err := s.q.QueryRowContext(ctx, `SELECT count(*) FROM study_sessions WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.q.QueryContext(ctx, `
		SELECT `+studyCols+` FROM study_sessions WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	list, err := collect(rows, err, scanStudySession)
	return list, total, err
}

func (s *Store) DeleteStudySession(ctx context.Context, userID, id int64) error {
	return affected(s.q.ExecContext(ctx, `DELETE FROM study_sessions WHERE id = $1 AND user_id = $2`, id, userID))
}

func (s *Store) ClearStudySessions(ctx context.Context, userID int64) (int64, error) {
	res, err := s.q.ExecContext(ctx, `DELETE FROM study_sessions WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type Totals struct {
	Workspaces, Folders, Cards int
	Sessions                   int
	CardsStudied               int
	Studied                    time.Duration
	CompletedCards             int
	CompletedMistakes          int
	ByMode                     map[domain.StudyMode]int
}

func (s *Store) Totals(ctx context.Context, userID int64) (Totals, error) {
	t := Totals{ByMode: map[domain.StudyMode]int{}}
	var ms int64
	err := s.q.QueryRowContext(ctx, `
		SELECT
			(SELECT count(*) FROM workspaces WHERE owner_id = $1),
			(SELECT count(*) FROM folders f JOIN workspaces w ON w.id = f.workspace_id WHERE w.owner_id = $1),
			(SELECT count(*) FROM cards c JOIN folders f ON f.id = c.folder_id JOIN workspaces w ON w.id = f.workspace_id WHERE w.owner_id = $1),
			count(*), COALESCE(sum(total_cards), 0), COALESCE(sum(duration_ms), 0),
			COALESCE(sum(total_cards) FILTER (WHERE completed), 0),
			COALESCE(sum(mistakes) FILTER (WHERE completed), 0)
		FROM study_sessions WHERE user_id = $1`, userID).
		Scan(&t.Workspaces, &t.Folders, &t.Cards, &t.Sessions, &t.CardsStudied, &ms, &t.CompletedCards, &t.CompletedMistakes)
	if err != nil {
		return t, err
	}
	t.Studied = time.Duration(ms) * time.Millisecond

	rows, err := s.q.QueryContext(ctx, `SELECT mode, count(*) FROM study_sessions WHERE user_id = $1 GROUP BY mode`, userID)
	type modeCount struct {
		mode domain.StudyMode
		n    int
	}
	counts, err := collect(rows, err, func(sc scanner) (modeCount, error) {
		var m modeCount
		return m, sc.Scan(&m.mode, &m.n)
	})
	for _, c := range counts {
		t.ByMode[c.mode] = c.n
	}
	return t, err
}

func (s *Store) DailyActivity(ctx context.Context, userID int64, loc *time.Location, days int) (map[string]domain.DayActivity, error) {
	rows, err := s.q.QueryContext(ctx, `
		SELECT to_char(created_at AT TIME ZONE $2, 'YYYY-MM-DD') AS day, count(*), COALESCE(sum(total_cards), 0)
		FROM study_sessions
		WHERE user_id = $1 AND created_at > now() - make_interval(days => $3)
		GROUP BY day`, userID, loc.String(), days)
	list, err := collect(rows, err, func(sc scanner) (domain.DayActivity, error) {
		var d domain.DayActivity
		return d, sc.Scan(&d.Day, &d.Sessions, &d.Cards)
	})
	out := make(map[string]domain.DayActivity, len(list))
	for _, d := range list {
		out[d.Day] = d
	}
	return out, err
}
