package service

import (
	"context"
	"errors"
	"time"

	"github.com/worddeck/internal/domain"
	"github.com/worddeck/internal/store"
)

const (
	DefaultHistoryLimit = 20
	MaxHistoryLimit     = 100
)

// В днях.
const (
	activityChartDays = 14 // столбцы графика активности
	activityWindow    = 60 // глубина истории для графика и серии
)

type Study struct {
	store *store.Store
}

func NewStudy(st *store.Store) *Study { return &Study{store: st} }

// Если тренировка шла по общей ссылке, область и папка чужие — их id не сохраняются.
func (s *Study) Record(ctx context.Context, userID int64, in domain.StudySession) (domain.StudySession, error) {
	if !in.Mode.Valid() {
		return in, domain.Invalid("mode", domain.CodeInvalid)
	}
	if in.TotalCards < 0 || in.Mistakes < 0 || in.Rounds < 0 || in.Duration < 0 ||
		in.TotalCards > domain.MaxSessionCards || in.Mistakes > domain.MaxSessionMistakes || in.Duration > domain.MaxSessionDuration {
		return in, domain.Invalid("total_cards", domain.CodeInvalid)
	}
	var err error
	if in.Title, err = domain.Text("title", in.Title, false, domain.MaxSessionTitleLen); err != nil {
		return in, err
	}
	if in.FolderID != nil {
		if err := s.store.CheckFolderOwner(ctx, userID, *in.FolderID); errors.Is(err, domain.ErrNotFound) {
			in.FolderID = nil
		} else if err != nil {
			return in, err
		}
	}
	if in.WorkspaceID != nil {
		if err := s.store.CheckWorkspaceOwner(ctx, userID, *in.WorkspaceID); errors.Is(err, domain.ErrNotFound) {
			in.WorkspaceID = nil
		} else if err != nil {
			return in, err
		}
	}
	return s.store.CreateStudySession(ctx, userID, in)
}

type HistoryPage struct {
	Items         []domain.StudySession
	Total         int
	Limit, Offset int // фактически применённые
}

func (s *Study) History(ctx context.Context, userID int64, limit, offset int) (HistoryPage, error) {
	p := HistoryPage{Limit: limit, Offset: max(offset, 0)}
	if p.Limit <= 0 || p.Limit > MaxHistoryLimit {
		p.Limit = DefaultHistoryLimit
	}
	var err error
	p.Items, p.Total, err = s.store.ListStudySessions(ctx, userID, p.Limit, p.Offset)
	return p, err
}

func (s *Study) Delete(ctx context.Context, userID, id int64) error {
	return s.store.DeleteStudySession(ctx, userID, id)
}

func (s *Study) Clear(ctx context.Context, userID int64) (int64, error) {
	return s.store.ClearStudySessions(ctx, userID)
}

func (s *Study) Stats(ctx context.Context, userID int64, tz string) (domain.Stats, error) {
	loc := time.UTC
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	t, err := s.store.Totals(ctx, userID)
	if err != nil {
		return domain.Stats{}, err
	}
	days, err := s.store.DailyActivity(ctx, userID, loc, activityWindow)
	if err != nil {
		return domain.Stats{}, err
	}

	st := domain.Stats{
		Workspaces: t.Workspaces, Folders: t.Folders, Cards: t.Cards,
		Sessions: t.Sessions, CardsStudied: t.CardsStudied,
		MinutesStudied: int(t.Studied / time.Minute),
		ByMode:         map[domain.StudyMode]int{},
	}
	for _, m := range domain.StudyModes {
		st.ByMode[m] = t.ByMode[m]
	}
	if t.CompletedCards > 0 {
		st.Accuracy = float64(max(t.CompletedCards-t.CompletedMistakes, 0)) / float64(t.CompletedCards)
	}
	today := time.Now().In(loc)
	st.LastDays, st.StreakDays = activity(days, today)
	return st, nil
}

// Серия — подряд идущие активные дни, заканчивая сегодняшним или, если
// сегодня ещё не занимались, вчерашним.
func activity(byDay map[string]domain.DayActivity, today time.Time) ([]domain.DayActivity, int) {
	day := func(back int) string { return today.AddDate(0, 0, -back).Format(time.DateOnly) }

	chart := make([]domain.DayActivity, 0, activityChartDays)
	for i := activityChartDays - 1; i >= 0; i-- {
		d, ok := byDay[day(i)]
		if !ok {
			d = domain.DayActivity{Day: day(i)}
		}
		chart = append(chart, d)
	}

	streak, start := 0, 0
	if _, ok := byDay[day(0)]; !ok {
		start = 1
	}
	for i := start; i < activityWindow; i++ {
		if _, ok := byDay[day(i)]; !ok {
			break
		}
		streak++
	}
	return chart, streak
}
