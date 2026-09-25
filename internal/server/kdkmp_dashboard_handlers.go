package server

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

var dailyMoneyPattern = regexp.MustCompile(`^\d+(?:\.\d{1,2})?$`)

type saveKdkmpDailyRequest struct {
	PlanRevenue  string  `json:"plan_revenue"`
	VariableCost *string `json:"variable_cost"`
}

type saveKdkmpTaskSelectionRequest struct {
	SelectedTaskIDs []int64 `json:"selected_task_ids"`
}

type saveOperationalAttendanceRequest struct {
	OperationalAttendance map[string]int `json:"operational_attendance"`
}

// RequireKdkmpManager membatasi fitur KDKMP hanya untuk Manager KDKMP.
func RequireKdkmpManager() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.CurrentUser(c)
		if user == nil || !user.IsKdkmpManager() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Fitur KDKMP hanya untuk Manager KDKMP"})
			return
		}
		c.Next()
	}
}

// KdkmpDashboardHandler godoc
//
//	@Summary	Dashboard harian Manager KDKMP
//	@Description	Metrik, financial matrix, dan riwayat harian Manager KDKMP.
//	@Tags		KDKMP Dashboard
//	@Produce	json
//	@Security	CookieAuth
//	@Param		date	query	string	false	"Tanggal matrix (YYYY-MM-DD)"
//	@Param		page	query	int		false	"Halaman riwayat"
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/kdkmp-dashboard [get]
func KdkmpDashboardHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	businessDate := kdkmp.BusinessDate()
	matrixDate, ok := kdkmpDashboardDate(c, businessDate)
	if !ok {
		return
	}

	kdkmpEntry, ok := currentKdkmpEntry(c, user)
	if !ok {
		return
	}
	if user.SDMKdkmpEntryID != nil {
		if err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			return kdkmp.SyncDailyMetrics(c.Request.Context(), tx, user, businessDate)
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyinkronkan metrik KDKMP"})
			return
		}
	}
	todayEntry, ok := dailyEntryFor(c, user, businessDate)
	if !ok {
		return
	}
	matrixEntry, ok := dailyEntryFor(c, user, matrixDate)
	if !ok {
		return
	}

	metrics, err := kdkmp.MetricsForUser(c.Request.Context(), AppDeps.DB, user, businessDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung metrik KDKMP"})
		return
	}
	actualVariableCost, err := kdkmp.ActualVariableCostForUser(c.Request.Context(), AppDeps.DB, user.ID, matrixDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung variable cost"})
		return
	}
	matrixMetrics, err := kdkmp.MetricsForUser(c.Request.Context(), AppDeps.DB, user, matrixDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung financial matrix"})
		return
	}
	matrix, err := kdkmp.FinancialMatrixForUser(
		c.Request.Context(), AppDeps.DB, user, matrixDate,
		entryField(matrixEntry, func(entry *models.EbitdamaxKdkmp) *string { return entry.PlanRevenue }),
		stringPointer(matrixMetrics.ActualRevenue),
		&actualVariableCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung financial matrix"})
		return
	}

	history, ok := kdkmpHistory(c, user, businessDate)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"business_date":         kdkmp.DateString(businessDate),
		"financial_matrix_date": kdkmp.DateString(matrixDate),
		"kdkmp":                 kdkmpIdentity(kdkmpEntry),
		"today_entry":           kdkmpDailyEntryResponse(todayEntry),
		"computed_values":       computedValues(todayEntry, metrics),
		"financial_matrix":      matrix,
		"history":               history,
	})
}

// KdkmpDashboardInputHandler godoc
//
//	@Summary	Data input harian Manager KDKMP
//	@Tags		KDKMP Dashboard
//	@Produce	json
//	@Security	CookieAuth
//	@Success	200	{object}	map[string]any
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Router		/api/v1/kdkmp-dashboard/input [get]
func KdkmpDashboardInputHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	businessDate := kdkmp.BusinessDate()
	kdkmpEntry, ok := currentKdkmpEntry(c, user)
	if !ok {
		return
	}
	todayEntry, ok := dailyEntryFor(c, user, businessDate)
	if !ok {
		return
	}
	metrics, err := kdkmp.MetricsForUser(c.Request.Context(), AppDeps.DB, user, businessDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghitung metrik KDKMP"})
		return
	}
	selection, err := kdkmpTaskSelectionResponse(c, user, businessDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat pilihan task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"business_date":   kdkmp.DateString(businessDate),
		"kdkmp":           kdkmpIdentity(kdkmpEntry),
		"today_entry":     kdkmpDailyEntryResponse(todayEntry),
		"computed_values": computedValues(todayEntry, metrics),
		"task_selection":  selection,
	})
}

// UpdateKdkmpDailyHandler godoc
//
//	@Summary	Simpan input harian KDKMP
//	@Tags		KDKMP Dashboard
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		payload	body		saveKdkmpDailyRequest	true	"Input harian"
//	@Success	200		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	422		{object}	map[string]string
//	@Router		/api/v1/kdkmp-dashboard/today [put]
func UpdateKdkmpDailyHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}
	if user.SDMKdkmpEntryID == nil {
		c.JSON(http.StatusForbidden, gin.H{"message": "Akun Manager belum terhubung ke data KDKMP"})
		return
	}

	var req saveKdkmpDailyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data input harian tidak valid"})
		return
	}
	planRevenue, valid := validatedDailyMoney(req.PlanRevenue, true)
	if !valid {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Plan revenue wajib berupa angka positif atau nol dengan maksimal dua desimal"})
		return
	}
	variableCost := ""
	if req.VariableCost != nil {
		var validCost bool
		variableCost, validCost = validatedDailyMoney(*req.VariableCost, false)
		if !validCost {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Variable cost harus berupa angka positif atau nol dengan maksimal dua desimal"})
			return
		}
	}

	err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		entry, err := lockedOrNewKdkmpDailyEntry(c, tx, user, kdkmp.BusinessDate())
		if err != nil {
			return err
		}
		metrics, err := kdkmp.MetricsForUser(c.Request.Context(), tx, user, kdkmp.BusinessDate())
		if err != nil {
			return err
		}
		actualVariableCost, err := kdkmp.ActualVariableCostForUser(c.Request.Context(), tx, user.ID, kdkmp.BusinessDate())
		if err != nil {
			return err
		}
		margin := kdkmp.CalculateActualEbitdaMargin(metrics.ActualRevenue)
		score := kdkmp.CalculatePerformanceScoring(planRevenue, metrics.ActualRevenue, metrics.CompletionRate, metrics.TimeComplianceRate)
		updates := map[string]any{
			"target_revenue":               strconv.Itoa(kdkmp.TargetRevenue),
			"plan_revenue":                 planRevenue,
			"plan_cost":                    nullableString(req.VariableCost, variableCost),
			"actual_revenue":               metrics.ActualRevenue,
			"actual_cost":                  metrics.ActualCost,
			"actual_variable_cost":         actualVariableCost,
			"actual_ebitda_margin":         margin,
			"performance_scoring":          score,
			"total_duration":               metrics.TotalDuration,
			"plan_revenue_requires_review": kdkmp.PlanRevenueRequiresReview(planRevenue),
			"updated_by":                   user.ID,
		}
		return tx.Model(&models.EbitdamaxKdkmp{}).Where("id = ?", entry.ID).Updates(updates).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan input harian"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dashboard harian KDKMP berhasil disimpan"})
}

// UpdateKdkmpTaskSelectionHandler godoc
//
//	@Summary	Simpan pilihan task harian KDKMP
//	@Tags		KDKMP Dashboard
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		payload	body		saveKdkmpTaskSelectionRequest	true	"Pilihan task"
//	@Success	200		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	422		{object}	map[string]string
//	@Router		/api/v1/kdkmp-dashboard/today/task-selection [put]
func UpdateKdkmpTaskSelectionHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}
	if user.SDMKdkmpEntryID == nil {
		c.JSON(http.StatusForbidden, gin.H{"message": "Akun Manager belum terhubung ke data KDKMP"})
		return
	}

	var req saveKdkmpTaskSelectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data pilihan task tidak valid"})
		return
	}
	selected := kdkmp.SortTaskIDs(req.SelectedTaskIDs)
	expanded, err := AppDeps.Selection.ExpandSelectedTaskIDsToBMCBundles(c.Request.Context(), user, selected)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Task pilihan tidak tersedia untuk role Anda"})
		return
	}

	err = AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		entry, err := lockedOrNewKdkmpDailyEntry(c, tx, user, kdkmp.BusinessDate())
		if err != nil {
			return err
		}
		lockedIDs, err := lockedInProgressOptionalTaskIDs(c, tx, user)
		if err != nil {
			return err
		}
		selectedSet := make(map[int64]bool, len(expanded))
		for _, id := range expanded {
			selectedSet[id] = true
		}
		for _, id := range lockedIDs {
			if !selectedSet[id] {
				return reportError{status: http.StatusUnprocessableEntity, message: "Task yang sedang dikerjakan tidak dapat dilepas dari pilihan hari ini."}
			}
		}
		return tx.Model(&models.EbitdamaxKdkmp{}).Where("id = ?", entry.ID).Updates(map[string]any{
			"selected_task_ids": models.IntList(expanded),
			"updated_by":        user.ID,
		}).Error
	})
	if err != nil {
		respondKdkmpDashboardError(c, err, "Gagal menyimpan pilihan task")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pilihan task hari ini berhasil disimpan"})
}

// UpdateKdkmpOperationalAttendanceHandler godoc
//
//	@Summary	Simpan kehadiran operasional KDKMP
//	@Tags		KDKMP Dashboard
//	@Accept		json
//	@Produce	json
//	@Security	CookieAuth
//	@Param		payload	body		saveOperationalAttendanceRequest	true	"Kehadiran operasional"
//	@Success	200		{object}	map[string]string
//	@Failure	403		{object}	map[string]string
//	@Failure	422		{object}	map[string]string
//	@Router		/api/v1/kdkmp-dashboard/today/operational-attendance [put]
func UpdateKdkmpOperationalAttendanceHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}
	if user.SDMKdkmpEntryID == nil {
		c.JSON(http.StatusForbidden, gin.H{"message": "Akun Manager belum terhubung ke data KDKMP"})
		return
	}

	var req saveOperationalAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data kehadiran tidak valid"})
		return
	}
	attendance, message := validatedOperationalAttendance(req.OperationalAttendance)
	if message != "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": message})
		return
	}

	err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		entry, err := lockedOrNewKdkmpDailyEntry(c, tx, user, kdkmp.BusinessDate())
		if err != nil {
			return err
		}
		allocated, _, err := AppDeps.Allocation.SummaryForUserWithTx(tx, user, kdkmp.BusinessDate(), attendance, nil, true)
		if err != nil {
			return err
		}
		for _, key := range models.OperationalAttendanceRoleKeys {
			if attendance[key] < allocated[key] {
				return reportError{status: http.StatusUnprocessableEntity, message: fmt.Sprintf("Jumlah anggota %s tidak boleh lebih kecil dari %d anggota yang sedang dialokasikan.", models.OperationalAttendanceRoleLabels[key], allocated[key])}
			}
		}
		now := time.Now()
		return tx.Model(&models.EbitdamaxKdkmp{}).Where("id = ?", entry.ID).Updates(map[string]any{
			"operational_attendance":          models.JSONIntMap(attendance),
			"operational_attendance_saved_at": now,
			"updated_by":                      user.ID,
		}).Error
	})
	if err != nil {
		respondKdkmpDashboardError(c, err, "Gagal menyimpan kehadiran")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kehadiran anggota hari ini berhasil disimpan"})
}

func currentKdkmpEntry(c *gin.Context, user *models.User) (*models.SdmKdkmpEntry, bool) {
	if user.SDMKdkmpEntryID == nil {
		return nil, true
	}
	var entry models.SdmKdkmpEntry
	err := AppDeps.DB.WithContext(c.Request.Context()).First(&entry, *user.SDMKdkmpEntryID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, true
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat data KDKMP"})
		return nil, false
	}
	return &entry, true
}

func dailyEntryFor(c *gin.Context, user *models.User, date time.Time) (*models.EbitdamaxKdkmp, bool) {
	if user.SDMKdkmpEntryID == nil {
		return nil, true
	}
	var entry models.EbitdamaxKdkmp
	err := AppDeps.DB.WithContext(c.Request.Context()).Where("sdm_kdkmp_entry_id = ? AND report_date = ?", *user.SDMKdkmpEntryID, kdkmp.DateString(date)).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, true
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat data harian KDKMP"})
		return nil, false
	}
	return &entry, true
}

func kdkmpDashboardDate(c *gin.Context, businessDate time.Time) (time.Time, bool) {
	value := strings.TrimSpace(c.Query("date"))
	if value == "" {
		return businessDate, true
	}
	date, err := time.ParseInLocation("2006-01-02", value, kdkmp.Location())
	if err != nil || date.After(businessDate) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Tanggal matrix harus valid dan tidak melebihi hari ini"})
		return time.Time{}, false
	}
	return date, true
}

func kdkmpHistory(c *gin.Context, user *models.User, businessDate time.Time) (gin.H, bool) {
	if user.SDMKdkmpEntryID == nil {
		return gin.H{"data": []gin.H{}, "page": 1, "per_page": 10, "total": 0, "last_page": 1}, true
	}
	page := 1
	if requested, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && requested > 0 {
		page = requested
	}
	const perPage = 10
	base := AppDeps.DB.WithContext(c.Request.Context()).Model(&models.EbitdamaxKdkmp{}).
		Where("sdm_kdkmp_entry_id = ? AND report_date < ?", *user.SDMKdkmpEntryID, kdkmp.DateString(businessDate))
	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat riwayat KDKMP"})
		return nil, false
	}
	var entries []models.EbitdamaxKdkmp
	if err := base.Order("report_date DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat riwayat KDKMP"})
		return nil, false
	}
	data := make([]gin.H, 0, len(entries))
	for index := range entries {
		data = append(data, kdkmpDailyEntryResponse(&entries[index]))
	}
	lastPage := int(math.Ceil(float64(total) / perPage))
	if lastPage == 0 {
		lastPage = 1
	}
	return gin.H{"data": data, "page": page, "per_page": perPage, "total": total, "last_page": lastPage}, true
}

func kdkmpTaskSelectionResponse(c *gin.Context, user *models.User, businessDate time.Time) (gin.H, error) {
	if user.RoleID == nil {
		return gin.H{"tasks": []gin.H{}, "selected_task_ids": []int64{}}, nil
	}
	var tasks []models.Task
	err := AppDeps.DB.WithContext(c.Request.Context()).Model(&models.Task{}).
		Preload("TaskCategory").
		Where("tasks.is_active = ?", true).
		Where("EXISTS (SELECT 1 FROM task_roles tr WHERE tr.task_id = tasks.id AND tr.role_id = ?)", *user.RoleID).
		Order("CASE WHEN tasks.sort_order IS NULL THEN 1 ELSE 0 END").Order("tasks.sort_order").Order("tasks.id").Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	selected, err := AppDeps.Selection.ExecutionTaskIDsForUser(c.Request.Context(), user, businessDate)
	if err != nil {
		return nil, err
	}
	locked, err := AppDeps.Selection.InProgressTaskIDsForUser(c.Request.Context(), user)
	if err != nil {
		return nil, err
	}
	lockedSet := make(map[int64]bool, len(locked))
	for _, id := range locked {
		lockedSet[id] = true
	}
	items := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, gin.H{
			"id": task.ID, "name": task.Name, "description": task.Description, "execution_time": task.ExecutionTime.String(),
			"time_require": task.TimeRequire, "is_mandatory": task.IsMandatory, "is_locked": lockedSet[task.ID],
			"bmc_status": task.BMCStatus, "bmc_status_label": models.OptionLabel(models.TaskBmcStatusOptions, task.BMCStatus),
			"task_category_name": taskCategoryName(task.TaskCategory),
		})
	}
	return gin.H{"tasks": items, "selected_task_ids": kdkmp.SortTaskIDs(selected)}, nil
}

func lockedOrNewKdkmpDailyEntry(c *gin.Context, tx *gorm.DB, user *models.User, businessDate time.Time) (*models.EbitdamaxKdkmp, error) {
	if user.SDMKdkmpEntryID == nil {
		return nil, reportError{status: http.StatusForbidden, message: "Akun Manager belum terhubung ke data KDKMP"}
	}
	var entry models.EbitdamaxKdkmp
	err := tx.WithContext(c.Request.Context()).Where("sdm_kdkmp_entry_id = ? AND report_date = ?", *user.SDMKdkmpEntryID, kdkmp.DateString(businessDate)).Clauses(lockClause()).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		target := strconv.Itoa(kdkmp.TargetRevenue)
		entry = models.EbitdamaxKdkmp{SDMKdkmpEntryID: *user.SDMKdkmpEntryID, ReportDate: businessDate, TargetRevenue: &target, CreatedBy: &user.ID, UpdatedBy: &user.ID}
		if err := tx.WithContext(c.Request.Context()).Create(&entry).Error; err != nil {
			return nil, err
		}
		return &entry, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func lockedInProgressOptionalTaskIDs(c *gin.Context, tx *gorm.DB, user *models.User) ([]int64, error) {
	var ids []int64
	err := tx.WithContext(c.Request.Context()).Table("task_reports AS tr").
		Joins("JOIN tasks t ON t.id = tr.task_id").
		Where("tr.user_id = ? AND tr.status = ? AND t.is_mandatory = ?", user.ID, models.TaskReportInProgress, false).
		Pluck("tr.task_id", &ids).Error
	return ids, err
}

func kdkmpIdentity(entry *models.SdmKdkmpEntry) gin.H {
	if entry == nil {
		return nil
	}
	return gin.H{"id": entry.ID, "nik": entry.NIK, "name": entry.NamaKoperasi, "desa": entry.Desa, "kecamatan": entry.Kecamatan, "kota_kabupaten": entry.KotaKabupaten, "provinsi": entry.Provinsi}
}

func kdkmpDailyEntryResponse(entry *models.EbitdamaxKdkmp) gin.H {
	if entry == nil {
		return nil
	}
	attendance := models.NormalizeOperationalAttendance(entry.OperationalAttendance)
	return gin.H{
		"id": entry.ID, "report_date": kdkmp.DateString(entry.ReportDate), "is_complete": true,
		"target_revenue": entry.TargetRevenue, "plan_revenue": entry.PlanRevenue, "actual_revenue": entry.ActualRevenue,
		"variable_cost": entry.PlanCost, "actual_cost": entry.ActualCost, "actual_ebitda_margin": entry.ActualEbitdaMargin,
		"total_duration": entry.TotalDuration, "performance_scoring": entry.PerformanceScoring,
		"plan_revenue_requires_review": entry.PlanRevenueRequiresReview, "operational_attendance": attendance,
		"operational_attendance_saved_at": entry.OperationalAttendanceSavedAt, "updated_at": entry.UpdatedAt,
	}
}

func computedValues(entry *models.EbitdamaxKdkmp, metrics kdkmp.Metrics) gin.H {
	score := kdkmp.CalculatePerformanceScoring(derefString(entryField(entry, func(value *models.EbitdamaxKdkmp) *string { return value.PlanRevenue })), metrics.ActualRevenue, metrics.CompletionRate, metrics.TimeComplianceRate)
	return gin.H{"target_revenue": strconv.Itoa(kdkmp.TargetRevenue), "actual_revenue": metrics.ActualRevenue, "actual_cost": metrics.ActualCost, "total_duration": metrics.TotalDuration, "performance_scoring": score, "task_completion_rate": metrics.CompletionRate, "time_compliance_rate": metrics.TimeComplianceRate}
}

func entryField(entry *models.EbitdamaxKdkmp, field func(*models.EbitdamaxKdkmp) *string) *string {
	if entry == nil {
		return nil
	}
	return field(entry)
}
func stringPointer(value string) *string { return &value }
func taskCategoryName(category *models.TaskCategory) *string {
	if category == nil {
		return nil
	}
	return &category.Name
}

func validatedDailyMoney(value string, required bool) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", !required
	}
	if len(value) > 255 || !dailyMoneyPattern.MatchString(value) {
		return "", false
	}
	return value, true
}

func nullableString(raw *string, normalized string) any {
	if raw == nil {
		return nil
	}
	return normalized
}

func validatedOperationalAttendance(values map[string]int) (map[string]int, string) {
	if values == nil {
		return nil, "Kehadiran operasional wajib diisi"
	}
	for key := range values {
		known := false
		for _, allowed := range models.OperationalAttendanceRoleKeys {
			if key == allowed {
				known = true
				break
			}
		}
		if !known {
			return nil, "Role kehadiran tidak dikenali"
		}
	}
	attendance := make(map[string]int, len(models.OperationalAttendanceRoleKeys))
	for _, key := range models.OperationalAttendanceRoleKeys {
		value, exists := values[key]
		if !exists || value < 0 {
			return nil, fmt.Sprintf("Jumlah %s wajib diisi dengan angka bulat minimal 0", models.OperationalAttendanceRoleLabels[key])
		}
		attendance[key] = value
	}
	return attendance, ""
}

func respondKdkmpDashboardError(c *gin.Context, err error, fallback string) {
	var requestError reportError
	if errors.As(err, &requestError) {
		c.JSON(requestError.status, gin.H{"message": requestError.message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": fallback})
}
