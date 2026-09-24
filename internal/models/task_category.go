package models

import "time"

// TaskCategory merepresentasikan tabel `task_categories` (skema existing).
type TaskCategory struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	UUID        string    `gorm:"column:uuid" json:"uuid"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (TaskCategory) TableName() string {
	return "task_categories"
}
