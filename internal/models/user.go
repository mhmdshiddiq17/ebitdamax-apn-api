package models

import "time"

// User merepresentasikan tabel `users` (skema existing, tanpa perubahan).
// Hanya kolom yang dipakai scope manager yang dipetakan.
type User struct {
	ID                     int64      `gorm:"primaryKey" json:"id"`
	RoleID                 *int64     `gorm:"column:role_id" json:"role_id"`
	Name                   string     `json:"name"`
	Username               *string    `gorm:"column:username" json:"username"`
	Email                  string     `json:"email"`
	EmailVerifiedAt        *time.Time `gorm:"column:email_verified_at" json:"email_verified_at"`
	Password               string     `gorm:"column:password" json:"-"`
	SDMKdkmpEntryID        *int64     `gorm:"column:sdm_kdkmp_entry_id" json:"sdm_kdkmp_entry_id"`
	HasCompletedOnboarding bool       `gorm:"column:has_completed_onboarding" json:"has_completed_onboarding"`
	TwoFactorSecret        *string    `gorm:"column:two_factor_secret" json:"-"`
	TwoFactorRecoveryCodes *string    `gorm:"column:two_factor_recovery_codes" json:"-"`
	TwoFactorConfirmedAt   *time.Time `gorm:"column:two_factor_confirmed_at" json:"-"`
	ManagerSKDocument      *string    `gorm:"column:manager_sk_document" json:"-"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`

	Role *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`

	RegionalAssignments []UserRegionalAssignment `gorm:"foreignKey:UserID" json:"regional_assignments,omitempty"`
	SDMKdkmpEntry       *SdmKdkmpEntry           `gorm:"foreignKey:SDMKdkmpEntryID" json:"sdm_kdkmp_entry,omitempty"`
}

// TableName memetakan struct ke tabel existing.
func (User) TableName() string {
	return "users"
}

// IsKdkmpManager meniru User::isKdkmpManager() pada aplikasi lama.
func (u *User) IsKdkmpManager() bool {
	return u.Role != nil &&
		u.Role.Domain == RoleDomainKdkmp &&
		u.Role.Slug == RoleSlugManager
}

// IsRegionalManager meniru User::isRegionalManager() pada aplikasi lama.
func (u *User) IsRegionalManager() bool {
	return u.Role != nil &&
		u.Role.Domain == RoleDomainKdkmp &&
		u.Role.Slug == RoleSlugRegionalManager
}

// IsSuperadmin meniru pengecekan level superadmin pada aplikasi lama.
func (u *User) IsSuperadmin() bool {
	return u.Role != nil && u.Role.Level == RoleLevelSuperadmin
}
