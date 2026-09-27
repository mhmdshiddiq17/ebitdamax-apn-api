package models

import "time"

// KoperasiSarprasStatusPoint merepresentasikan tabel `koperasi_sarpras_status_points`
// (titik peta koperasi dari portal pembangunan, hasil sync berkala).
type KoperasiSarprasStatusPoint struct {
	ID                      int64      `gorm:"primaryKey" json:"id"`
	NIK                     *string    `gorm:"column:nik" json:"nik"`
	NamaKoperasi            *string    `gorm:"column:nama_koperasi" json:"nama_koperasi"`
	Provinsi                *string    `json:"provinsi"`
	KotaKabupaten           *string    `gorm:"column:kota_kabupaten" json:"kota_kabupaten"`
	Kecamatan               *string    `json:"kecamatan"`
	Desa                    *string    `json:"desa"`
	Kodim                   *string    `json:"kodim"`
	Lat                     float64    `json:"lat"`
	Lng                     float64    `json:"lng"`
	ValidationStatus        *string    `gorm:"column:validation_status" json:"validation_status"`
	ProgressPercentage      float64    `gorm:"column:progress_percentage" json:"progress_percentage"`
	Batch                   *string    `json:"batch"`
	CompletedSarprasCount   int        `gorm:"column:completed_sarpras_count" json:"completed_sarpras_count"`
	SarprasLessThan6        bool       `gorm:"column:sarpras_less_than_6" json:"sarpras_less_than_6"`
	SarprasPrimaryLengkap   bool       `gorm:"column:sarpras_primary_lengkap" json:"sarpras_primary_lengkap"`
	SarprasSecondaryLengkap bool       `gorm:"column:sarpras_secondary_lengkap" json:"sarpras_secondary_lengkap"`
	SarprasLengkap          bool       `gorm:"column:sarpras_lengkap" json:"sarpras_lengkap"`
	HasPO                   bool       `gorm:"column:has_po" json:"has_po"`
	HasReceipt              bool       `gorm:"column:has_receipt" json:"has_receipt"`
	HasSales                bool       `gorm:"column:has_sales" json:"has_sales"`
	SyncedAt                *time.Time `gorm:"column:synced_at" json:"synced_at"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// TableName memetakan struct ke tabel existing.
func (KoperasiSarprasStatusPoint) TableName() string {
	return "koperasi_sarpras_status_points"
}
