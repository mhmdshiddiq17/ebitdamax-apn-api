package kdkmp

import (
	"context"
	"time"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
)

// MetricsForUser menghitung metrik task harian dari laporan yang selesai.
func MetricsForUser(ctx context.Context, db *gorm.DB, user *models.User, businessDate time.Time) (Metrics, error) {
	metrics, err := MetricsForUsers(ctx, db, []int64{user.ID}, businessDate)
	if err != nil {
		return Metrics{}, err
	}
	return metrics[user.ID], nil
}

// MetricsForUsers menghitung metrik harian banyak user sekaligus
// (dipakai grid monitoring agar tidak query per KDKMP).
func MetricsForUsers(ctx context.Context, db *gorm.DB, userIDs []int64, businessDate time.Time) (map[int64]Metrics, error) {
	ids := SortTaskIDs(userIDs)
	result := make(map[int64]Metrics, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var users []models.User
	if err := db.WithContext(ctx).Preload("Role").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}

	usersByID := make(map[int64]*models.User, len(users))
	roleIDs := make([]int64, 0, len(users))
	entryIDs := make([]int64, 0, len(users))
	for index := range users {
		user := &users[index]
		usersByID[user.ID] = user
		if user.RoleID != nil {
			roleIDs = append(roleIDs, *user.RoleID)
		}
		if user.IsKdkmpManager() && user.SDMKdkmpEntryID != nil {
			entryIDs = append(entryIDs, *user.SDMKdkmpEntryID)
		}
	}

	tasksByRole, err := tasksByRoleForRoles(ctx, db, SortTaskIDs(roleIDs))
	if err != nil {
		return nil, err
	}

	selectedByEntryDate := map[string]map[int64]bool{}
	if len(entryIDs) > 0 {
		selectedByEntryDate, err = NewSelectionService(db).
			DailySelectedTaskIDsByKdkmpEntryAndDate(ctx, SortTaskIDs(entryIDs), businessDate, businessDate)
		if err != nil {
			return nil, err
		}
	}

	start, end := DayRange(businessDate)
	completedOnceByUser, err := completedOnceTaskIDsByUser(ctx, db, ids, start)
	if err != nil {
		return nil, err
	}
	reportsByUser, err := completedReportsByUser(ctx, db, ids, start, end)
	if err != nil {
		return nil, err
	}
	costByUser, revenueByUser, err := revenueAndCostByUser(ctx, db, ids, start, end)
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		user := usersByID[id]

		var roleTasks []models.Task
		if user != nil && user.RoleID != nil {
			roleTasks = tasksByRole[*user.RoleID]
		}
		if user != nil && user.IsKdkmpManager() && user.SDMKdkmpEntryID != nil {
			key := entryDateKey(*user.SDMKdkmpEntryID, businessDate)
			roleTasks = filterMandatoryOrSelected(roleTasks, selectedByEntryDate[key])
		}

		onceIDs := completedOnceByUser[id]
		expected := make(map[int64]models.Task, len(roleTasks))
		for _, task := range roleTasks {
			if task.Period == "once" && onceIDs[task.ID] {
				continue
			}
			expected[task.ID] = task
		}

		assigned := make([]models.Task, 0, len(expected))
		for _, task := range roleTasks {
			if _, ok := expected[task.ID]; ok {
				assigned = append(assigned, task)
			}
		}

		completed := make([]models.TaskReport, 0, len(expected))
		seen := make(map[int64]bool, len(expected))
		for _, report := range reportsByUser[id] {
			if _, ok := expected[report.TaskID]; !ok || seen[report.TaskID] {
				continue
			}
			seen[report.TaskID] = true
			completed = append(completed, report)
		}

		actualCost := 0.0
		for taskID, value := range costByUser[id] {
			if _, ok := expected[taskID]; ok {
				actualCost += value
			}
		}

		result[id] = computeMetrics(assigned, completed, actualCost, revenueByUser[id])
	}

	return result, nil
}

// computeMetrics merangkum metrik dari task yang diharapkan dan laporan selesai.
func computeMetrics(assigned []models.Task, completed []models.TaskReport, actualCost, actualRevenue float64) Metrics {
	assignedByID := make(map[int64]models.Task, len(assigned))
	for _, task := range assigned {
		assignedByID[task.ID] = task
	}

	totalDuration := 0
	withThreshold := 0
	withinThreshold := 0
	for _, report := range completed {
		if report.DurationMinutes != nil {
			totalDuration += *report.DurationMinutes
		}
		task := assignedByID[report.TaskID]
		if task.LowerThreshold != nil && task.UpperThreshold != nil {
			withThreshold++
			if report.DurationMinutes != nil && *report.DurationMinutes >= *task.LowerThreshold && *report.DurationMinutes <= *task.UpperThreshold {
				withinThreshold++
			}
		}
	}

	return Metrics{
		ActualRevenue:      formatNumber(actualRevenue),
		ActualCost:         formatNumber(actualCost),
		TotalDuration:      formatDuration(totalDuration),
		CompletionRate:     percentage(len(completed), len(assigned)),
		TimeComplianceRate: percentage(withinThreshold, withThreshold),
	}
}

// tasksByRoleForRoles mengambil task aktif per role (satu query untuk banyak role).
func tasksByRoleForRoles(ctx context.Context, db *gorm.DB, roleIDs []int64) (map[int64][]models.Task, error) {
	result := make(map[int64][]models.Task, len(roleIDs))
	if len(roleIDs) == 0 {
		return result, nil
	}

	type roleTaskRow struct {
		TaskID int64 `gorm:"column:task_id"`
		RoleID int64 `gorm:"column:role_id"`
	}

	var pairs []roleTaskRow
	if err := db.WithContext(ctx).
		Table("task_roles").
		Select("task_id, role_id").
		Where("role_id IN ?", roleIDs).
		Scan(&pairs).Error; err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return result, nil
	}

	taskIDs := make([]int64, 0, len(pairs))
	rolesByTask := make(map[int64][]int64, len(pairs))
	for _, pair := range pairs {
		taskIDs = append(taskIDs, pair.TaskID)
		rolesByTask[pair.TaskID] = append(rolesByTask[pair.TaskID], pair.RoleID)
	}

	var tasks []models.Task
	if err := db.WithContext(ctx).
		Model(&models.Task{}).
		Where("is_active = ?", true).
		Where("id IN ?", taskIDs).
		Order("CASE WHEN sort_order IS NULL THEN 1 ELSE 0 END").
		Order("sort_order").
		Order("id").
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	for _, task := range tasks {
		for _, roleID := range rolesByTask[task.ID] {
			result[roleID] = append(result[roleID], task)
		}
	}

	return result, nil
}

// filterMandatoryOrSelected menyisakan task wajib atau yang dipilih hari itu.
func filterMandatoryOrSelected(tasks []models.Task, selected map[int64]bool) []models.Task {
	filtered := make([]models.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.IsMandatory || selected[task.ID] {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func completedOnceTaskIDsByUser(ctx context.Context, db *gorm.DB, userIDs []int64, before time.Time) (map[int64]map[int64]bool, error) {
	type row struct {
		UserID int64 `gorm:"column:user_id"`
		TaskID int64 `gorm:"column:task_id"`
	}

	var rows []row
	if err := db.WithContext(ctx).
		Table("task_reports AS tr").
		Select("tr.user_id, tr.task_id").
		Joins("JOIN tasks t ON t.id = tr.task_id").
		Where("tr.user_id IN ? AND tr.status = ? AND t.period = ? AND tr.finished_at < ?", userIDs, models.TaskReportCompleted, "once", before).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[int64]map[int64]bool)
	for _, row := range rows {
		if result[row.UserID] == nil {
			result[row.UserID] = make(map[int64]bool)
		}
		result[row.UserID][row.TaskID] = true
	}
	return result, nil
}

func completedReportsByUser(ctx context.Context, db *gorm.DB, userIDs []int64, start, end time.Time) (map[int64][]models.TaskReport, error) {
	var reports []models.TaskReport
	if err := db.WithContext(ctx).
		Select("user_id, task_id, duration_minutes").
		Where("user_id IN ? AND status = ? AND finished_at BETWEEN ? AND ?", userIDs, models.TaskReportCompleted, start, end).
		Find(&reports).Error; err != nil {
		return nil, err
	}

	result := make(map[int64][]models.TaskReport)
	for _, report := range reports {
		result[report.UserID] = append(result[report.UserID], report)
	}
	return result, nil
}

// revenueAndCostByUser menjumlahkan nilai pengeluaran per task dan revenue per user.
func revenueAndCostByUser(ctx context.Context, db *gorm.DB, userIDs []int64, start, end time.Time) (map[int64]map[int64]float64, map[int64]float64, error) {
	type row struct {
		UserID    int64
		TaskID    int64
		TaskName  string
		FieldName string
		Value     *string
	}

	var rows []row
	if err := db.WithContext(ctx).
		Table("task_report_values AS trv").
		Select("tr.user_id, tr.task_id, t.name AS task_name, taf.field_name, trv.value").
		Joins("JOIN task_reports tr ON tr.id = trv.task_report_id").
		Joins("JOIN tasks t ON t.id = tr.task_id").
		Joins("JOIN task_additional_fields taf ON taf.id = trv.task_additional_field_id").
		Where("tr.user_id IN ? AND tr.status = ? AND tr.finished_at BETWEEN ? AND ?", userIDs, models.TaskReportCompleted, start, end).
		Where("t.name IN ?", []string{expenseTaskName, revenueTaskName}).
		Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	costByUser := make(map[int64]map[int64]float64)
	revenueByUser := make(map[int64]float64)
	for _, row := range rows {
		value, ok := numericValue(derefString(row.Value))
		if !ok {
			continue
		}
		if row.TaskName == expenseTaskName {
			if costByUser[row.UserID] == nil {
				costByUser[row.UserID] = make(map[int64]float64)
			}
			costByUser[row.UserID][row.TaskID] += value
		}
		if row.TaskName == revenueTaskName && row.FieldName == revenueFieldName {
			revenueByUser[row.UserID] += value
		}
	}
	return costByUser, revenueByUser, nil
}
