package server

import (
	"testing"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/models"
)

func TestMonitoringPhotosPayload(t *testing.T) {
	startPhoto := "reports/6/start.jpg"
	report := &models.TaskReport{ID: 45, StartedPhoto: &startPhoto}

	photos := monitoringPhotosPayload(report)

	if len(photos) != 1 {
		t.Fatalf("jumlah foto = %d", len(photos))
	}
	if photos[0]["phase"] != "start" || photos[0]["phase_label"] != "Mulai" {
		t.Fatalf("foto = %+v", photos[0])
	}
	if photos[0]["name"] != "Foto Mulai.jpg" {
		t.Fatalf("nama foto = %v", photos[0]["name"])
	}
	if photos[0]["preview_url"] != "/task-reports/45/photos/start/preview" {
		t.Fatalf("preview url = %v", photos[0]["preview_url"])
	}

	empty := monitoringPhotosPayload(&models.TaskReport{ID: 45})
	if len(empty) != 0 {
		t.Fatalf("foto kosong = %+v", empty)
	}
}

func TestMonitoringDocumentsPayload(t *testing.T) {
	report := &models.TaskReport{
		ID: 45,
		StartedDocuments: models.StoredDocuments{
			{Path: "reports/6/doc.pdf", OriginalName: "berita-acara.pdf", MimeType: "application/pdf", Size: 2048},
			{Path: ""},
		},
		FinishedDocuments: models.StoredDocuments{
			{Path: "reports/6/nota.jpg", OriginalName: "nota.jpg", MimeType: "image/jpeg", Size: 512},
		},
	}

	documents := monitoringDocumentsPayload(report)

	if len(documents) != 2 {
		t.Fatalf("jumlah dokumen = %d", len(documents))
	}
	if documents[0]["phase_label"] != "Mulai" || documents[1]["phase_label"] != "Selesai" {
		t.Fatalf("label fase = %v / %v", documents[0]["phase_label"], documents[1]["phase_label"])
	}
	if documents[0]["download_url"] != "/task-reports/45/documents/start/0/download" {
		t.Fatalf("download url = %v", documents[0]["download_url"])
	}
	if documents[1]["name"] != "nota.jpg" || documents[1]["size"] != int64(512) {
		t.Fatalf("dokumen selesai = %+v", documents[1])
	}
}

func TestMonitoringReportValuesPayload(t *testing.T) {
	fileValue := `{"path":"reports/6/lampiran.pdf","original_name":"lampiran.pdf","mime_type":"application/pdf","size":100}`
	brokenValue := `bukan-json`
	numberValue := "200000"

	fields := map[int64]models.TaskAdditionalField{
		1: {ID: 1, Label: "Token Listrik", InputType: "number", ShowWhen: "finish", SortOrder: 2},
		2: {ID: 2, Label: "Catatan Mulai", InputType: "text", ShowWhen: "start", SortOrder: 1},
		3: {ID: 3, Label: "Lampiran", InputType: "file", ShowWhen: "finish", SortOrder: 1},
		4: {ID: 4, Label: "Terlewat", InputType: "text", ShowWhen: "finish", SortOrder: 3},
	}
	values := []models.TaskReportValue{
		{ID: 11, TaskReportID: 45, TaskAdditionalFieldID: 1, Value: &numberValue},
		{ID: 12, TaskReportID: 45, TaskAdditionalFieldID: 3, Value: &fileValue},
		{ID: 13, TaskReportID: 45, TaskAdditionalFieldID: 2, Value: &brokenValue},
		{ID: 14, TaskReportID: 45, TaskAdditionalFieldID: 99, Value: &numberValue},
	}

	payload := monitoringReportValuesPayload(values, fields)

	if len(payload) != 3 {
		t.Fatalf("jumlah nilai = %d (%+v)", len(payload), payload)
	}
	// Urutan mengikuti sortBy([show_when, sort_order, id]) aplikasi lama:
	// "finish" < "start" secara leksikografis.
	if payload[0]["label"] != "Lampiran" {
		t.Fatalf("urutan pertama = %+v", payload[0])
	}
	file, ok := payload[0]["file"].(gin.H)
	if !ok {
		t.Fatalf("file lampiran tidak terdeteksi: %+v", payload[0]["file"])
	}
	if file["name"] != "lampiran.pdf" || file["preview_url"] != "/task-reports/45/additional-fields/12/preview" {
		t.Fatalf("file = %+v", file)
	}
	if payload[1]["label"] != "Token Listrik" {
		t.Fatalf("urutan kedua = %+v", payload[1])
	}
	if payload[2]["label"] != "Catatan Mulai" || payload[2]["phase_label"] != "Mulai Task" {
		t.Fatalf("urutan ketiga = %+v", payload[2])
	}
}

func TestMonitoringValueFilePayloadIgnoresInvalid(t *testing.T) {
	field := models.TaskAdditionalField{ID: 3, InputType: "file", ShowWhen: "finish"}
	broken := "bukan-json"
	if file := monitoringValueFilePayload(field, models.TaskReportValue{Value: &broken}); file != nil {
		t.Fatalf("file tidak valid diterima: %+v", file)
	}
	if file := monitoringValueFilePayload(models.TaskAdditionalField{ID: 3, InputType: "text"}, models.TaskReportValue{Value: &broken}); file != nil {
		t.Fatalf("field non-file menghasilkan file: %+v", file)
	}
}

func TestMonitoringShowWhenLabel(t *testing.T) {
	if monitoringShowWhenLabel("start") != "Mulai Task" || monitoringShowWhenLabel("finish") != "Selesaikan Task" {
		t.Fatalf("label fase salah")
	}
	if monitoringShowWhenLabel("lain") != "lain" {
		t.Fatalf("label fallback salah")
	}
}
