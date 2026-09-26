package kdkmp

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
)

// MonthlyFinancialMatrix adalah deret biaya harian + kumulatif satu KDKMP
// selama satu periode (dipakai grafik monitoring).
type MonthlyFinancialMatrix struct {
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	HasData   bool                          `json:"has_data"`
	Points    []MonthlyFinancialMatrixPoint `json:"points"`
}

// MonthlyFinancialMatrixPoint adalah satu titik harian pada matrix bulanan.
type MonthlyFinancialMatrixPoint struct {
	Date                 string  `json:"date"`
	PlanCost             float64 `json:"plan_cost"`
	ActualCost           float64 `json:"actual_cost"`
	CumulativePlanCost   float64 `json:"cumulative_plan_cost"`
	CumulativeActualCost float64 `json:"cumulative_actual_cost"`
	PlanRevenue          float64 `json:"plan_revenue"`
	ActualRevenue        float64 `json:"actual_revenue"`
}

// MonthlyFinancialMatrixForEntry membangun titik grafik biaya bulanan untuk
// satu entry KDKMP (manager = user pertama yang terhubung ke entry tersebut).
func MonthlyFinancialMatrixForEntry(ctx context.Context, db *gorm.DB, entryID int64, periodStart, periodEnd time.Time) (MonthlyFinancialMatrix, error) {
	dates := datesBetween(periodStart, periodEnd)
	matrix := MonthlyFinancialMatrix{
		StartDate: DateString(periodStart),
		EndDate:   DateString(periodEnd),
		Points:    monthlyZeroPoints(dates),
	}

	var manager models.User
	err := db.WithContext(ctx).
		Preload("Role").
		Where("sdm_kdkmp_entry_id = ?", entryID).
		Order("users.id").
		First(&manager).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return matrix, nil
	}
	if err != nil {
		return matrix, err
	}

	var roleTasks []models.Task
	if manager.RoleID != nil {
		tasksByRole, err := tasksByRoleForRoles(ctx, db, []int64{*manager.RoleID})
		if err != nil {
			return matrix, err
		}
		roleTasks = tasksByRole[*manager.RoleID]
	}

	recordsByDate, err := monthlyRecordsByDate(ctx, db, entryID, periodStart, periodEnd)
	if err != nil {
		return matrix, err
	}
	reportsByDate, err := monthlyReportsByDate(ctx, db, manager.ID, periodStart, periodEnd)
	if err != nil {
		return matrix, err
	}

	matrix.HasData = true
	cumulativePlan := 0.0
	cumulativeActual := 0.0
	for index, date := range dates {
		dateKey := DateString(date)
		record := recordsByDate[dateKey]

		var selected models.IntList
		if record != nil {
			selected = record.SelectedTaskIDs
		}
		tasks := monthlyExecutionTasks(roleTasks, selected)

		taskIDs := make(map[int64]bool, len(tasks))
		for _, task := range tasks {
			taskIDs[task.ID] = true
		}
		totalActualDuration := 0
		for _, report := range reportsByDate[dateKey] {
			if taskIDs[report.TaskID] && report.DurationMinutes != nil {
				totalActualDuration += *report.DurationMinutes
			}
		}

		variableCost := 0.0
		for _, task := range tasks {
			variableCost += float64(models.CostTotal(task.VariableCost))
		}

		overage := 0.0
		planRevenue := 0.0
		actualRevenue := 0.0
		if record != nil {
			overage = numericOrZero(record.ActualVariableCost)
			planRevenue = numericOrZero(record.PlanRevenue)
			actualRevenue = numericOrZero(record.ActualRevenue)
		}

		planCost, actualCost := monthlyCosts(monthlyFixedCost(tasks), variableCost, overage, totalActualDuration)
		cumulativePlan += planCost
		cumulativeActual += actualCost

		matrix.Points[index] = MonthlyFinancialMatrixPoint{
			Date:                 dateKey,
			PlanCost:             roundMoney(planCost),
			ActualCost:           roundMoney(actualCost),
			CumulativePlanCost:   roundMoney(cumulativePlan),
			CumulativeActualCost: roundMoney(cumulativeActual),
			PlanRevenue:          roundMoney(planRevenue),
			ActualRevenue:        roundMoney(actualRevenue),
		}
	}

	return matrix, nil
}

func monthlyRecordsByDate(ctx context.Context, db *gorm.DB, entryID int64, periodStart, periodEnd time.Time) (map[string]*models.EbitdamaxKdkmp, error) {
	var records []models.EbitdamaxKdkmp
	if err := db.WithContext(ctx).
		Where("sdm_kdkmp_entry_id = ? AND report_date BETWEEN ? AND ?", entryID, DateString(periodStart), DateString(periodEnd)).
		Find(&records).Error; err != nil {
		return nil, err
	}

	result := make(map[string]*models.EbitdamaxKdkmp, len(records))
	for index := range records {
		record := &records[index]
		result[DateString(record.ReportDate)] = record
	}
	return result, nil
}

func monthlyReportsByDate(ctx context.Context, db *gorm.DB, userID int64, periodStart, periodEnd time.Time) (map[string][]models.TaskReport, error) {
	start, _ := DayRange(periodStart)
	_, end := DayRange(periodEnd)

	var reports []models.TaskReport
	if err := db.WithContext(ctx).
		Select("task_id, duration_minutes, finished_at").
		Where("user_id = ? AND status = ? AND finished_at BETWEEN ? AND ?", userID, models.TaskReportCompleted, start, end).
		Find(&reports).Error; err != nil {
		return nil, err
	}

	result := make(map[string][]models.TaskReport)
	for _, report := range reports {
		if report.FinishedAt == nil {
			continue
		}
		key := DateString(report.FinishedAt.In(Location()))
		result[key] = append(result[key], report)
	}
	return result, nil
}

// monthlyExecutionTasks menyisakan task wajib atau yang dipilih pada tanggal itu.
func monthlyExecutionTasks(tasks []models.Task, selected models.IntList) []models.Task {
	selectedSet := make(map[int64]bool, len(selected))
	for _, taskID := range selected {
		selectedSet[taskID] = true
	}

	result := make([]models.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.IsMandatory || selectedSet[task.ID] {
			result = append(result, task)
		}
	}
	return result
}

// monthlyFixedCost memakai total fixed cost task bila semuanya terkonfigurasi,
// selain itu memakai default harian (mirror aplikasi lama).
func monthlyFixedCost(tasks []models.Task) float64 {
	if len(tasks) == 0 {
		return DefaultDailyFixedCost
	}

	total := 0.0
	for _, task := range tasks {
		if task.FixedCost[models.FixedCostConfiguredKey] == 0 {
			return DefaultDailyFixedCost
		}
		total += float64(models.CostTotal(task.FixedCost))
	}
	return total
}

// monthlyCosts menghitung plan cost dan actual cost harian; actual cost hanya
// dihitung bila ada durasi aktual (mirror aplikasi lama).
func monthlyCosts(fixedCost, variableCost, actualVariableOverage float64, totalActualDuration int) (float64, float64) {
	planCost := fixedCost + variableCost
	if totalActualDuration <= 0 {
		return planCost, 0
	}
	return planCost, planCost + actualVariableOverage
}

func monthlyZeroPoints(dates []time.Time) []MonthlyFinancialMatrixPoint {
	points := make([]MonthlyFinancialMatrixPoint, 0, len(dates))
	for _, date := range dates {
		points = append(points, MonthlyFinancialMatrixPoint{Date: DateString(date)})
	}
	return points
}

func datesBetween(periodStart, periodEnd time.Time) []time.Time {
	first := time.Date(periodStart.Year(), periodStart.Month(), periodStart.Day(), 0, 0, 0, 0, Location())
	last := time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), 0, 0, 0, 0, Location())
	if last.Before(first) {
		return nil
	}

	dates := make([]time.Time, 0, int(last.Sub(first).Hours()/24)+1)
	for date := first; !date.After(last); date = date.AddDate(0, 0, 1) {
		dates = append(dates, date)
	}
	return dates
}

func numericOrZero(value *string) float64 {
	parsed, ok := numericValue(derefString(value))
	if !ok {
		return 0
	}
	return parsed
}
