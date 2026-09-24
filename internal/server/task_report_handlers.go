package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

// lockClause menghasilkan klausa SELECT ... FOR UPDATE.
func lockClause() clause.Locking {
	return clause.Locking{Strength: "UPDATE"}
}

const (
	photoFieldStart      = "started_photo"
	photoFieldFinish     = "finished_photo"
	maxPhotoSize         = 3 << 20
	maxDocumentSize      = 10 << 20
	maxDocumentsPerPhase = 10
)

var allowedDocumentExtensions = map[string]bool{
	"pdf": true, "doc": true, "docx": true, "xls": true, "xlsx": true,
	"ppt": true, "pptx": true, "txt": true, "csv": true,
	"jpg": true, "jpeg": true, "png": true,
}

var allowedPhotoExtensions = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "webp": true, "gif": true,
}

// reportError adalah error dengan status HTTP dan pesan untuk klien.
type reportError struct {
	status  int
	message string
}

func (e reportError) Error() string {
	return e.message
}

func newReportError(status int, message string) reportError {
	return reportError{status: status, message: message}
}

// StartTaskReportHandler godoc
//
//	@Summary      Mulai task
//	@Description  Memulai task: unggah foto (wajib), dokumen (opsional), isi field tambahan fase mulai, dan alokasi anggota (manager KDKMP).
//	@Tags         Task Reports
//	@Accept       multipart/form-data
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id                path      int     true   "ID task"
//	@Param        started_photo     formData  file    true   "Foto mulai (maks 3 MB)"
//	@Param        documents         formData  []file  false  "Dokumen pendukung (maks 10 file, 10 MB/file)"
//	@Param        values            formData  string  false  "JSON nilai field tambahan: {\"field_name\": \"nilai\"}"
//	@Param        value_files       formData  file    false  "File field tambahan dengan key value_files[field_name]"
//	@Param        member_allocations formData string  false  "JSON alokasi anggota (manager KDKMP)"
//	@Param        manager_self_assigned formData string false "true bila manager mengerjakan sendiri"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Failure      422  {object}  map[string]string
//	@Router       /api/v1/tasks/{id}/start [post]
func StartTaskReportHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	task, ok := findTask(c)
	if !ok {
		return
	}

	allowed, err := taskAccessAllowed(c.Request.Context(), user, task, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa akses task"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"message": "Task pilihan belum dipilih untuk dilaksanakan hari ini."})
		return
	}

	form, ok := parseTaskReportForm(c)
	if !ok {
		return
	}

	photo, ok := validatePhotoUpload(c, form, photoFieldStart, "Foto mulai task wajib diunggah.")
	if !ok {
		return
	}

	documents, ok := validateDocumentUploads(c, form.File["documents"])
	if !ok {
		return
	}

	values, ok := parseValuesJSON(c, c.PostForm("values"))
	if !ok {
		return
	}

	isManager := user.IsKdkmpManager()
	memberAllocations, managerSelfAssigned, ok := parseMemberAllocationFields(c, isManager)
	if !ok {
		return
	}

	businessDate := kdkmp.BusinessDate()
	businessNow := time.Now().In(kdkmp.Location())
	periodKey := taskPeriodKey(task.Period, businessNow)

	var storedFiles []models.StoredDocument

	err = AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var completedCount int64
		if err := tx.Model(&models.TaskReport{}).
			Where("task_id = ? AND user_id = ? AND period_key = ? AND status = ?", task.ID, user.ID, periodKey, models.TaskReportCompleted).
			Count(&completedCount).Error; err != nil {
			return err
		}
		if completedCount > 0 {
			return newReportError(http.StatusUnprocessableEntity, "Task sudah diselesaikan untuk periode ini.")
		}

		var attendance *models.EbitdamaxKdkmp
		if isManager {
			entry, err := lockedAttendanceForStart(tx, user, businessDate)
			if err != nil {
				return err
			}
			attendance = entry
		}

		var report models.TaskReport
		err := tx.Where("task_id = ? AND user_id = ? AND period_key = ? AND status = ?",
			task.ID, user.ID, periodKey, models.TaskReportInProgress).
			Clauses(lockClause()).First(&report).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var reportID *int64
		if report.ID > 0 {
			reportID = &report.ID
		}

		if isManager {
			if err := validateAllocations(tx, user, businessDate, attendance, memberAllocations, reportID); err != nil {
				return err
			}
		}

		if report.ID == 0 {
			report = models.TaskReport{
				UUID:      uuid.NewString(),
				TaskID:    task.ID,
				UserID:    user.ID,
				PeriodKey: &periodKey,
				Status:    models.TaskReportInProgress,
			}
			if err := tx.Create(&report).Error; err != nil {
				return err
			}
		}

		if len(report.StartedDocuments)+len(documents) > maxDocumentsPerPhase {
			return newReportError(http.StatusUnprocessableEntity, "Total dokumen untuk tahap ini maksimal 10 file.")
		}

		photoDocument, err := storePhoto(c.Request.Context(), report.UUID, "start", photo)
		if err != nil {
			return err
		}
		storedFiles = append(storedFiles, photoDocument)

		newDocuments, err := AppDeps.TaskReports.StoreDocuments(c.Request.Context(), report.UUID, "start", documents)
		if err != nil {
			return err
		}
		storedFiles = append(storedFiles, newDocuments...)

		startedAt := report.StartedAt
		if startedAt == nil {
			now := time.Now()
			startedAt = &now
		}

		updates := map[string]any{
			"started_photo":     photoDocument.Path,
			"started_documents": append(report.StartedDocuments, newDocuments...),
			"started_at":        startedAt,
			"status":            models.TaskReportInProgress,
		}
		if isManager {
			updates["member_allocations"] = memberAllocations
			updates["manager_self_assigned"] = managerSelfAssigned
		}

		if err := tx.Model(&models.TaskReport{}).Where("id = ?", report.ID).Updates(updates).Error; err != nil {
			return err
		}

		return syncTaskReportValues(c.Request.Context(), tx, &report, task, form, values, "start", &storedFiles)
	})

	if err != nil {
		AppDeps.TaskReports.RemoveAll(c.Request.Context(), storedFiles)
		respondReportError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task berhasil dimulai."})
}

// FinishTaskReportHandler godoc
//
//	@Summary      Selesaikan task
//	@Description  Menyelesaikan task: unggah foto selesai (wajib), dokumen (opsional), dan isi field tambahan fase selesai.
//	@Tags         Task Reports
//	@Accept       multipart/form-data
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id              path      int     true   "ID task"
//	@Param        finished_photo  formData  file    true   "Foto selesai (maks 3 MB)"
//	@Param        documents       formData  []file  false  "Dokumen pendukung (maks 10 file, 10 MB/file)"
//	@Param        values          formData  string  false  "JSON nilai field tambahan"
//	@Param        value_files     formData  file    false  "File field tambahan dengan key value_files[field_name]"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Failure      422  {object}  map[string]string
//	@Router       /api/v1/tasks/{id}/finish [post]
func FinishTaskReportHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	task, ok := findTask(c)
	if !ok {
		return
	}

	allowed, err := taskAccessAllowed(c.Request.Context(), user, task, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa akses task"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"message": "Anda tidak memiliki akses"})
		return
	}

	form, ok := parseTaskReportForm(c)
	if !ok {
		return
	}

	photo, ok := validatePhotoUpload(c, form, photoFieldFinish, "Foto selesai task wajib diunggah.")
	if !ok {
		return
	}

	documents, ok := validateDocumentUploads(c, form.File["documents"])
	if !ok {
		return
	}

	values, ok := parseValuesJSON(c, c.PostForm("values"))
	if !ok {
		return
	}

	businessNow := time.Now().In(kdkmp.Location())
	periodKey := taskPeriodKey(task.Period, businessNow)

	var storedFiles []models.StoredDocument

	err = AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var report models.TaskReport
		err := tx.Where("task_id = ? AND user_id = ? AND period_key = ? AND status = ?",
			task.ID, user.ID, periodKey, models.TaskReportInProgress).
			Order("started_at DESC").
			First(&report).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newReportError(http.StatusUnprocessableEntity, "Task belum dimulai untuk periode ini.")
		}
		if err != nil {
			return err
		}

		if len(report.FinishedDocuments)+len(documents) > maxDocumentsPerPhase {
			return newReportError(http.StatusUnprocessableEntity, "Total dokumen untuk tahap ini maksimal 10 file.")
		}

		photoDocument, err := storePhoto(c.Request.Context(), report.UUID, "finish", photo)
		if err != nil {
			return err
		}
		storedFiles = append(storedFiles, photoDocument)

		newDocuments, err := AppDeps.TaskReports.StoreDocuments(c.Request.Context(), report.UUID, "finish", documents)
		if err != nil {
			return err
		}
		storedFiles = append(storedFiles, newDocuments...)

		finishedAt := time.Now()
		var duration *int
		if report.StartedAt != nil {
			minutes := int(math.Floor(finishedAt.Sub(*report.StartedAt).Seconds() / 60))
			if minutes < 0 {
				minutes = 0
			}
			duration = &minutes
		}

		updates := map[string]any{
			"finished_photo":     photoDocument.Path,
			"finished_documents": append(report.FinishedDocuments, newDocuments...),
			"finished_at":        finishedAt,
			"duration_minutes":   duration,
			"status":             models.TaskReportCompleted,
		}

		if err := tx.Model(&models.TaskReport{}).Where("id = ?", report.ID).Updates(updates).Error; err != nil {
			return err
		}

		if err := syncTaskReportValues(c.Request.Context(), tx, &report, task, form, values, "finish", &storedFiles); err != nil {
			return err
		}

		return kdkmp.SyncDailyMetrics(c.Request.Context(), tx, user, kdkmp.BusinessDate())
	})

	if err != nil {
		AppDeps.TaskReports.RemoveAll(c.Request.Context(), storedFiles)
		respondReportError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task berhasil diselesaikan."})
}

// taskAccessAllowed mengecek task aktif + role user terpasang; untuk start
// juga mengecek pemilihan task KDKMP.
func taskAccessAllowed(ctx context.Context, user *models.User, task *models.Task, forStart bool) (bool, error) {
	if !task.IsActive || user.RoleID == nil {
		return false, nil
	}

	var roleCount int64
	err := AppDeps.DB.WithContext(ctx).Table("task_roles").
		Where("task_id = ? AND role_id = ?", task.ID, *user.RoleID).
		Count(&roleCount).Error
	if err != nil {
		return false, err
	}
	if roleCount == 0 {
		return false, nil
	}

	if forStart {
		return AppDeps.Selection.CanStartTask(ctx, user, task, kdkmp.BusinessDate())
	}

	return true, nil
}

func lockedAttendanceForStart(tx *gorm.DB, user *models.User, businessDate time.Time) (*models.EbitdamaxKdkmp, error) {
	if user.SDMKdkmpEntryID == nil {
		return nil, newReportError(http.StatusUnprocessableEntity, "Akun Manager belum terhubung ke data KDKMP.")
	}

	var entry models.EbitdamaxKdkmp
	err := tx.Where("sdm_kdkmp_entry_id = ? AND report_date = ?", *user.SDMKdkmpEntryID, kdkmp.DateString(businessDate)).
		Clauses(lockClause()).
		First(&entry).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if entry.ID == 0 || !entry.HasConfirmedOperationalAttendance() {
		return nil, newReportError(http.StatusUnprocessableEntity, "Simpan kehadiran anggota hari ini terlebih dahulu sebelum memulai task.")
	}

	return &entry, nil
}

func validateAllocations(
	tx *gorm.DB,
	user *models.User,
	businessDate time.Time,
	attendance *models.EbitdamaxKdkmp,
	allocations map[string]int,
	exceptReportID *int64,
) error {
	attendanceValues := map[string]int{}
	if attendance != nil {
		attendanceValues = attendance.OperationalAttendance
	}

	allocated, available, err := AppDeps.Allocation.SummaryForUserWithTx(tx, user, businessDate, attendanceValues, exceptReportID, true)
	if err != nil {
		return err
	}
	_ = allocated

	for key, value := range allocations {
		if value > available[key] {
			label := models.OperationalAttendanceRoleLabels[key]
			return newReportError(
				http.StatusUnprocessableEntity,
				fmt.Sprintf("Jumlah alokasi %s melebihi sisa anggota yang tersedia (Sisa: %d).", label, available[key]),
			)
		}
	}

	return nil
}

func storePhoto(ctx context.Context, reportUUID string, phase string, photo *multipart.FileHeader) (models.StoredDocument, error) {
	documents, err := AppDeps.TaskReports.StoreDocuments(ctx, reportUUID, phase, []*multipart.FileHeader{photo})
	if err != nil {
		return models.StoredDocument{}, err
	}
	if len(documents) == 0 {
		return models.StoredDocument{}, fmt.Errorf("foto gagal disimpan")
	}

	return documents[0], nil
}

// syncTaskReportValues menyimpan nilai field tambahan sesuai fase.
func syncTaskReportValues(
	ctx context.Context,
	tx *gorm.DB,
	report *models.TaskReport,
	task *models.Task,
	form *multipart.Form,
	values map[string]any,
	phase string,
	storedFiles *[]models.StoredDocument,
) error {
	var fields []models.TaskAdditionalField
	err := tx.Where("task_id = ? AND show_when = ?", task.ID, phase).
		Order("sort_order").Order("id").
		Find(&fields).Error
	if err != nil {
		return err
	}

	for _, field := range fields {
		var value *string

		if field.InputType == "file" {
			file := valueFile(form, field.FieldName)
			if file == nil {
				if field.IsRequired {
					return newReportError(http.StatusUnprocessableEntity, fmt.Sprintf("%s wajib diisi.", field.Label))
				}
				continue
			}

			document, err := AppDeps.TaskReports.StoreAdditionalField(ctx, report.UUID, field.UUID, phase, file)
			if err != nil {
				return err
			}
			*storedFiles = append(*storedFiles, document)

			encoded, err := json.Marshal(document)
			if err != nil {
				return err
			}
			serialized := string(encoded)
			value = &serialized
		} else {
			raw, exists := values[field.FieldName]
			if !exists || raw == nil {
				if field.IsRequired {
					return newReportError(http.StatusUnprocessableEntity, fmt.Sprintf("%s wajib diisi.", field.Label))
				}
				continue
			}

			serialized, ok := serializeFieldValue(raw)
			if !ok {
				continue
			}
			if field.IsRequired && strings.TrimSpace(serialized) == "" {
				return newReportError(http.StatusUnprocessableEntity, fmt.Sprintf("%s wajib diisi.", field.Label))
			}
			value = &serialized
		}

		row := models.TaskReportValue{
			UUID:                  uuid.NewString(),
			TaskReportID:          report.ID,
			TaskAdditionalFieldID: field.ID,
			Value:                 value,
		}

		err := tx.Where("task_report_id = ? AND task_additional_field_id = ?", report.ID, field.ID).
			Assign(map[string]any{"value": value, "updated_at": time.Now()}).
			FirstOrCreate(&row).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func serializeFieldValue(raw any) (string, bool) {
	switch typed := raw.(type) {
	case string:
		return typed, true
	case bool:
		if typed {
			return "1", true
		}
		return "0", true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case []any:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "", false
		}
		return string(encoded), true
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "", false
		}
		return string(encoded), true
	}
}

func parseTaskReportForm(c *gin.Context) (*multipart.Form, bool) {
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return nil, false
	}
	if c.Request.MultipartForm == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return nil, false
	}
	return c.Request.MultipartForm, true
}

func validatePhotoUpload(c *gin.Context, form *multipart.Form, field string, requiredMessage string) (*multipart.FileHeader, bool) {
	files := form.File[field]
	if len(files) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": requiredMessage})
		return nil, false
	}

	photo := files[0]
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(photo.Filename)), ".")
	if !allowedPhotoExtensions[extension] {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Format foto harus JPG, PNG, WEBP, atau GIF."})
		return nil, false
	}
	if photo.Size > maxPhotoSize {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Ukuran foto maksimal 3 MB."})
		return nil, false
	}

	return photo, true
}

func validateDocumentUploads(c *gin.Context, documents []*multipart.FileHeader) ([]*multipart.FileHeader, bool) {
	if len(documents) > maxDocumentsPerPhase {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Maksimal 10 dokumen dalam satu kali upload."})
		return nil, false
	}

	for _, document := range documents {
		extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(document.Filename)), ".")
		if !allowedDocumentExtensions[extension] {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Format dokumen tidak didukung."})
			return nil, false
		}
		if document.Size > maxDocumentSize {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Ukuran dokumen maksimal 10 MB per file."})
			return nil, false
		}
	}

	return documents, true
}

func parseValuesJSON(c *gin.Context, raw string) (map[string]any, bool) {
	values := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return values, true
	}

	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Format nilai field tambahan tidak valid."})
		return nil, false
	}

	return values, true
}

func parseMemberAllocationFields(c *gin.Context, isManager bool) (map[string]int, bool, bool) {
	if !isManager {
		if c.PostForm("member_allocations") != "" || c.PostForm("manager_self_assigned") != "" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Alokasi anggota hanya untuk Manager KDKMP."})
			return nil, false, false
		}
		return nil, false, true
	}

	raw := c.PostForm("member_allocations")
	if strings.TrimSpace(raw) == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Alokasi anggota wajib diisi."})
		return nil, false, false
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Format alokasi anggota tidak valid."})
		return nil, false, false
	}

	allocations := make(map[string]int, len(models.OperationalAttendanceRoleKeys))
	for _, role := range models.OperationalAttendanceRoleKeys {
		value, exists := parsed[role]
		number, isNumber := value.(float64)
		if !exists || !isNumber || number < 0 || number != math.Trunc(number) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": fmt.Sprintf("Alokasi %s wajib diisi dengan angka bulat minimal 0.", models.OperationalAttendanceRoleLabels[role]),
			})
			return nil, false, false
		}
		allocations[role] = int(number)
	}

	rawSelf := strings.TrimSpace(c.PostForm("manager_self_assigned"))
	if rawSelf == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Status pengerjaan manager wajib diisi."})
		return nil, false, false
	}
	selfAssigned := rawSelf == "true" || rawSelf == "1"

	total := 0
	for _, value := range allocations {
		total += value
	}

	if !selfAssigned && total == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Alokasikan minimal satu anggota atau centang bahwa Manager KDKMP yang mengerjakan."})
		return nil, false, false
	}
	if selfAssigned && total > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Alokasi anggota harus bernilai 0 saat Manager KDKMP mengerjakan sendiri."})
		return nil, false, false
	}

	return allocations, selfAssigned, true
}

func valueFile(form *multipart.Form, fieldName string) *multipart.FileHeader {
	files := form.File["value_files["+fieldName+"]"]
	if len(files) == 0 {
		return nil
	}
	return files[0]
}

func respondReportError(c *gin.Context, err error) {
	var reportErr reportError
	if errors.As(err, &reportErr) {
		c.JSON(reportErr.status, gin.H{"message": reportErr.message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "Terjadi kesalahan pada server"})
}
