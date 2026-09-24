package kdkmp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
)

// SelectionService mengelola pantauan pemilihan task harian manager KDKMP
// (task wajib selalu boleh, task opsional harus dipilih).
type SelectionService struct {
	db *gorm.DB
}

// NewSelectionService membuat service pemilihan task.
func NewSelectionService(db *gorm.DB) *SelectionService {
	return &SelectionService{db: db}
}

// DailySelectedTaskIDsForUser mengambil ID task yang dipilih hari itu.
func (s *SelectionService) DailySelectedTaskIDsForUser(ctx context.Context, user *models.User, businessDate time.Time) ([]int64, error) {
	if user.SDMKdkmpEntryID == nil {
		return nil, nil
	}

	var entry models.EbitdamaxKdkmp
	err := s.db.WithContext(ctx).
		Select("selected_task_ids").
		Where("sdm_kdkmp_entry_id = ? AND report_date = ?", *user.SDMKdkmpEntryID, DateString(businessDate)).
		First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return entry.SelectedTaskIDs, nil
}

// InProgressTaskIDsForUser mengambil ID task opsional yang sedang dikerjakan.
func (s *SelectionService) InProgressTaskIDsForUser(ctx context.Context, user *models.User) ([]int64, error) {
	var ids []int64
	err := s.db.WithContext(ctx).
		Table("task_reports AS tr").
		Joins("JOIN tasks t ON t.id = tr.task_id").
		Where("tr.user_id = ? AND tr.status = ? AND t.is_mandatory = ?", user.ID, models.TaskReportInProgress, false).
		Distinct().
		Pluck("tr.task_id", &ids).Error
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// ExecutionTaskIDsForUser menggabungkan task terpilih dan task yang sedang berjalan.
func (s *SelectionService) ExecutionTaskIDsForUser(ctx context.Context, user *models.User, businessDate time.Time) ([]int64, error) {
	selected, err := s.DailySelectedTaskIDsForUser(ctx, user, businessDate)
	if err != nil {
		return nil, err
	}

	inProgress, err := s.InProgressTaskIDsForUser(ctx, user)
	if err != nil {
		return nil, err
	}

	seen := make(map[int64]bool, len(selected)+len(inProgress))
	merged := make([]int64, 0, len(selected)+len(inProgress))
	for _, id := range append(selected, inProgress...) {
		if !seen[id] {
			seen[id] = true
			merged = append(merged, id)
		}
	}

	return merged, nil
}

// CanStartTask mengecek task boleh dimulai: manager KDKMP hanya boleh
// task wajib atau yang termasuk eksekusi hari ini.
func (s *SelectionService) CanStartTask(ctx context.Context, user *models.User, task *models.Task, businessDate time.Time) (bool, error) {
	if !user.IsKdkmpManager() {
		return true, nil
	}
	if task.IsMandatory {
		return true, nil
	}

	ids, err := s.ExecutionTaskIDsForUser(ctx, user, businessDate)
	if err != nil {
		return false, err
	}

	for _, id := range ids {
		if id == task.ID {
			return true, nil
		}
	}

	return false, nil
}

// DailySelectedTaskIDsByKdkmpEntryAndDate mengambil pilihan task per entry &
// tanggal (dipakai riwayat harian). Hasil: "entryID|YYYY-MM-DD" → set task ID.
func (s *SelectionService) DailySelectedTaskIDsByKdkmpEntryAndDate(
	ctx context.Context,
	entryIDs []int64,
	startDate time.Time,
	endDate time.Time,
) (map[string]map[int64]bool, error) {
	result := make(map[string]map[int64]bool)
	if len(entryIDs) == 0 {
		return result, nil
	}

	type row struct {
		SDMKdkmpEntryID int64     `gorm:"column:sdm_kdkmp_entry_id"`
		ReportDate      time.Time `gorm:"column:report_date"`
		SelectedTaskIDs models.IntList
	}

	var rows []row
	err := s.db.WithContext(ctx).
		Model(&models.EbitdamaxKdkmp{}).
		Select("sdm_kdkmp_entry_id, report_date, selected_task_ids").
		Where("sdm_kdkmp_entry_id IN ?", entryIDs).
		Where("report_date BETWEEN ? AND ?", DateString(startDate), DateString(endDate)).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, item := range rows {
		key := fmt.Sprintf("%d|%s", item.SDMKdkmpEntryID, item.ReportDate.Format("2006-01-02"))
		set := make(map[int64]bool, len(item.SelectedTaskIDs))
		for _, taskID := range item.SelectedTaskIDs {
			set[taskID] = true
		}
		result[key] = set
	}

	return result, nil
}
