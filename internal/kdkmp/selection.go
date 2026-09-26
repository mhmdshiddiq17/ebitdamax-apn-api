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

// entryDateKey membuat key pilihan task per entry & tanggal ("entryID|YYYY-MM-DD").
func entryDateKey(entryID int64, date time.Time) string {
	return fmt.Sprintf("%d|%s", entryID, DateString(date))
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

// SelectableTasksForUser mengambil task opsional aktif yang boleh dipilih manager.
func (s *SelectionService) SelectableTasksForUser(ctx context.Context, user *models.User) ([]models.Task, error) {
	if user.RoleID == nil {
		return []models.Task{}, nil
	}

	var tasks []models.Task
	err := s.db.WithContext(ctx).
		Model(&models.Task{}).
		Where("tasks.is_active = ? AND tasks.is_mandatory = ?", true, false).
		Where("EXISTS (SELECT 1 FROM task_roles tr WHERE tr.task_id = tasks.id AND tr.role_id = ?)", *user.RoleID).
		Order("CASE WHEN tasks.sort_order IS NULL THEN 1 ELSE 0 END").
		Order("tasks.sort_order").
		Order("tasks.id").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// ExpandSelectedTaskIDsToBMCBundles memperluas pilihan task ke seluruh task
// opsional dengan status BMC yang sama, mengikuti perilaku aplikasi lama.
func (s *SelectionService) ExpandSelectedTaskIDsToBMCBundles(ctx context.Context, user *models.User, selectedIDs []int64) ([]int64, error) {
	selectable, err := s.SelectableTasksForUser(ctx, user)
	if err != nil {
		return nil, err
	}

	selected := make(map[int64]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		if id > 0 {
			selected[id] = true
		}
	}

	if len(selected) == 0 {
		return []int64{}, nil
	}

	statuses := make(map[string]bool)
	validIDs := make(map[int64]bool, len(selectable))
	for _, task := range selectable {
		validIDs[task.ID] = true
		if selected[task.ID] {
			statuses[task.BMCStatus] = true
		}
	}
	for id := range selected {
		if !validIDs[id] {
			return nil, fmt.Errorf("task pilihan tidak tersedia")
		}
	}

	expanded := make([]int64, 0, len(selectable))
	for _, task := range selectable {
		if statuses[task.BMCStatus] {
			expanded = append(expanded, task.ID)
		}
	}

	return expanded, nil
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
		key := entryDateKey(item.SDMKdkmpEntryID, item.ReportDate)
		set := make(map[int64]bool, len(item.SelectedTaskIDs))
		for _, taskID := range item.SelectedTaskIDs {
			set[taskID] = true
		}
		result[key] = set
	}

	return result, nil
}
