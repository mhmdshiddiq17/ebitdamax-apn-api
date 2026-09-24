package kdkmp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"agrinaspangan/ebitda-api/internal/models"
)

const (
	TargetRevenue                = 20_000_000
	ActualEbitdaMarginFixedCost  = 9_235_467
	DefaultDailyFixedCost        = 9_235_467
	TaskCompletionWeight         = 55
	TimeComplianceWeight         = 30
	RevenueWeight                = 15
	expenseTaskName              = "Pencatatan Pengeluaran Operasional Harian"
	revenueTaskName              = "Penyetoran Struk dan Uang"
	revenueFieldName             = "rekonsiliasi_uang_masuk"
	tokenListrikFieldName        = "token_listrik"
	bahanBakarKendaraanFieldName = "bahan_bakar_kendaraan"
	tokenListrikThreshold        = 3_000_000
	bahanBakarKendaraanThreshold = 2_000_000
)

var numericValuePattern = regexp.MustCompile(`^[+-]?\d+(?:[.,]\d+)*$`)

// Metrics adalah nilai terhitung untuk dashboard harian KDKMP.
type Metrics struct {
	ActualRevenue      string
	ActualCost         string
	TotalDuration      string
	CompletionRate     float64
	TimeComplianceRate float64
}

// FinancialMatrix adalah ringkasan biaya dan EBITDA per task.
type FinancialMatrix struct {
	FixedCost                  float64                `json:"fixed_cost"`
	TotalVariableCost          float64                `json:"total_variable_cost"`
	TotalActualVariableCost    float64                `json:"total_actual_variable_cost"`
	TotalEstimatedMinutes      int                    `json:"total_estimated_minutes"`
	TotalActualDurationMinutes int                    `json:"total_actual_duration_minutes"`
	TotalPlanCost              float64                `json:"total_plan_cost"`
	TotalActualCost            float64                `json:"total_actual_cost"`
	PlanRevenue                *float64               `json:"plan_revenue"`
	ActualRevenue              *float64               `json:"actual_revenue"`
	PlanEbitda                 *float64               `json:"plan_ebitda"`
	ActualEbitda               *float64               `json:"actual_ebitda"`
	Points                     []FinancialMatrixPoint `json:"points"`
}

// FinancialMatrixPoint adalah satu baris proses pada matrix biaya.
type FinancialMatrixPoint struct {
	Process               int     `json:"process"`
	TaskID                int64   `json:"task_id"`
	TaskName              string  `json:"task_name"`
	EstimatedMinutes      int     `json:"estimated_minutes"`
	ActualDurationMinutes int     `json:"actual_duration_minutes"`
	PlanFixedCost         float64 `json:"plan_fixed_cost"`
	PlanVariableCost      float64 `json:"plan_variable_cost"`
	PlanCost              float64 `json:"plan_cost"`
	ActualFixedCost       float64 `json:"actual_fixed_cost"`
	ActualVariableCost    float64 `json:"actual_variable_cost"`
	ActualCost            float64 `json:"actual_cost"`
	CumulativePlanCost    float64 `json:"cumulative_plan_cost"`
	CumulativeActualCost  float64 `json:"cumulative_actual_cost"`
}

// MetricsForUser menghitung metrik task harian dari laporan yang selesai.
func MetricsForUser(ctx context.Context, db *gorm.DB, user *models.User, businessDate time.Time) (Metrics, error) {
	tasks, err := TasksForUser(ctx, db, user, businessDate, false)
	if err != nil {
		return Metrics{}, err
	}

	start, end := DayRange(businessDate)
	taskIDs := make([]int64, 0, len(tasks))
	taskByID := make(map[int64]models.Task, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
		taskByID[task.ID] = task
	}

	completedOnce := make(map[int64]bool)
	if len(taskIDs) > 0 {
		var rows []struct{ TaskID int64 }
		if err := db.WithContext(ctx).
			Table("task_reports AS tr").
			Select("tr.task_id").
			Joins("JOIN tasks t ON t.id = tr.task_id").
			Where("tr.user_id = ? AND tr.status = ? AND t.period = ? AND tr.finished_at < ?", user.ID, models.TaskReportCompleted, "once", start).
			Scan(&rows).Error; err != nil {
			return Metrics{}, err
		}
		for _, row := range rows {
			completedOnce[row.TaskID] = true
		}
	}

	expected := make(map[int64]models.Task, len(tasks))
	for _, task := range tasks {
		if task.Period == "once" && completedOnce[task.ID] {
			continue
		}
		expected[task.ID] = task
	}

	var reports []models.TaskReport
	if len(expected) > 0 {
		expectedIDs := make([]int64, 0, len(expected))
		for id := range expected {
			expectedIDs = append(expectedIDs, id)
		}
		if err := db.WithContext(ctx).
			Where("user_id = ? AND status = ? AND finished_at BETWEEN ? AND ?", user.ID, models.TaskReportCompleted, start, end).
			Where("task_id IN ?", expectedIDs).
			Find(&reports).Error; err != nil {
			return Metrics{}, err
		}
	}

	completed := make(map[int64]models.TaskReport, len(reports))
	for _, report := range reports {
		if _, exists := completed[report.TaskID]; !exists {
			completed[report.TaskID] = report
		}
	}

	totalDuration := 0
	withThreshold := 0
	withinThreshold := 0
	for taskID, report := range completed {
		if report.DurationMinutes != nil {
			totalDuration += *report.DurationMinutes
		}
		task := taskByID[taskID]
		if task.LowerThreshold != nil && task.UpperThreshold != nil {
			withThreshold++
			if report.DurationMinutes != nil && *report.DurationMinutes >= *task.LowerThreshold && *report.DurationMinutes <= *task.UpperThreshold {
				withinThreshold++
			}
		}
	}

	actualCost, actualRevenue, err := dailyRevenueAndCost(ctx, db, user.ID, start, end, expected)
	if err != nil {
		return Metrics{}, err
	}

	return Metrics{
		ActualRevenue:      formatNumber(actualRevenue),
		ActualCost:         formatNumber(actualCost),
		TotalDuration:      formatDuration(totalDuration),
		CompletionRate:     percentage(len(completed), len(expected)),
		TimeComplianceRate: percentage(withinThreshold, withThreshold),
	}, nil
}

// ActualVariableCostForUser menghitung kelebihan variable cost 30 hari.
func ActualVariableCostForUser(ctx context.Context, db *gorm.DB, userID int64, businessDate time.Time) (string, error) {
	_, end := DayRange(businessDate)
	start, _ := DayRange(businessDate.AddDate(0, 0, -29))
	totals, err := expenseFieldTotals(ctx, db, userID, start, end)
	if err != nil {
		return "", err
	}

	value := math.Max(0, totals[tokenListrikFieldName]-tokenListrikThreshold) +
		math.Max(0, totals[bahanBakarKendaraanFieldName]-bahanBakarKendaraanThreshold)
	return formatNumber(value), nil
}

// FinancialMatrixForUser membuat financial matrix untuk tanggal bisnis tertentu.
func FinancialMatrixForUser(ctx context.Context, db *gorm.DB, user *models.User, businessDate time.Time, planRevenue, actualRevenue, actualVariableCost *string) (FinancialMatrix, error) {
	tasks, err := TasksForUser(ctx, db, user, businessDate, true)
	if err != nil {
		return FinancialMatrix{}, err
	}

	totalEstimated := 0
	totalVariable := 0.0
	usesTaskFixedCost := len(tasks) > 0
	totalFixed := 0.0
	for _, task := range tasks {
		totalEstimated += task.TimeRequire
		totalVariable += float64(models.CostTotal(task.VariableCost))
		if task.FixedCost[models.FixedCostConfiguredKey] == 0 {
			usesTaskFixedCost = false
		}
		totalFixed += float64(models.CostTotal(task.FixedCost))
	}
	if !usesTaskFixedCost {
		totalFixed = DefaultDailyFixedCost
	}

	durations, err := completedDurationsByTask(ctx, db, user.ID, businessDate, tasks)
	if err != nil {
		return FinancialMatrix{}, err
	}
	totalActualDuration := 0
	for _, duration := range durations {
		totalActualDuration += duration
	}

	overage, _ := numericValue(derefString(actualVariableCost))
	totalActualVariable := 0.0
	if totalActualDuration > 0 {
		totalActualVariable = totalVariable + overage
	}
	totalPlanCost := totalFixed + totalVariable
	totalActualCost := 0.0
	if totalActualDuration > 0 {
		totalActualCost = totalFixed + totalActualVariable
	}

	planFixed := plannedFixedCosts(tasks, totalFixed, totalEstimated, usesTaskFixedCost)
	points := make([]FinancialMatrixPoint, 0, len(tasks))
	cumulativePlan := 0.0
	cumulativeActual := 0.0
	for index, task := range tasks {
		duration := durations[task.ID]
		planVariable := float64(models.CostTotal(task.VariableCost))
		planCost := planFixed[task.ID] + planVariable
		ratio := 0.0
		if totalActualDuration > 0 {
			ratio = float64(duration) / float64(totalActualDuration)
		}
		actualFixed := ratio * totalFixed
		actualVariable := ratio * totalActualVariable
		actualCost := actualFixed + actualVariable
		cumulativePlan += planCost
		cumulativeActual += actualCost
		points = append(points, FinancialMatrixPoint{
			Process: index + 1, TaskID: task.ID, TaskName: task.Name,
			EstimatedMinutes: task.TimeRequire, ActualDurationMinutes: duration,
			PlanFixedCost: roundMoney(planFixed[task.ID]), PlanVariableCost: roundMoney(planVariable), PlanCost: roundMoney(planCost),
			ActualFixedCost: roundMoney(actualFixed), ActualVariableCost: roundMoney(actualVariable), ActualCost: roundMoney(actualCost),
			CumulativePlanCost: roundMoney(cumulativePlan), CumulativeActualCost: roundMoney(cumulativeActual),
		})
	}

	plan := numericPointer(planRevenue)
	actual := numericPointer(actualRevenue)
	var planEbitda, actualEbitda *float64
	if plan != nil {
		value := roundMoney(*plan - totalPlanCost)
		planEbitda = &value
	}
	if actual != nil {
		value := roundMoney(*actual - totalActualCost)
		actualEbitda = &value
	}

	return FinancialMatrix{
		FixedCost: roundMoney(totalFixed), TotalVariableCost: roundMoney(totalVariable), TotalActualVariableCost: roundMoney(totalActualVariable),
		TotalEstimatedMinutes: totalEstimated, TotalActualDurationMinutes: totalActualDuration,
		TotalPlanCost: roundMoney(totalPlanCost), TotalActualCost: roundMoney(totalActualCost),
		PlanRevenue: plan, ActualRevenue: actual, PlanEbitda: planEbitda, ActualEbitda: actualEbitda, Points: points,
	}, nil
}

// SyncDailyMetrics menyimpan nilai terhitung setelah task selesai.
func SyncDailyMetrics(ctx context.Context, db *gorm.DB, user *models.User, businessDate time.Time) error {
	if !user.IsKdkmpManager() || user.SDMKdkmpEntryID == nil {
		return nil
	}

	entry, err := lockedDailyEntry(ctx, db, user, businessDate)
	if err != nil {
		return err
	}
	metrics, err := MetricsForUser(ctx, db, user, businessDate)
	if err != nil {
		return err
	}
	variableCost, err := ActualVariableCostForUser(ctx, db, user.ID, businessDate)
	if err != nil {
		return err
	}
	margin := CalculateActualEbitdaMargin(metrics.ActualRevenue)
	score := CalculatePerformanceScoring(derefString(entry.PlanRevenue), metrics.ActualRevenue, metrics.CompletionRate, metrics.TimeComplianceRate)
	review := PlanRevenueRequiresReview(derefString(entry.PlanRevenue))

	return db.WithContext(ctx).Model(&models.EbitdamaxKdkmp{}).Where("id = ?", entry.ID).Updates(map[string]any{
		"target_revenue":               strconv.Itoa(TargetRevenue),
		"actual_revenue":               metrics.ActualRevenue,
		"actual_cost":                  metrics.ActualCost,
		"actual_variable_cost":         variableCost,
		"actual_ebitda_margin":         margin,
		"performance_scoring":          score,
		"total_duration":               metrics.TotalDuration,
		"plan_revenue_requires_review": review,
		"updated_by":                   user.ID,
	}).Error
}

// TasksForUser mengambil task manager sesuai role dan pilihan harian.
func TasksForUser(ctx context.Context, db *gorm.DB, user *models.User, businessDate time.Time, includeInProgress bool) ([]models.Task, error) {
	if user.RoleID == nil {
		return []models.Task{}, nil
	}
	var tasks []models.Task
	if err := db.WithContext(ctx).
		Model(&models.Task{}).
		Where("tasks.is_active = ?", true).
		Where("EXISTS (SELECT 1 FROM task_roles tr WHERE tr.task_id = tasks.id AND tr.role_id = ?)", *user.RoleID).
		Order("CASE WHEN tasks.sort_order IS NULL THEN 1 ELSE 0 END").
		Order("tasks.sort_order").
		Order("tasks.id").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	if !user.IsKdkmpManager() {
		return tasks, nil
	}

	selection := NewSelectionService(db)
	selected, err := selection.DailySelectedTaskIDsForUser(ctx, user, businessDate)
	if err != nil {
		return nil, err
	}
	if includeInProgress && DateString(businessDate) == DateString(BusinessDate()) {
		inProgress, err := selection.InProgressTaskIDsForUser(ctx, user)
		if err != nil {
			return nil, err
		}
		selected = append(selected, inProgress...)
	}
	allowed := make(map[int64]bool, len(selected))
	for _, id := range selected {
		allowed[id] = true
	}
	filtered := make([]models.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.IsMandatory || allowed[task.ID] {
			filtered = append(filtered, task)
		}
	}
	return filtered, nil
}

// CalculateActualEbitdaMargin meniru formula margin legacy.
func CalculateActualEbitdaMargin(actualRevenue string) *string {
	revenue, ok := numericValue(actualRevenue)
	if !ok || revenue == 0 {
		return nil
	}
	value := formatNumber(((revenue-ActualEbitdaMarginFixedCost)/revenue)*100) + "%"
	return &value
}

// CalculatePerformanceScoring meniru bobot scoring legacy.
func CalculatePerformanceScoring(planRevenue, actualRevenue string, completionRate, timeComplianceRate float64) *string {
	plan, planOK := numericValue(planRevenue)
	actual, actualOK := numericValue(actualRevenue)
	revenueRate := 0.0
	if planOK && actualOK && plan > 0 {
		revenueRate = clampPercentage((actual / plan) * 100)
	}
	score := clampPercentage(
		clampPercentage(completionRate)*TaskCompletionWeight/100 +
			clampPercentage(timeComplianceRate)*TimeComplianceWeight/100 +
			revenueRate*RevenueWeight/100,
	)
	value := formatNumber(score) + "%"
	return &value
}

// PlanRevenueRequiresReview menandai plan di bawah target harian.
func PlanRevenueRequiresReview(planRevenue string) bool {
	value, ok := numericValue(planRevenue)
	return ok && value < TargetRevenue
}

func lockedDailyEntry(ctx context.Context, db *gorm.DB, user *models.User, businessDate time.Time) (*models.EbitdamaxKdkmp, error) {
	if user.SDMKdkmpEntryID == nil {
		return nil, fmt.Errorf("akun manager belum terhubung ke data KDKMP")
	}
	var entry models.EbitdamaxKdkmp
	err := db.WithContext(ctx).Where("sdm_kdkmp_entry_id = ? AND report_date = ?", *user.SDMKdkmpEntryID, DateString(businessDate)).Clauses(clause.Locking{Strength: "UPDATE"}).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		target := strconv.Itoa(TargetRevenue)
		entry = models.EbitdamaxKdkmp{SDMKdkmpEntryID: *user.SDMKdkmpEntryID, ReportDate: businessDate, TargetRevenue: &target, CreatedBy: &user.ID, UpdatedBy: &user.ID}
		if err := db.WithContext(ctx).Create(&entry).Error; err != nil {
			return nil, err
		}
		return &entry, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func dailyRevenueAndCost(ctx context.Context, db *gorm.DB, userID int64, start, end time.Time, expected map[int64]models.Task) (float64, float64, error) {
	type row struct {
		TaskID    int64
		TaskName  string
		FieldName string
		Value     *string
	}
	var rows []row
	if err := db.WithContext(ctx).
		Table("task_report_values AS trv").
		Select("tr.task_id, t.name AS task_name, taf.field_name, trv.value").
		Joins("JOIN task_reports tr ON tr.id = trv.task_report_id").
		Joins("JOIN tasks t ON t.id = tr.task_id").
		Joins("JOIN task_additional_fields taf ON taf.id = trv.task_additional_field_id").
		Where("tr.user_id = ? AND tr.status = ? AND tr.finished_at BETWEEN ? AND ?", userID, models.TaskReportCompleted, start, end).
		Where("t.name IN ?", []string{expenseTaskName, revenueTaskName}).
		Find(&rows).Error; err != nil {
		return 0, 0, err
	}
	cost := 0.0
	revenue := 0.0
	for _, row := range rows {
		value, ok := numericValue(derefString(row.Value))
		if !ok {
			continue
		}
		if row.TaskName == expenseTaskName && expected[row.TaskID].ID != 0 {
			cost += value
		}
		if row.TaskName == revenueTaskName && row.FieldName == revenueFieldName {
			revenue += value
		}
	}
	return cost, revenue, nil
}

func expenseFieldTotals(ctx context.Context, db *gorm.DB, userID int64, start, end time.Time) (map[string]float64, error) {
	totals := map[string]float64{tokenListrikFieldName: 0, bahanBakarKendaraanFieldName: 0}
	type row struct {
		FieldName string
		Value     *string
	}
	var rows []row
	err := db.WithContext(ctx).
		Table("task_report_values AS trv").
		Select("taf.field_name, trv.value").
		Joins("JOIN task_reports tr ON tr.id = trv.task_report_id").
		Joins("JOIN tasks t ON t.id = tr.task_id").
		Joins("JOIN task_additional_fields taf ON taf.id = trv.task_additional_field_id").
		Where("tr.user_id = ? AND tr.status = ? AND tr.finished_at BETWEEN ? AND ?", userID, models.TaskReportCompleted, start, end).
		Where("t.name = ? AND taf.field_name IN ?", expenseTaskName, []string{tokenListrikFieldName, bahanBakarKendaraanFieldName}).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if value, ok := numericValue(derefString(row.Value)); ok {
			totals[row.FieldName] += value
		}
	}
	return totals, nil
}

func completedDurationsByTask(ctx context.Context, db *gorm.DB, userID int64, businessDate time.Time, tasks []models.Task) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(tasks) == 0 {
		return result, nil
	}
	ids := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	start, end := DayRange(businessDate)
	type row struct {
		TaskID          int64
		DurationMinutes *int
	}
	var rows []row
	if err := db.WithContext(ctx).Model(&models.TaskReport{}).
		Select("task_id, duration_minutes").
		Where("user_id = ? AND task_id IN ? AND status = ? AND finished_at BETWEEN ? AND ?", userID, ids, models.TaskReportCompleted, start, end).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.DurationMinutes != nil {
			result[row.TaskID] += *row.DurationMinutes
		}
	}
	return result, nil
}

func plannedFixedCosts(tasks []models.Task, totalFixed float64, totalEstimated int, usesTaskFixedCost bool) map[int64]float64 {
	result := make(map[int64]float64, len(tasks))
	if len(tasks) == 0 {
		return result
	}
	if usesTaskFixedCost {
		for _, task := range tasks {
			result[task.ID] = float64(models.CostTotal(task.FixedCost))
		}
		return result
	}
	units := totalEstimated
	if units == 0 {
		units = len(tasks)
	}
	cumulative := 0.0
	for index, task := range tasks {
		if index == len(tasks)-1 {
			result[task.ID] = totalFixed - cumulative
			continue
		}
		weight := task.TimeRequire
		if totalEstimated == 0 {
			weight = 1
		}
		value := float64(weight) / float64(units) * totalFixed
		result[task.ID] = value
		cumulative += value
	}
	return result
}

func percentage(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return roundMoney(clampPercentage(float64(numerator) / float64(denominator) * 100))
}

func clampPercentage(value float64) float64 { return math.Min(100, math.Max(0, value)) }
func roundMoney(value float64) float64      { return math.Round(value*100) / 100 }

func formatNumber(value float64) string {
	formatted := strconv.FormatFloat(roundMoney(value), 'f', 2, 64)
	formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	if formatted == "" || formatted == "-0" {
		return "0"
	}
	return formatted
}

func formatDuration(minutes int) string {
	hours := minutes / 60
	remaining := minutes % 60
	if hours == 0 {
		return fmt.Sprintf("%d menit", remaining)
	}
	if remaining == 0 {
		return fmt.Sprintf("%d jam", hours)
	}
	return fmt.Sprintf("%d jam %d menit", hours, remaining)
}

func numericPointer(value *string) *float64 {
	parsed, ok := numericValue(derefString(value))
	if !ok {
		return nil
	}
	return &parsed
}

func numericValue(value string) (float64, bool) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.TrimPrefix(normalized, "rp")
	normalized = strings.TrimPrefix(normalized, ".")
	normalized = strings.ReplaceAll(normalized, " ", "")
	if normalized == "" || !numericValuePattern.MatchString(normalized) {
		return 0, false
	}
	lastDot, lastComma := strings.LastIndex(normalized, "."), strings.LastIndex(normalized, ",")
	switch {
	case lastDot >= 0 && lastComma >= 0 && lastComma > lastDot:
		normalized = strings.ReplaceAll(normalized, ".", "")
		normalized = strings.ReplaceAll(normalized, ",", ".")
	case lastDot >= 0 && lastComma >= 0:
		normalized = strings.ReplaceAll(normalized, ",", "")
	case lastDot >= 0:
		normalized = normalizeSingleSeparator(normalized, ".")
	case lastComma >= 0:
		normalized = normalizeSingleSeparator(normalized, ",")
	}
	parsed, err := strconv.ParseFloat(normalized, 64)
	return parsed, err == nil
}

func normalizeSingleSeparator(value, separator string) string {
	count := strings.Count(value, separator)
	last := strings.LastIndex(value, separator)
	decimalLength := len(value) - last - 1
	if count > 1 || decimalLength == 3 {
		return strings.ReplaceAll(value, separator, "")
	}
	if separator == "," {
		return strings.ReplaceAll(value, ",", ".")
	}
	return value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// SortTaskIDs membuat urutan stabil untuk data JSON dan pengujian.
func SortTaskIDs(ids []int64) []int64 {
	unique := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id > 0 {
			unique[id] = true
		}
	}
	result := make([]int64, 0, len(unique))
	for id := range unique {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
