package models

import "time"

const (
	MeetingMinuteItemStatusOpen       = "open"
	MeetingMinuteItemStatusInProgress = "in_progress"
	MeetingMinuteItemStatusCompleted  = "completed"
	MeetingMinuteItemStatusCancelled  = "cancelled"
)

// IsMeetingMinuteItemStatus memastikan status item sesuai kontrak legacy.
func IsMeetingMinuteItemStatus(status string) bool {
	switch status {
	case MeetingMinuteItemStatusOpen,
		MeetingMinuteItemStatusInProgress,
		MeetingMinuteItemStatusCompleted,
		MeetingMinuteItemStatusCancelled:
		return true
	default:
		return false
	}
}

// MeetingMinute merepresentasikan tabel meeting_minutes yang sudah ada.
type MeetingMinute struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	Title       string     `json:"title"`
	MeetingDate time.Time  `gorm:"column:meeting_date;type:date" json:"meeting_date"`
	StartTime   *ClockTime `gorm:"column:start_time" json:"-"`
	EndTime     *ClockTime `gorm:"column:end_time" json:"-"`
	Location    *string    `json:"location"`
	Attendees   *string    `json:"attendees"`
	CreatedBy   *int64     `gorm:"column:created_by" json:"created_by"`
	UpdatedBy   *int64     `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Items       []MeetingMinuteItem       `gorm:"foreignKey:MeetingMinuteID" json:"-"`
	Attachments []MeetingMinuteAttachment `gorm:"foreignKey:MeetingMinuteID" json:"-"`
}

// TableName memetakan struct ke tabel existing.
func (MeetingMinute) TableName() string {
	return "meeting_minutes"
}

// MeetingMinuteItem merepresentasikan butir tindak lanjut pada meeting minute.
type MeetingMinuteItem struct {
	ID              int64      `gorm:"primaryKey" json:"id"`
	MeetingMinuteID int64      `gorm:"column:meeting_minute_id" json:"meeting_minute_id"`
	Subject         string     `json:"subject"`
	Description     *string    `json:"description"`
	Action          *string    `json:"action"`
	Objectives      *string    `json:"objectives"`
	DateStart       *time.Time `gorm:"column:date_start;type:date" json:"date_start"`
	DateFinish      *time.Time `gorm:"column:date_finish;type:date" json:"date_finish"`
	PIC             *string    `gorm:"column:pic" json:"pic"`
	Status          string     `json:"status"`
	Remarks         *string    `json:"remarks"`
	SortOrder       int        `gorm:"column:sort_order" json:"sort_order"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	MeetingMinute   *MeetingMinute                   `gorm:"foreignKey:MeetingMinuteID" json:"-"`
	StatusHistories []MeetingMinuteItemStatusHistory `gorm:"foreignKey:MeetingMinuteItemID" json:"-"`
}

// TableName memetakan struct ke tabel existing.
func (MeetingMinuteItem) TableName() string {
	return "meeting_minute_items"
}

// MeetingMinuteItemStatusHistory merepresentasikan audit perubahan item rapat.
type MeetingMinuteItemStatusHistory struct {
	ID                  int64     `gorm:"primaryKey" json:"id"`
	MeetingMinuteItemID int64     `gorm:"column:meeting_minute_item_id" json:"meeting_minute_item_id"`
	FromStatus          string    `gorm:"column:from_status" json:"from_status"`
	ToStatus            string    `gorm:"column:to_status" json:"to_status"`
	Note                *string   `json:"note"`
	ChangedBy           *int64    `gorm:"column:changed_by" json:"changed_by"`
	ChangedByName       string    `gorm:"column:changed_by_name" json:"changed_by_name"`
	CreatedAt           time.Time `json:"created_at"`

	ChangedByUser *User `gorm:"foreignKey:ChangedBy" json:"-"`
}

// TableName memetakan struct ke tabel existing.
func (MeetingMinuteItemStatusHistory) TableName() string {
	return "meeting_minute_item_status_histories"
}

// MeetingMinuteAttachment merepresentasikan metadata objek MinIO lampiran rapat.
type MeetingMinuteAttachment struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	MeetingMinuteID int64     `gorm:"column:meeting_minute_id" json:"meeting_minute_id"`
	Disk            string    `json:"disk"`
	Path            string    `json:"path"`
	OriginalName    string    `gorm:"column:original_name" json:"original_name"`
	MimeType        *string   `gorm:"column:mime_type" json:"mime_type"`
	Size            int64     `json:"size"`
	UploadedBy      *int64    `gorm:"column:uploaded_by" json:"uploaded_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (MeetingMinuteAttachment) TableName() string {
	return "meeting_minute_attachments"
}
