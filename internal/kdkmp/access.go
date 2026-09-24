package kdkmp

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
)

// AccessibleKdkmpEntryIDs mengembalikan ID entry KDKMP yang dapat diakses user.
// all=true berarti seluruh entry (superadmin).
func AccessibleKdkmpEntryIDs(ctx context.Context, db *gorm.DB, user *models.User) ([]int64, bool, error) {
	if user.IsSuperadmin() {
		return nil, true, nil
	}

	conditions := []string{"sdm_kdkmp_entries.id = ?"}
	args := []any{int64(-1)}
	if user.SDMKdkmpEntryID != nil {
		args[0] = *user.SDMKdkmpEntryID
	}

	var assignments []models.UserRegionalAssignment
	if err := db.WithContext(ctx).Where("user_id = ?", user.ID).Find(&assignments).Error; err != nil {
		return nil, false, err
	}

	for _, assignment := range assignments {
		switch assignment.ScopeLevel {
		case models.RegionalScopeRegency:
			conditions = append(conditions, "(sdm_kdkmp_entries.provinsi = ? AND sdm_kdkmp_entries.kota_kabupaten = ?)")
			args = append(args, assignment.Provinsi, deref(assignment.KotaKabupaten))
		case models.RegionalScopeDistrict:
			conditions = append(conditions, "(sdm_kdkmp_entries.provinsi = ? AND sdm_kdkmp_entries.kota_kabupaten = ? AND sdm_kdkmp_entries.kecamatan = ?)")
			args = append(args, assignment.Provinsi, deref(assignment.KotaKabupaten), deref(assignment.Kecamatan))
		default:
			conditions = append(conditions, "(sdm_kdkmp_entries.provinsi = ?)")
			args = append(args, assignment.Provinsi)
		}
	}

	var ids []int64
	err := db.WithContext(ctx).
		Model(&models.SdmKdkmpEntry{}).
		Where(strings.Join(conditions, " OR "), args...).
		Pluck("sdm_kdkmp_entries.id", &ids).Error
	if err != nil {
		return nil, false, err
	}

	return ids, false, nil
}

// CanViewTaskReport meniru TaskReportPolicy::view (scope manager + manager wilayah).
func CanViewTaskReport(ctx context.Context, db *gorm.DB, user *models.User, report *models.TaskReport) (bool, error) {
	if user.IsSuperadmin() || report.UserID == user.ID {
		return true, nil
	}

	if !user.IsRegionalManager() {
		return false, nil
	}

	var entryIDs []int64
	if err := db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", report.UserID).
		Pluck("sdm_kdkmp_entry_id", &entryIDs).Error; err != nil {
		return false, err
	}
	if len(entryIDs) == 0 || entryIDs[0] == 0 {
		return false, nil
	}

	accessibleIDs, all, err := AccessibleKdkmpEntryIDs(ctx, db, user)
	if err != nil {
		return false, err
	}
	if all {
		return true, nil
	}

	for _, id := range accessibleIDs {
		if id == entryIDs[0] {
			return true, nil
		}
	}

	return false, nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
