package kdkmp

import (
	"context"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
)

// AccessibleKdkmpEntryIDs mengembalikan ID entry KDKMP yang dapat diakses user.
// all=true berarti seluruh entry (superadmin).
func AccessibleKdkmpEntryIDs(ctx context.Context, db *gorm.DB, user *models.User) ([]int64, bool, error) {
	if user.IsSuperadmin() {
		return nil, true, nil
	}

	assignments, err := loadRegionalAssignments(ctx, db, user.ID)
	if err != nil {
		return nil, false, err
	}

	scopeSQL, args := accessibleScopeConditions(user.SDMKdkmpEntryID, assignments)

	var ids []int64
	err = db.WithContext(ctx).
		Model(&models.SdmKdkmpEntry{}).
		Where("("+scopeSQL+")", args...).
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
