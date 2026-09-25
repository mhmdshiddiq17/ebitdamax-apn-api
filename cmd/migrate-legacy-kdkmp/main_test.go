package main

import (
	"testing"
	"time"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestFindFileReferences(t *testing.T) {
	managerDocument := `{"path":"manager-sk.pdf"}`
	photo := "task-start.jpg"
	value := "field-file.pdf"
	plan := &migrationPlan{
		Managers:    []models.User{{ManagerSKDocument: &managerDocument}},
		Reports:     []models.TaskReport{{StartedPhoto: &photo, StartedDocuments: models.StoredDocuments{{Path: "start.pdf"}}}},
		Fields:      []models.TaskAdditionalField{{ID: 5, InputType: "file"}},
		Values:      []models.TaskReportValue{{TaskAdditionalFieldID: 5, Value: &value}},
		Attachments: []models.MeetingMinuteAttachment{{ID: 1}},
	}

	got := findFileReferences(plan)
	if got.ManagerSK != 1 || got.TaskReportPhotos != 1 || got.TaskReportDocuments != 1 || got.AdditionalFieldFiles != 1 || got.MeetingAttachments != 1 || got.Total() != 5 {
		t.Fatalf("file references = %#v", got)
	}
}

func TestRemapTaskIDsRejectsOutOfScopeTask(t *testing.T) {
	if _, err := remapTaskIDs(models.IntList{1, 2}, map[int64]int64{1: 101}); err == nil {
		t.Fatal("out-of-scope task must be rejected")
	}
}

func TestMigrationOptionsRequireIsolatedRehearsal(t *testing.T) {
	cases := []struct {
		name     string
		options  migrationOptions
		database string
		wantErr  bool
	}{
		{name: "normal apply", database: "ebitdamax_apn"},
		{name: "rehearsal", options: migrationOptions{Rehearsal: true, WithoutFiles: true, QAEmail: "qa@example.test", QAPassword: "password123"}, database: rehearsalDatabase},
		{name: "missing without files", options: migrationOptions{Rehearsal: true}, database: rehearsalDatabase, wantErr: true},
		{name: "wrong database", options: migrationOptions{Rehearsal: true, WithoutFiles: true, QAEmail: "qa@example.test", QAPassword: "password123"}, database: "ebitdamax_apn", wantErr: true},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.options.validate(test.database); (err != nil) != test.wantErr {
				t.Fatalf("validate() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestStripFileReferencesAndSelectQA(t *testing.T) {
	entryID := int64(10)
	fileValue := "laporan.pdf"
	plan := &migrationPlan{
		Managers: []models.User{
			{ID: 1, SDMKdkmpEntryID: &entryID},
			{ID: 2},
		},
		Fields:       []models.TaskAdditionalField{{ID: 5, InputType: "file"}},
		Reports:      []models.TaskReport{{UserID: 1, StartedPhoto: &fileValue, FinishedPhoto: &fileValue, StartedDocuments: models.StoredDocuments{{Path: "start.pdf"}}, FinishedDocuments: models.StoredDocuments{{Path: "finish.pdf"}}}},
		Values:       []models.TaskReportValue{{TaskAdditionalFieldID: 5, Value: &fileValue}},
		DailyEntries: []models.EbitdamaxKdkmp{{SDMKdkmpEntryID: entryID, ReportDate: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)}},
		Meetings:     []models.MeetingMinute{{CreatedBy: ptrInt64(1)}},
		Attachments:  []models.MeetingMinuteAttachment{{ID: 1}},
	}

	manager, err := rehearsalManager(plan)
	if err != nil || manager.ID != 1 {
		t.Fatalf("rehearsal manager = %#v, %v", manager, err)
	}
	stripFileReferences(plan)
	if plan.Reports[0].StartedPhoto != nil || plan.Reports[0].FinishedPhoto != nil || plan.Reports[0].StartedDocuments != nil || plan.Reports[0].FinishedDocuments != nil || plan.Values[0].Value != nil || len(plan.Attachments) != 0 {
		t.Fatalf("file references were not stripped: %#v", plan)
	}
}

func ptrInt64(value int64) *int64 { return &value }
