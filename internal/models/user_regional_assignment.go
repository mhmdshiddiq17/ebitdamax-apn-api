package models

// UserRegionalAssignment merepresentasikan tabel `user_regional_assignments`.
type UserRegionalAssignment struct {
	ID            int64   `gorm:"primaryKey" json:"id"`
	UserID        int64   `gorm:"column:user_id" json:"user_id"`
	ScopeLevel    string  `gorm:"column:scope_level" json:"scope_level"`
	Provinsi      string  `json:"provinsi"`
	KotaKabupaten *string `gorm:"column:kota_kabupaten" json:"kota_kabupaten"`
	Kecamatan     *string `json:"kecamatan"`
}

// TableName memetakan struct ke tabel existing.
func (UserRegionalAssignment) TableName() string {
	return "user_regional_assignments"
}

// Scope level cakupan wilayah (mirror RegionalScopeLevel enum).
const (
	RegionalScopeProvince = "province"
	RegionalScopeRegency  = "regency"
	RegionalScopeDistrict = "district"
)

var validRegionalScopes = map[string]bool{
	RegionalScopeProvince: true,
	RegionalScopeRegency:  true,
	RegionalScopeDistrict: true,
}

// IsValidRegionalScope mengecek nilai scope level.
func IsValidRegionalScope(scope string) bool {
	return validRegionalScopes[scope]
}
