package kdkmp

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
)

// RegionFields adalah urutan field wilayah (mirror SdmKdkmpEntry::REGION_FIELDS).
var RegionFields = []string{"provinsi", "kota_kabupaten", "kecamatan", "desa"}

// RegionOption adalah satu kombinasi wilayah unik.
type RegionOption struct {
	Provinsi      string `json:"provinsi"`
	KotaKabupaten string `json:"kota_kabupaten"`
	Kecamatan     string `json:"kecamatan"`
	Desa          string `json:"desa"`
}

// LockedFilters adalah filter wilayah yang terkunci untuk user non-nasional.
type LockedFilters struct {
	Provinsi      *string `json:"provinsi"`
	KotaKabupaten *string `json:"kota_kabupaten"`
	Kecamatan     *string `json:"kecamatan"`
	Desa          *string `json:"desa"`
}

// RegionalAccess adalah konteks akses regional user (mirror RegionalAccessService::filterContext).
type RegionalAccess struct {
	IsNational    bool          `json:"is_national"`
	ScopeLabel    string        `json:"scope_label"`
	LockedFilters LockedFilters `json:"locked_filters"`
}

// RegionalService mengelola akses regional KDKMP.
type RegionalService struct {
	db *gorm.DB
}

// NewRegionalService membuat service akses regional.
func NewRegionalService(db *gorm.DB) *RegionalService {
	return &RegionalService{db: db}
}

// ManagedKdkmpQuery membatasi entry kepada KDKMP yang dikelola manager KDKMP.
func ManagedKdkmpQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&models.SdmKdkmpEntry{}).
		Where(
			`EXISTS (SELECT 1 FROM users mu JOIN roles mr ON mr.id = mu.role_id
			  WHERE mu.sdm_kdkmp_entry_id = sdm_kdkmp_entries.id AND mr.domain = ? AND mr.slug = ?)`,
			models.RoleDomainKdkmp,
			models.RoleSlugManager,
		)
}

// AccessibleManagedKdkmpQuery = managed + cakupan akses user + filter wilayah.
func AccessibleManagedKdkmpQuery(
	ctx context.Context,
	db *gorm.DB,
	user *models.User,
	filters map[string]string,
) (*gorm.DB, error) {
	query := ManagedKdkmpQuery(db.WithContext(ctx))

	if !user.IsSuperadmin() {
		assignments, err := loadRegionalAssignments(ctx, db, user.ID)
		if err != nil {
			return nil, err
		}

		scopeSQL, args := accessibleScopeConditions(user.SDMKdkmpEntryID, assignments)
		query = query.Where("("+scopeSQL+")", args...)
	}

	return applyRegionFilters(query, filters), nil
}

// RegionOptions mengembalikan kombinasi wilayah unik pada cakupan akses user.
func (s *RegionalService) RegionOptions(ctx context.Context, user *models.User) ([]RegionOption, error) {
	query, err := AccessibleManagedKdkmpQuery(ctx, s.db, user, nil)
	if err != nil {
		return nil, err
	}

	return s.regionOptionsForQuery(ctx, query)
}

// AllRegionOptions mengembalikan seluruh kombinasi wilayah (tanpa batas akses).
func (s *RegionalService) AllRegionOptions(ctx context.Context) ([]RegionOption, error) {
	return s.regionOptionsForQuery(ctx, ManagedKdkmpQuery(s.db.WithContext(ctx)))
}

// FilterContext menghasilkan konteks akses regional (nasional/cakupan/locked filters).
func (s *RegionalService) FilterContext(ctx context.Context, user *models.User) (RegionalAccess, error) {
	if user.IsSuperadmin() {
		return RegionalAccess{IsNational: true, ScopeLabel: "Nasional"}, nil
	}

	options, err := s.RegionOptions(ctx, user)
	if err != nil {
		return RegionalAccess{}, err
	}

	var assignmentCount int64
	err = s.db.WithContext(ctx).
		Model(&models.UserRegionalAssignment{}).
		Where("user_id = ?", user.ID).
		Count(&assignmentCount).Error
	if err != nil {
		return RegionalAccess{}, err
	}

	return RegionalAccess{
		IsNational:    false,
		ScopeLabel:    scopeLabel(assignmentCount, user.SDMKdkmpEntryID != nil),
		LockedFilters: lockedFiltersFromOptions(options),
	}, nil
}

func (s *RegionalService) regionOptionsForQuery(ctx context.Context, query *gorm.DB) ([]RegionOption, error) {
	query = query.
		Select("sdm_kdkmp_entries.provinsi, sdm_kdkmp_entries.kota_kabupaten, sdm_kdkmp_entries.kecamatan, sdm_kdkmp_entries.desa").
		Distinct()

	for _, field := range RegionFields {
		query = query.
			Where("sdm_kdkmp_entries." + field + " IS NOT NULL").
			Where("sdm_kdkmp_entries." + field + " <> ''").
			Order("sdm_kdkmp_entries." + field)
	}

	var entries []models.SdmKdkmpEntry
	if err := query.WithContext(ctx).Find(&entries).Error; err != nil {
		return nil, err
	}

	options := make([]RegionOption, 0, len(entries))
	for _, entry := range entries {
		options = append(options, RegionOption{
			Provinsi:      deref(entry.Provinsi),
			KotaKabupaten: deref(entry.KotaKabupaten),
			Kecamatan:     deref(entry.Kecamatan),
			Desa:          deref(entry.Desa),
		})
	}

	return options, nil
}

func loadRegionalAssignments(ctx context.Context, db *gorm.DB, userID int64) ([]models.UserRegionalAssignment, error) {
	var assignments []models.UserRegionalAssignment
	err := db.WithContext(ctx).Where("user_id = ?", userID).Find(&assignments).Error
	return assignments, err
}

// accessibleScopeConditions membangun kondisi akses entry (mirror scopeAccessibleBy).
func accessibleScopeConditions(ownEntryID *int64, assignments []models.UserRegionalAssignment) (string, []any) {
	conditions := []string{"sdm_kdkmp_entries.id = ?"}
	args := []any{int64(-1)}
	if ownEntryID != nil {
		args[0] = *ownEntryID
	}

	for _, assignment := range assignments {
		switch assignment.ScopeLevel {
		case models.RegionalScopeRegency:
			conditions = append(conditions,
				"(sdm_kdkmp_entries.provinsi = ? AND sdm_kdkmp_entries.kota_kabupaten = ?)")
			args = append(args, assignment.Provinsi, deref(assignment.KotaKabupaten))
		case models.RegionalScopeDistrict:
			conditions = append(conditions,
				"(sdm_kdkmp_entries.provinsi = ? AND sdm_kdkmp_entries.kota_kabupaten = ? AND sdm_kdkmp_entries.kecamatan = ?)")
			args = append(args, assignment.Provinsi, deref(assignment.KotaKabupaten), deref(assignment.Kecamatan))
		default:
			conditions = append(conditions, "(sdm_kdkmp_entries.provinsi = ?)")
			args = append(args, assignment.Provinsi)
		}
	}

	return strings.Join(conditions, " OR "), args
}

// regionFilterConditions membangun kondisi exact-match filter wilayah (mirror scopeForRegions).
func regionFilterConditions(filters map[string]string) (string, []any) {
	conditions := make([]string, 0, len(RegionFields))
	args := make([]any, 0, len(RegionFields))

	for _, field := range RegionFields {
		value := strings.TrimSpace(filters[field])
		if value == "" {
			continue
		}
		conditions = append(conditions, "sdm_kdkmp_entries."+field+" = ?")
		args = append(args, value)
	}

	return strings.Join(conditions, " AND "), args
}

func applyRegionFilters(query *gorm.DB, filters map[string]string) *gorm.DB {
	sql, args := regionFilterConditions(filters)
	if sql == "" {
		return query
	}
	return query.Where(sql, args...)
}

// lockedFiltersFromOptions mengunci field wilayah yang hanya punya satu nilai unik.
func lockedFiltersFromOptions(options []RegionOption) LockedFilters {
	lock := func(field string) *string {
		seen := make(map[string]bool)
		unique := make([]string, 0, 1)

		for _, option := range options {
			value := optionField(option, field)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			unique = append(unique, value)
		}

		if len(unique) == 1 {
			return &unique[0]
		}
		return nil
	}

	return LockedFilters{
		Provinsi:      lock("provinsi"),
		KotaKabupaten: lock("kota_kabupaten"),
		Kecamatan:     lock("kecamatan"),
		Desa:          lock("desa"),
	}
}

func optionField(option RegionOption, field string) string {
	switch field {
	case "provinsi":
		return option.Provinsi
	case "kota_kabupaten":
		return option.KotaKabupaten
	case "kecamatan":
		return option.Kecamatan
	case "desa":
		return option.Desa
	default:
		return ""
	}
}

// scopeLabel meniru RegionalAccessService::filterContext untuk label cakupan.
func scopeLabel(assignmentCount int64, hasOwnEntry bool) string {
	if assignmentCount > 1 {
		return fmt.Sprintf("%d cakupan wilayah", assignmentCount)
	}
	if hasOwnEntry {
		return "KDKMP sendiri"
	}
	return "Wilayah penugasan"
}
