package service

import (
	"testing"
	"time"

	"github.com/worddeck/internal/domain"
)

func TestActivity(t *testing.T) {
	today := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	days := func(ds ...string) map[string]domain.DayActivity {
		m := map[string]domain.DayActivity{}
		for _, d := range ds {
			m[d] = domain.DayActivity{Day: d, Sessions: 1, Cards: 5}
		}
		return m
	}

	chart, streak := activity(days("2026-03-10", "2026-03-09", "2026-03-08", "2026-03-06"), today)
	if streak != 3 {
		t.Errorf("streak ending today = %d, want 3", streak)
	}
	if len(chart) != activityChartDays || chart[0].Day != "2026-02-25" || chart[13].Day != "2026-03-10" || chart[13].Cards != 5 {
		t.Errorf("bad chart: first=%+v last=%+v len=%d", chart[0], chart[len(chart)-1], len(chart))
	}

	// Сегодня ещё не занимались: серия считается до вчерашнего дня.
	if _, streak := activity(days("2026-03-09", "2026-03-08"), today); streak != 2 {
		t.Errorf("streak ending yesterday = %d, want 2", streak)
	}
	if _, streak := activity(days("2026-03-07"), today); streak != 0 {
		t.Errorf("broken streak = %d, want 0", streak)
	}
}
