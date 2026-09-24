package models

import "time"

// Passkey merepresentasikan tabel `passkeys` (skema existing, tanpa perubahan).
// `credential` menyimpan JSON webauthn.Credential (format Go; passkey lama
// aplikasi Laravel perlu di-enroll ulang saat migrasi data).
type Passkey struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	UserID       int64      `gorm:"column:user_id" json:"user_id"`
	Name         string     `json:"name"`
	CredentialID string     `gorm:"column:credential_id" json:"credential_id"`
	Credential   string     `gorm:"column:credential" json:"-"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (Passkey) TableName() string {
	return "passkeys"
}
