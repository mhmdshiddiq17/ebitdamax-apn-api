package main

import (
	"testing"

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
