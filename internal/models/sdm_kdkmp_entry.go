package models

// SdmKdkmpEntry merepresentasikan tabel `sdm_kdkmp_entries` (skema existing).
// Hanya kolom yang dipakai modul Users/master data yang dipetakan.
type SdmKdkmpEntry struct {
	ID            int64   `gorm:"primaryKey" json:"id"`
	NIK           *string `gorm:"column:nik" json:"nik"`
	NamaKoperasi  *string `gorm:"column:nama_koperasi" json:"nama_koperasi"`
	Provinsi      *string `json:"provinsi"`
	KotaKabupaten *string `gorm:"column:kota_kabupaten" json:"kota_kabupaten"`
	Kecamatan     *string `json:"kecamatan"`
	Desa          *string `json:"desa"`
}

// TableName memetakan struct ke tabel existing.
func (SdmKdkmpEntry) TableName() string {
	return "sdm_kdkmp_entries"
}
