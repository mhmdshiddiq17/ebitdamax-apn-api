package models

import "time"

// SdmKdkmpEntry merepresentasikan tabel `sdm_kdkmp_entries` (skema existing).
// Hanya kolom yang dipakai modul Users/master data & monitoring yang dipetakan.
type SdmKdkmpEntry struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	NIK            *string   `gorm:"column:nik" json:"nik"`
	NamaKoperasi   *string   `gorm:"column:nama_koperasi" json:"nama_koperasi"`
	Provinsi       *string   `json:"provinsi"`
	KotaKabupaten  *string   `gorm:"column:kota_kabupaten" json:"kota_kabupaten"`
	Kecamatan      *string   `json:"kecamatan"`
	Desa           *string   `json:"desa"`
	Batch          *string   `json:"batch"`
	NamaKodim      *string   `gorm:"column:nama_kodim" json:"nama_kodim"`
	JumlahKaryawan int       `gorm:"column:jumlah_karyawan" json:"jumlah_karyawan"`
	Catatan        *string   `json:"catatan"`
	CreatedBy      *int64    `gorm:"column:created_by" json:"created_by"`
	UpdatedBy      *int64    `gorm:"column:updated_by" json:"updated_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	DailyEbitdaRecords []EbitdamaxKdkmp `gorm:"foreignKey:SDMKdkmpEntryID" json:"-"`
}

// TableName memetakan struct ke tabel existing.
func (SdmKdkmpEntry) TableName() string {
	return "sdm_kdkmp_entries"
}
