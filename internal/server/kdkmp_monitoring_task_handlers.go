package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

// KdkmpMonitoringTasksHandler godoc
//
//	@Summary	Detail task selesai per KDKMP & tanggal
//	@Description	Daftar laporan task yang diselesaikan manager KDKMP pada tanggal tertentu (foto, dokumen, nilai field).
//	@Tags		KDKMP Monitoring
//	@Produce	json
//	@Security	CookieAuth
//	@Param		entryID	path	int		true	"ID entry KDKMP"
//	@Param		date	path	string	true	"Tanggal (YYYY-MM-DD)"
//	@Success	200	{object}	map[string]any
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	422	{object}	map[string]string
//	@Router		/api/v1/admin/kdkmp-dashboard/{entryID}/tasks/{date} [get]
func KdkmpMonitoringTasksHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	entryID, err := strconv.ParseInt(c.Param("entryID"), 10, 64)
	if err != nil || entryID <= 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "KDKMP tidak ditemukan"})
		return
	}

	date := strings.TrimSpace(c.Param("date"))
	dateValue, err := time.ParseInLocation("2006-01-02", date, kdkmp.Location())
	if err != nil || dateValue.After(kdkmp.BusinessDate()) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Tanggal detail task tidak valid"})
		return
	}

	ctx := c.Request.Context()
	db := AppDeps.DB

	accessible, err := kdkmp.AccessibleManagedKdkmpQuery(ctx, db, user, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa akses KDKMP"})
		return
	}

	var entry models.SdmKdkmpEntry
	err = accessible.Session(&gorm.Session{}).
		Where("sdm_kdkmp_entries.id = ?", entryID).
		First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "KDKMP tidak ditemukan"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat KDKMP"})
		return
	}

	var manager models.User
	hasManager := true
	err = db.WithContext(ctx).
		Where("sdm_kdkmp_entry_id = ?", entryID).
		Order("users.id").
		First(&manager).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hasManager = false
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat akun manager"})
		return
	}

	reports := make([]models.TaskReport, 0)
	if hasManager {
		start, end := kdkmp.DayRange(dateValue)
		err = db.WithContext(ctx).
			Preload("Task.TaskCategory").
			Preload("Task.Roles").
			Where("user_id = ? AND status = ?", manager.ID, models.TaskReportCompleted).
			Where("(period_key = ? OR finished_at BETWEEN ? AND ? OR started_at BETWEEN ? AND ?)", date, start, end, start, end).
			Order("finished_at DESC").
			Find(&reports).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat laporan task"})
			return
		}
	}

	valuesByReport, fieldsByID, ok := monitoringReportValues(c, reports)
	if !ok {
		return
	}

	managerPayload := any(nil)
	if hasManager {
		managerPayload = gin.H{"name": manager.Name, "email": manager.Email}
	}

	payload := make([]gin.H, 0, len(reports))
	for index := range reports {
		report := &reports[index]
		payload = append(payload, monitoringTaskReportPayload(report, valuesByReport[report.ID], fieldsByID))
	}

	c.JSON(http.StatusOK, gin.H{
		"kdkmp_entry": gin.H{"id": entry.ID, "name": entry.NamaKoperasi, "manager": managerPayload},
		"date":        date,
		"reports":     payload,
	})
}

func monitoringReportValues(c *gin.Context, reports []models.TaskReport) (map[int64][]models.TaskReportValue, map[int64]models.TaskAdditionalField, bool) {
	valuesByReport := make(map[int64][]models.TaskReportValue, len(reports))
	fieldsByID := make(map[int64]models.TaskAdditionalField)
	if len(reports) == 0 {
		return valuesByReport, fieldsByID, true
	}

	reportIDs := make([]int64, 0, len(reports))
	for _, report := range reports {
		reportIDs = append(reportIDs, report.ID)
	}

	var values []models.TaskReportValue
	if err := AppDeps.DB.WithContext(c.Request.Context()).
		Where("task_report_id IN ?", reportIDs).
		Order("id").
		Find(&values).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat nilai laporan"})
		return nil, nil, false
	}

	fieldIDs := make([]int64, 0, len(values))
	seenField := make(map[int64]bool, len(values))
	for _, value := range values {
		valuesByReport[value.TaskReportID] = append(valuesByReport[value.TaskReportID], value)
		if !seenField[value.TaskAdditionalFieldID] {
			seenField[value.TaskAdditionalFieldID] = true
			fieldIDs = append(fieldIDs, value.TaskAdditionalFieldID)
		}
	}

	if len(fieldIDs) > 0 {
		var fields []models.TaskAdditionalField
		if err := AppDeps.DB.WithContext(c.Request.Context()).
			Where("id IN ?", fieldIDs).
			Find(&fields).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat field laporan"})
			return nil, nil, false
		}
		for _, field := range fields {
			fieldsByID[field.ID] = field
		}
	}

	return valuesByReport, fieldsByID, true
}

func monitoringTaskReportPayload(
	report *models.TaskReport,
	values []models.TaskReportValue,
	fields map[int64]models.TaskAdditionalField,
) gin.H {
	return gin.H{
		"id":                    report.ID,
		"uuid":                  report.UUID,
		"started_at":            report.StartedAt,
		"finished_at":           report.FinishedAt,
		"duration_minutes":      report.DurationMinutes,
		"manager_self_assigned": report.ManagerSelfAssigned,
		"status_label":          models.TaskReportStatusLabel(report.Status),
		"photos":                monitoringPhotosPayload(report),
		"documents":             monitoringDocumentsPayload(report),
		"values":                monitoringReportValuesPayload(values, fields),
		"task":                  monitoringTaskPayload(report.Task),
	}
}

func monitoringTaskPayload(task *models.Task) any {
	if task == nil {
		return nil
	}

	categoryPayload := any(nil)
	if task.TaskCategory != nil {
		categoryPayload = gin.H{"id": task.TaskCategory.ID, "name": task.TaskCategory.Name, "slug": task.TaskCategory.Slug}
	}

	rolesPayload := make([]gin.H, 0, len(task.Roles))
	for _, role := range task.Roles {
		rolesPayload = append(rolesPayload, gin.H{
			"id":          role.ID,
			"name":        role.Name,
			"slug":        role.Slug,
			"level":       role.Level,
			"level_label": models.RoleLevelLabel(role.Level),
		})
	}

	return gin.H{
		"id":                           task.ID,
		"uuid":                         task.UUID,
		"name":                         task.Name,
		"description":                  task.Description,
		"time_require":                 task.TimeRequire,
		"lower_time_threshold_minutes": task.LowerThreshold,
		"upper_time_threshold_minutes": task.UpperThreshold,
		"task_category":                categoryPayload,
		"roles":                        rolesPayload,
	}
}

func monitoringPhotosPayload(report *models.TaskReport) []gin.H {
	type photo struct {
		phase string
		path  *string
	}

	photos := make([]gin.H, 0, 2)
	for _, item := range []photo{{phase: "start", path: report.StartedPhoto}, {phase: "finish", path: report.FinishedPhoto}} {
		if item.path == nil || *item.path == "" {
			continue
		}

		phaseLabel := "Mulai"
		if item.phase == "finish" {
			phaseLabel = "Selesai"
		}
		extension := filepath.Ext(*item.path)
		photos = append(photos, gin.H{
			"phase":        item.phase,
			"phase_label":  phaseLabel,
			"name":         "Foto " + phaseLabel + extension,
			"preview_url":  "/task-reports/" + strconv.FormatInt(report.ID, 10) + "/photos/" + item.phase + "/preview",
			"download_url": "/task-reports/" + strconv.FormatInt(report.ID, 10) + "/photos/" + item.phase + "/download",
		})
	}

	return photos
}

func monitoringDocumentsPayload(report *models.TaskReport) []gin.H {
	documents := make([]gin.H, 0)
	for _, phase := range []string{"start", "finish"} {
		stored := report.StartedDocuments
		phaseLabel := "Mulai"
		if phase == "finish" {
			stored = report.FinishedDocuments
			phaseLabel = "Selesai"
		}

		for index, document := range stored {
			if document.Path == "" {
				continue
			}
			documents = append(documents, gin.H{
				"phase":        phase,
				"phase_label":  phaseLabel,
				"name":         document.OriginalName,
				"mime_type":    document.MimeType,
				"size":         document.Size,
				"preview_url":  "/task-reports/" + strconv.FormatInt(report.ID, 10) + "/documents/" + phase + "/" + strconv.Itoa(index) + "/preview",
				"download_url": "/task-reports/" + strconv.FormatInt(report.ID, 10) + "/documents/" + phase + "/" + strconv.Itoa(index) + "/download",
			})
		}
	}

	return documents
}

func monitoringReportValuesPayload(values []models.TaskReportValue, fields map[int64]models.TaskAdditionalField) []gin.H {
	ordered := make([]models.TaskReportValue, 0, len(values))
	for _, value := range values {
		if _, exists := fields[value.TaskAdditionalFieldID]; exists {
			ordered = append(ordered, value)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		left := fields[ordered[i].TaskAdditionalFieldID]
		right := fields[ordered[j].TaskAdditionalFieldID]
		if left.ShowWhen != right.ShowWhen {
			return left.ShowWhen < right.ShowWhen
		}
		if left.SortOrder != right.SortOrder {
			return left.SortOrder < right.SortOrder
		}
		return ordered[i].ID < ordered[j].ID
	})

	payload := make([]gin.H, 0, len(ordered))
	for _, value := range ordered {
		field := fields[value.TaskAdditionalFieldID]
		payload = append(payload, gin.H{
			"phase":       field.ShowWhen,
			"phase_label": monitoringShowWhenLabel(field.ShowWhen),
			"label":       field.Label,
			"value":       value.Value,
			"file":        monitoringValueFilePayload(field, value),
		})
	}

	return payload
}

func monitoringValueFilePayload(field models.TaskAdditionalField, value models.TaskReportValue) any {
	if field.InputType != "file" || value.Value == nil {
		return nil
	}

	var document models.StoredDocument
	if err := json.Unmarshal([]byte(*value.Value), &document); err != nil || document.Path == "" {
		return nil
	}

	name := document.OriginalName
	if name == "" {
		name = filepath.Base(document.Path)
	}

	return gin.H{
		"phase":        field.ShowWhen,
		"phase_label":  monitoringShowWhenLabel(field.ShowWhen),
		"name":         name,
		"mime_type":    document.MimeType,
		"size":         document.Size,
		"preview_url":  "/task-reports/" + strconv.FormatInt(value.TaskReportID, 10) + "/additional-fields/" + strconv.FormatInt(value.ID, 10) + "/preview",
		"download_url": "/task-reports/" + strconv.FormatInt(value.TaskReportID, 10) + "/additional-fields/" + strconv.FormatInt(value.ID, 10) + "/download",
	}
}

func monitoringShowWhenLabel(showWhen string) string {
	switch showWhen {
	case "start":
		return "Mulai Task"
	case "finish":
		return "Selesaikan Task"
	default:
		return showWhen
	}
}
