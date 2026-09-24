package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

// TaskDashboardHandler godoc
//
//	@Summary      Dashboard task harian
//	@Description  Daftar task aktif sesuai role pengguna beserta status laporan pada periode berjalan, ringkasan, dan (untuk manager KDKMP) ringkasan kehadiran operasional.
//	@Tags         Task Dashboard
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Router       /api/v1/task-dashboard [get]
func TaskDashboardHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	ctx := c.Request.Context()
	businessDate := kdkmp.BusinessDate()

	query := AppDeps.DB.WithContext(ctx).Model(&models.Task{}).Where("tasks.is_active = ?", true)

	if user.RoleID == nil {
		query = query.Where("1 = 0")
	} else {
		query = query.Where(
			"EXISTS (SELECT 1 FROM task_roles tr WHERE tr.task_id = tasks.id AND tr.role_id = ?)",
			*user.RoleID,
		)
	}

	if user.IsKdkmpManager() {
		executionIDs, err := AppDeps.Selection.ExecutionTaskIDsForUser(ctx, user, businessDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat pemilihan task"})
			return
		}

		if len(executionIDs) == 0 {
			query = query.Where("tasks.is_mandatory = ?", true)
		} else {
			query = query.Where("(tasks.is_mandatory = ? OR tasks.id IN ?)", true, executionIDs)
		}
	}

	var tasks []models.Task
	err := query.
		Preload("TaskCategory").
		Preload("Roles").
		Preload("AdditionalFields", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order").Order("id")
		}).
		Order("CASE WHEN tasks.sort_order IS NULL THEN 1 ELSE 0 END").
		Order("tasks.sort_order").
		Order("tasks.id").
		Find(&tasks).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat task"})
		return
	}

	// Period key per task + laporan terbaru per (task, period).
	periodKeyByTask := make(map[int64]string, len(tasks))
	taskIDs := make([]int64, 0, len(tasks))
	periodKeys := make([]string, 0, len(tasks))
	seenPeriodKeys := make(map[string]bool)

	for i := range tasks {
		key := taskPeriodKey(tasks[i].Period, businessDate)
		periodKeyByTask[tasks[i].ID] = key
		taskIDs = append(taskIDs, tasks[i].ID)
		if !seenPeriodKeys[key] {
			seenPeriodKeys[key] = true
			periodKeys = append(periodKeys, key)
		}
	}

	reportByKey := make(map[string]*models.TaskReport)
	if len(taskIDs) > 0 {
		var reports []models.TaskReport
		err = AppDeps.DB.WithContext(ctx).
			Where("user_id = ? AND task_id IN ? AND period_key IN ?", user.ID, taskIDs, periodKeys).
			Order("created_at DESC").
			Find(&reports).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat laporan task"})
			return
		}

		for i := range reports {
			key := reportKey(reports[i].TaskID, derefString(reports[i].PeriodKey))
			if _, exists := reportByKey[key]; !exists {
				reportByKey[key] = &reports[i]
			}
		}
	}

	items := make([]gin.H, 0, len(tasks))
	for i := range tasks {
		report := reportByKey[reportKey(tasks[i].ID, periodKeyByTask[tasks[i].ID])]
		items = append(items, transformDashboardTask(&tasks[i], report, periodKeyByTask[tasks[i].ID]))
	}

	// Non-superadmin tidak melihat task yang sudah selesai pada periode ini.
	isSuperadmin := user.IsSuperadmin()
	visible := make([]gin.H, 0, len(items))
	summary := gin.H{"total": len(items), "pending": 0, "in_progress": 0, "completed": 0}

	for _, item := range items {
		summary[item["status"].(string)] = summary[item["status"].(string)].(int) + 1
		if isSuperadmin || item["status"] != models.TaskReportCompleted {
			visible = append(visible, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"business_date":          kdkmp.DateString(businessDate),
		"tasks":                  visible,
		"summary":                summary,
		"operational_attendance": operationalAttendanceForUser(c, user, businessDate),
	})
}

func operationalAttendanceForUser(c *gin.Context, user *models.User, businessDate time.Time) gin.H {
	if !user.IsKdkmpManager() {
		return nil
	}

	values := models.NormalizeOperationalAttendance(nil)
	isSaved := false

	if user.SDMKdkmpEntryID != nil {
		var entry models.EbitdamaxKdkmp
		err := AppDeps.DB.WithContext(c.Request.Context()).
			Where("sdm_kdkmp_entry_id = ? AND report_date = ?", *user.SDMKdkmpEntryID, kdkmp.DateString(businessDate)).
			First(&entry).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err == nil {
			values = models.NormalizeOperationalAttendance(entry.OperationalAttendance)
			isSaved = entry.HasConfirmedOperationalAttendance()
		}
	}

	allocated, available, err := AppDeps.Allocation.SummaryForUser(c.Request.Context(), user, businessDate, values, nil, false)
	if err != nil {
		return nil
	}

	return gin.H{
		"business_date": kdkmp.DateString(businessDate),
		"is_saved":      isSaved,
		"values":        values,
		"allocated":     allocated,
		"available":     available,
	}
}

func taskPeriodKey(period string, date time.Time) string {
	switch period {
	case "daily":
		return date.Format("2006-01-02")
	case "weekly":
		year, week := date.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case "monthly":
		return date.Format("2006-01")
	default:
		return "once"
	}
}

func transformDashboardTask(task *models.Task, report *models.TaskReport, periodKey string) gin.H {
	status := models.TaskReportPending
	documents := []gin.H{}

	if report != nil {
		status = report.Status
		documents = transformReportDocuments(report)
	}

	response := gin.H{
		"id":                           task.ID,
		"uuid":                         task.UUID,
		"sort_order":                   task.SortOrder,
		"name":                         task.Name,
		"description":                  task.Description,
		"bmc_status":                   task.BMCStatus,
		"bmc_status_label":             models.OptionLabel(models.TaskBmcStatusOptions, task.BMCStatus),
		"execution_time":               task.ExecutionTime.String(),
		"time_require":                 task.TimeRequire,
		"lower_time_threshold_minutes": task.LowerThreshold,
		"upper_time_threshold_minutes": task.UpperThreshold,
		"period":                       task.Period,
		"period_label":                 models.OptionLabel(models.TaskPeriodOptions, task.Period),
		"period_key":                   periodKey,
		"status":                       status,
		"status_label":                 models.TaskReportStatusLabel(status),
		"is_mandatory":                 task.IsMandatory,
		"documents":                    documents,
		"additional_fields":            dashboardAdditionalFields(task),
	}

	if task.TaskCategory != nil {
		response["task_category"] = gin.H{
			"id":   task.TaskCategory.ID,
			"name": task.TaskCategory.Name,
			"slug": task.TaskCategory.Slug,
		}
	} else {
		response["task_category"] = nil
	}

	roles := make([]gin.H, 0, len(task.Roles))
	for _, role := range task.Roles {
		roles = append(roles, roleSummary(role))
	}
	response["roles"] = roles
	if len(task.Roles) > 0 {
		response["role"] = roleSummary(task.Roles[0])
	} else {
		response["role"] = nil
	}

	return response
}

func dashboardAdditionalFields(task *models.Task) []gin.H {
	fields := make([]gin.H, 0, len(task.AdditionalFields))
	for _, field := range task.AdditionalFields {
		options := field.Options
		if options == nil {
			options = models.StringList{}
		}
		fields = append(fields, gin.H{
			"id":          field.ID,
			"label":       field.Label,
			"field_name":  field.FieldName,
			"input_type":  field.InputType,
			"show_when":   field.ShowWhen,
			"is_required": field.IsRequired,
			"options":     options,
		})
	}
	return fields
}

func transformReportDocuments(report *models.TaskReport) []gin.H {
	phases := []struct {
		key   string
		label string
		docs  models.StoredDocuments
	}{
		{"start", "Mulai", report.StartedDocuments},
		{"finish", "Selesai", report.FinishedDocuments},
	}

	result := make([]gin.H, 0)
	for _, phase := range phases {
		for index, document := range phase.docs {
			result = append(result, gin.H{
				"phase":        phase.key,
				"phase_label":  phase.label,
				"name":         document.OriginalName,
				"mime_type":    document.MimeType,
				"size":         document.Size,
				"preview_url":  fmt.Sprintf("/task-reports/%d/documents/%s/%d/preview", report.ID, phase.key, index),
				"download_url": fmt.Sprintf("/task-reports/%d/documents/%s/%d/download", report.ID, phase.key, index),
			})
		}
	}

	return result
}

func reportKey(taskID int64, periodKey string) string {
	return fmt.Sprintf("%d|%s", taskID, periodKey)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
