package models

import "time"

const UserNotifiableType = "App\\Models\\User"

// Notification merepresentasikan notifikasi database untuk satu pengguna.
type Notification struct {
	ID             string     `gorm:"primaryKey;column:id" json:"id"`
	Type           string     `gorm:"column:type" json:"type"`
	NotifiableType string     `gorm:"column:notifiable_type" json:"notifiable_type"`
	NotifiableID   int64      `gorm:"column:notifiable_id" json:"notifiable_id"`
	Data           string     `gorm:"column:data" json:"data"`
	ReadAt         *time.Time `gorm:"column:read_at" json:"read_at"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (Notification) TableName() string { return "notifications" }
