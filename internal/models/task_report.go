package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONIntMap adalah map string→int yang disimpan sebagai JSONB
// (dipakai untuk operational_attendance & member_allocations).
type JSONIntMap map[string]int

// GormDataType memberi tahu GORM tipe kolom JSONB.
func (JSONIntMap) GormDataType() string {
	return "jsonb"
}

// Value mengubah map menjadi JSON.
func (m JSONIntMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

// Scan membaca JSONB menjadi map dengan normalisasi nilai.
func (m *JSONIntMap) Scan(value any) error {
	if value == nil {
		*m = JSONIntMap{}
		return nil
	}

	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("json int map: tipe tidak didukung %T", value)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return err
	}

	result := make(JSONIntMap, len(parsed))
	for key, item := range parsed {
		result[key] = coerceCostValue(item)
	}
	*m = result

	return nil
}

// IntList adalah daftar integer yang disimpan sebagai JSONB.
type IntList []int64

// GormDataType memberi tahu GORM tipe kolom JSONB.
func (IntList) GormDataType() string {
	return "jsonb"
}

// Value mengubah daftar menjadi JSON.
func (l IntList) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

// Scan membaca JSONB menjadi daftar integer.
func (l *IntList) Scan(value any) error {
	if value == nil {
		*l = nil
		return nil
	}

	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("int list: tipe tidak didukung %T", value)
	}

	var parsed []any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return err
	}

	result := make(IntList, 0, len(parsed))
	for _, item := range parsed {
		if number := coerceCostValue(item); number > 0 {
			result = append(result, int64(number))
		}
	}
	*l = result

	return nil
}

// StoredDocument adalah metadata satu dokumen tersimpan (MinIO).
type StoredDocument struct {
	Disk         string `json:"disk"`
	Path         string `json:"path"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
}

// StoredDocuments adalah daftar dokumen yang disimpan sebagai JSONB.
type StoredDocuments []StoredDocument

// GormDataType memberi tahu GORM tipe kolom JSONB.
func (StoredDocuments) GormDataType() string {
	return "jsonb"
}

// Value mengubah daftar dokumen menjadi JSON.
func (d StoredDocuments) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

// Scan membaca JSONB menjadi daftar dokumen.
func (d *StoredDocuments) Scan(value any) error {
	if value == nil {
		*d = nil
		return nil
	}

	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("stored documents: tipe tidak didukung %T", value)
	}

	return json.Unmarshal(raw, d)
}

// TaskReport merepresentasikan tabel `task_reports` (skema existing).
type TaskReport struct {
	ID                  int64           `gorm:"primaryKey" json:"id"`
	UUID                string          `gorm:"column:uuid" json:"uuid"`
	TaskID              int64           `gorm:"column:task_id" json:"task_id"`
	UserID              int64           `gorm:"column:user_id" json:"user_id"`
	PeriodKey           *string         `gorm:"column:period_key" json:"period_key"`
	StartedPhoto        *string         `gorm:"column:started_photo" json:"-"`
	StartedDocuments    StoredDocuments `gorm:"column:started_documents;type:jsonb" json:"-"`
	FinishedPhoto       *string         `gorm:"column:finished_photo" json:"-"`
	FinishedDocuments   StoredDocuments `gorm:"column:finished_documents;type:jsonb" json:"-"`
	StartedAt           *time.Time      `gorm:"column:started_at" json:"started_at"`
	FinishedAt          *time.Time      `gorm:"column:finished_at" json:"finished_at"`
	DurationMinutes     *int            `gorm:"column:duration_minutes" json:"duration_minutes"`
	MemberAllocations   JSONIntMap      `gorm:"column:member_allocations;type:jsonb" json:"-"`
	ManagerSelfAssigned bool            `gorm:"column:manager_self_assigned" json:"manager_self_assigned"`
	Status              string          `json:"status"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`

	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName memetakan struct ke tabel existing.
func (TaskReport) TableName() string {
	return "task_reports"
}

// Status task report (mirror TaskReportStatus enum).
const (
	TaskReportPending    = "pending"
	TaskReportInProgress = "in_progress"
	TaskReportCompleted  = "completed"
)

// TaskReportStatusLabel mengembalikan label status laporan.
func TaskReportStatusLabel(status string) string {
	switch status {
	case TaskReportInProgress:
		return "Sedang Dikerjakan"
	case TaskReportCompleted:
		return "Selesai"
	default:
		return "Belum Dimulai"
	}
}

// TaskReportValue merepresentasikan tabel `task_report_values`.
type TaskReportValue struct {
	ID                    int64     `gorm:"primaryKey" json:"id"`
	UUID                  string    `gorm:"column:uuid" json:"uuid"`
	TaskReportID          int64     `gorm:"column:task_report_id" json:"task_report_id"`
	TaskAdditionalFieldID int64     `gorm:"column:task_additional_field_id" json:"task_additional_field_id"`
	Value                 *string   `json:"value"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (TaskReportValue) TableName() string {
	return "task_report_values"
}
