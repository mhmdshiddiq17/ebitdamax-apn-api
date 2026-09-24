package models

import "time"

// Role merepresentasikan tabel `roles` (skema existing, tanpa perubahan).
type Role struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UUID      string    `gorm:"column:uuid" json:"uuid"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Level     string    `json:"level"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (Role) TableName() string {
	return "roles"
}

// Konstanta slug role yang dipakai aplikasi (mirror Role::SLUG_* di Laravel).
const (
	RoleSlugSuperadmin      = "superadmin"
	RoleSlugManager         = "manager"
	RoleSlugRegionalManager = "manager-wilayah"
)

// Konstanta level role (mirror RoleLevel enum).
const (
	RoleLevelStaff      = "staff"
	RoleLevelManager    = "manager"
	RoleLevelSuperadmin = "superadmin"
)

// Konstanta domain role (mirror RoleDomain enum).
const (
	RoleDomainApn   = "apn"
	RoleDomainKdkmp = "kdkmp"
)

// RoleLevelLabel mengembalikan label tampilan level role.
func RoleLevelLabel(level string) string {
	switch level {
	case RoleLevelStaff:
		return "Staff"
	case RoleLevelManager:
		return "Manager"
	case RoleLevelSuperadmin:
		return "Superadmin"
	default:
		return level
	}
}

// RoleLevelOptions mengembalikan opsi level untuk form (mirror RoleLevel::options()).
func RoleLevelOptions() []map[string]string {
	return []map[string]string{
		{"value": RoleLevelStaff, "label": "Staff"},
		{"value": RoleLevelManager, "label": "Manager"},
		{"value": RoleLevelSuperadmin, "label": "Superadmin"},
	}
}
