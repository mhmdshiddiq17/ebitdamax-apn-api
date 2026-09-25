package server

import (
	"testing"
	"time"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestParseMeetingMinutePayload(t *testing.T) {
	startTime := "09:15"
	dueDate := "2026-10-01"
	status := "in_progress"
	pic := "Koordinator Gerai"
	payload := meetingMinutePayload{
		Title:       "Koordinasi pembukaan gerai",
		MeetingDate: "2026-09-25",
		StartTime:   &startTime,
		Items: []meetingMinuteItemPayload{{
			Subject:    "Siapkan dokumen pembukaan",
			DateFinish: &dueDate,
			PIC:        &pic,
			Status:     &status,
		}},
	}

	input, err := parseMeetingMinutePayload(payload, false)
	if err != nil {
		t.Fatalf("parseMeetingMinutePayload() error = %v", err)
	}
	if input.MeetingDate.Format("2006-01-02") != "2026-09-25" || input.StartTime.String() != "09:15" {
		t.Fatalf("tanggal atau jam meeting tidak diparse dengan benar: %#v", input)
	}
	if len(input.Items) != 1 || input.Items[0].Status != "in_progress" || input.Items[0].DateFinish.Format("2006-01-02") != dueDate {
		t.Fatalf("item meeting tidak diparse dengan benar: %#v", input.Items)
	}

	invalidStatus := "selesai"
	payload.Items[0].Status = &invalidStatus
	if _, err := parseMeetingMinutePayload(payload, false); err == nil {
		t.Fatal("status di luar kontrak legacy harus ditolak")
	}
}

func TestActionItemIsOverdue(t *testing.T) {
	dueDate := time.Date(2026, time.September, 24, 0, 0, 0, 0, time.UTC)
	item := &models.MeetingMinuteItem{DateFinish: &dueDate, Status: models.MeetingMinuteItemStatusOpen}

	if !actionItemIsOverdue(item, "2026-09-25") {
		t.Fatal("open item before today must be overdue")
	}
	item.Status = models.MeetingMinuteItemStatusCompleted
	if actionItemIsOverdue(item, "2026-09-25") {
		t.Fatal("completed item must not be overdue")
	}
}
