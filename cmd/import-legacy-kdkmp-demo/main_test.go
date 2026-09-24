package main

import (
	"testing"
	"time"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestCalendarDaysBetween(t *testing.T) {
	from := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.September, 24, 0, 0, 0, 0, time.UTC)
	if got := calendarDaysBetween(from, to); got != 35 {
		t.Fatalf("calendarDaysBetween() = %d, want 35", got)
	}
}

func TestLegacyEnvUsesFallbackForBlankValue(t *testing.T) {
	t.Setenv("LEGACY_TEST_EMPTY", "")
	if got := legacyEnv("LEGACY_TEST_EMPTY", "fallback"); got != "fallback" {
		t.Fatalf("legacyEnv() = %q, want fallback", got)
	}
}

func TestLegacyDSNOmitsEmptyPassword(t *testing.T) {
	t.Setenv("LEGACY_DB_HOST", "127.0.0.1")
	t.Setenv("LEGACY_DB_PORT", "5432")
	t.Setenv("LEGACY_DB_NAME", "ebitda")
	t.Setenv("LEGACY_DB_USER", "shiddiq")
	t.Setenv("LEGACY_DB_PASSWORD", "")

	dsn := legacyDSN()
	if dsn != "postgres://shiddiq@127.0.0.1:5432/ebitda?sslmode=disable" {
		t.Fatalf("legacyDSN() = %q", dsn)
	}
}

func TestSelectedTaskIDsAddsCompletedOptionalTasks(t *testing.T) {
	day := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	finished := day.Add(10 * time.Hour)
	daily := models.EbitdamaxKdkmp{ReportDate: day, SelectedTaskIDs: models.IntList{1}}
	reports := []models.TaskReport{{TaskID: 2, FinishedAt: &finished}}
	tasks := map[int64]models.Task{
		1: {ID: 1, IsMandatory: true},
		2: {ID: 2, IsMandatory: false},
	}

	selected, err := selectedTaskIDs(daily, reports, tasks, map[int64]int64{1: 101, 2: 102})
	if err != nil {
		t.Fatalf("selectedTaskIDs() error = %v", err)
	}
	if len(selected) != 2 || selected[0] != 101 || selected[1] != 102 {
		t.Fatalf("selectedTaskIDs() = %v, want [101 102]", selected)
	}
}

func TestRebasedPeriodKey(t *testing.T) {
	finished := time.Date(2026, time.September, 24, 9, 0, 0, 0, time.UTC)
	if got := rebasedPeriodKey("daily", &finished, nil); got == nil || *got != "2026-09-24" {
		t.Fatalf("daily period key = %v, want 2026-09-24", got)
	}
	once := "once"
	if got := rebasedPeriodKey("once", &finished, &once); got == nil || *got != "once" {
		t.Fatalf("once period key = %v, want once", got)
	}
}
