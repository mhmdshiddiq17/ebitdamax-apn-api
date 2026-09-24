package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/slug"
)

const (
	taskPerPageDefault = 15
	taskSortDefault    = "sort_order"
)

type costPayload struct {
	Man      *int `json:"man"`
	Machine  *int `json:"machine"`
	Method   *int `json:"method"`
	Material *int `json:"material"`
}

type additionalFieldPayload struct {
	ID         *int64   `json:"id"`
	Label      string   `json:"label"`
	InputType  string   `json:"input_type"`
	ShowWhen   string   `json:"show_when"`
	IsRequired *bool    `json:"is_required"`
	Options    []string `json:"options"`
}

type taskPayload struct {
	TaskCategoryID   int64                    `json:"task_category_id"`
	BMCStatus        string                   `json:"bmc_status"`
	RoleIDs          []int64                  `json:"role_ids"`
	SortOrder        *int                     `json:"sort_order"`
	Name             string                   `json:"name"`
	Description      *string                  `json:"description"`
	ExecutionTime    *string                  `json:"execution_time"`
	TimeRequire      *int                     `json:"time_require"`
	LowerThreshold   *int                     `json:"lower_time_threshold_minutes"`
	UpperThreshold   *int                     `json:"upper_time_threshold_minutes"`
	Period           string                   `json:"period"`
	IsActive         *bool                    `json:"is_active"`
	IsMandatory      *bool                    `json:"is_mandatory"`
	FixedCost        *costPayload             `json:"fixed_cost"`
	VariableCost     *costPayload             `json:"variable_cost"`
	AdditionalFields []additionalFieldPayload `json:"additional_fields"`
}

// ListTasksHandler godoc
//
//	@Summary      Daftar task
//	@Description  Daftar task dengan pencarian, filter kategori/role/status, urutan, dan paginasi.
//	@Tags         Tasks
//	@Produce      json
//	@Security     CookieAuth
//	@Param        search            query  string  false  "Cari nama/deskripsi/kategori/role"
//	@Param        task_category_id  query  int     false  "Filter kategori"
//	@Param        role_id           query  int     false  "Filter role"
//	@Param        status            query  string  false  "active|inactive|all"
//	@Param        sort              query  string  false  "Urutkan: sort_order|name|execution_time|time_require|created_at"
//	@Param        direction         query  string  false  "asc|desc"
//	@Param        page              query  int     false  "Halaman"
//	@Success      200               {object}  map[string]any
//	@Failure      401               {object}  map[string]string
//	@Failure      403               {object}  map[string]string
//	@Router       /api/v1/tasks [get]
func ListTasksHandler(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	status := c.DefaultQuery("status", "active")
	sort := c.DefaultQuery("sort", taskSortDefault)
	direction := "asc"
	if strings.ToLower(c.Query("direction")) == "desc" {
		direction = "desc"
	}

	allowedSorts := map[string]bool{
		"sort_order": true, "name": true, "execution_time": true, "time_require": true, "created_at": true,
	}
	if !allowedSorts[sort] {
		sort = taskSortDefault
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	query := AppDeps.DB.WithContext(c.Request.Context()).Model(&models.Task{})

	if categoryID, err := strconv.ParseInt(c.Query("task_category_id"), 10, 64); err == nil && categoryID > 0 {
		query = query.Where("tasks.task_category_id = ?", categoryID)
	}

	if roleID, err := strconv.ParseInt(c.Query("role_id"), 10, 64); err == nil && roleID > 0 {
		query = query.Where("EXISTS (SELECT 1 FROM task_roles tr WHERE tr.task_id = tasks.id AND tr.role_id = ?)", roleID)
	}

	if status == "active" {
		query = query.Where("tasks.is_active = ?", true)
	} else if status == "inactive" {
		query = query.Where("tasks.is_active = ?", false)
	}

	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			`(LOWER(tasks.name) LIKE ? OR LOWER(tasks.description) LIKE ?
			  OR EXISTS (SELECT 1 FROM task_categories tc WHERE tc.id = tasks.task_category_id AND LOWER(tc.name) LIKE ?)
			  OR EXISTS (SELECT 1 FROM task_roles tr JOIN roles r ON r.id = tr.role_id WHERE tr.task_id = tasks.id AND LOWER(r.name) LIKE ?))`,
			pattern, pattern, pattern, pattern,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat task"})
		return
	}

	if sort == "sort_order" {
		query = query.
			Order("CASE WHEN tasks.sort_order IS NULL THEN 1 ELSE 0 END").
			Order("tasks.sort_order " + direction).
			Order("tasks.id")
	} else {
		query = query.Order("tasks." + sort + " " + direction).Order("tasks.id")
	}

	var tasks []models.Task
	err := query.
		Preload("TaskCategory").
		Preload("Roles").
		Preload("AdditionalFields", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order").Order("id")
		}).
		Offset((page - 1) * taskPerPageDefault).
		Limit(taskPerPageDefault).
		Find(&tasks).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat task"})
		return
	}

	data := make([]gin.H, 0, len(tasks))
	for i := range tasks {
		data = append(data, transformTask(&tasks[i]))
	}

	totalPages := int((total + int64(taskPerPageDefault) - 1) / int64(taskPerPageDefault))

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":        page,
			"per_page":    taskPerPageDefault,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// CreateTaskHandler godoc
//
//	@Summary      Tambah task
//	@Description  Membuat task dengan role, biaya fixed/variable, dan field tambahan dinamis.
//	@Tags         Tasks
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      taskPayload  true  "Data task"
//	@Success      201      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/tasks [post]
func CreateTaskHandler(c *gin.Context) {
	var req taskPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	task, fields, ok := validateTaskPayload(c, &req, 0)
	if !ok {
		return
	}

	task.UUID = uuid.NewString()

	err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		if err := syncTaskRoles(tx, task.ID, req.RoleIDs); err != nil {
			return err
		}
		return syncTaskAdditionalFields(tx, task.ID, fields)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan task"})
		return
	}

	reloaded := reloadTask(c, task.ID)
	if reloaded == nil {
		c.JSON(http.StatusCreated, transformTask(task))
		return
	}

	c.JSON(http.StatusCreated, transformTask(reloaded))
}

// UpdateTaskHandler godoc
//
//	@Summary      Ubah task
//	@Description  Mengubah task beserta role, biaya, dan field tambahan (field yang dihapus akan dihilangkan).
//	@Tags         Tasks
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id       path      int          true  "ID task"
//	@Param        payload  body      taskPayload  true  "Data task"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      404      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/tasks/{id} [put]
func UpdateTaskHandler(c *gin.Context) {
	task, ok := findTask(c)
	if !ok {
		return
	}

	var req taskPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	validated, fields, ok := validateTaskPayload(c, &req, task.ID)
	if !ok {
		return
	}

	task.TaskCategoryID = validated.TaskCategoryID
	task.BMCStatus = validated.BMCStatus
	task.SortOrder = validated.SortOrder
	task.Name = validated.Name
	task.Description = validated.Description
	task.ExecutionTime = validated.ExecutionTime
	task.TimeRequire = validated.TimeRequire
	task.LowerThreshold = validated.LowerThreshold
	task.UpperThreshold = validated.UpperThreshold
	task.Period = validated.Period
	task.IsActive = validated.IsActive
	task.IsMandatory = validated.IsMandatory
	task.FixedCost = validated.FixedCost
	task.VariableCost = validated.VariableCost

	err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(task).Error; err != nil {
			return err
		}
		if err := syncTaskRoles(tx, task.ID, req.RoleIDs); err != nil {
			return err
		}
		return syncTaskAdditionalFields(tx, task.ID, fields)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui task"})
		return
	}

	reloaded := reloadTask(c, task.ID)
	if reloaded == nil {
		c.JSON(http.StatusOK, transformTask(task))
		return
	}

	c.JSON(http.StatusOK, transformTask(reloaded))
}

// DeleteTaskHandler godoc
//
//	@Summary      Hapus task
//	@Description  Menghapus task; ditolak bila sudah memiliki laporan.
//	@Tags         Tasks
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id  path  int  true  "ID task"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Failure      409  {object}  map[string]string
//	@Router       /api/v1/tasks/{id} [delete]
func DeleteTaskHandler(c *gin.Context) {
	task, ok := findTask(c)
	if !ok {
		return
	}

	var reports int64
	if err := AppDeps.DB.Table("task_reports").Where("task_id = ?", task.ID).Count(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa laporan task"})
		return
	}
	if reports > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "Task tidak dapat dihapus karena sudah memiliki laporan."})
		return
	}

	if err := AppDeps.DB.WithContext(c.Request.Context()).Delete(task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghapus task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task berhasil dihapus."})
}

func validateTaskPayload(c *gin.Context, req *taskPayload, excludeTaskID int64) (*models.Task, []additionalFieldPayload, bool) {
	db := AppDeps.DB.WithContext(c.Request.Context())

	// Kategori
	var category int64
	if err := db.Model(&models.TaskCategory{}).Where("id = ?", req.TaskCategoryID).Count(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa kategori"})
		return nil, nil, false
	}
	if category == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kategori task tidak valid"})
		return nil, nil, false
	}

	// Poin BMC
	if !models.OptionValueValid(models.TaskBmcStatusOptions, req.BMCStatus) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Poin BMC tidak valid"})
		return nil, nil, false
	}

	// Roles
	if len(req.RoleIDs) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Minimal satu role wajib dipilih"})
		return nil, nil, false
	}
	seenRoles := make(map[int64]bool, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		if seenRoles[roleID] {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Role tidak boleh duplikat"})
			return nil, nil, false
		}
		seenRoles[roleID] = true
	}
	var roleCount int64
	if err := db.Model(&models.Role{}).Where("id IN ?", req.RoleIDs).Count(&roleCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa role"})
		return nil, nil, false
	}
	if int(roleCount) != len(req.RoleIDs) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Role tidak valid"})
		return nil, nil, false
	}

	// Nomor urut
	if req.SortOrder != nil {
		if *req.SortOrder < 1 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nomor urut minimal 1"})
			return nil, nil, false
		}
		query := db.Model(&models.Task{}).Where("sort_order = ?", *req.SortOrder)
		if excludeTaskID > 0 {
			query = query.Where("id <> ?", excludeTaskID)
		}
		var duplicate int64
		if err := query.Count(&duplicate).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa nomor urut"})
			return nil, nil, false
		}
		if duplicate > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nomor urut sudah digunakan"})
			return nil, nil, false
		}
	}

	// Info dasar
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama task wajib diisi"})
		return nil, nil, false
	}
	if len([]rune(name)) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama task maksimal 255 karakter"})
		return nil, nil, false
	}

	var description *string
	if req.Description != nil {
		if trimmed := strings.TrimSpace(*req.Description); trimmed != "" {
			description = &trimmed
		}
	}

	var executionTime *models.ClockTime
	if req.ExecutionTime != nil && strings.TrimSpace(*req.ExecutionTime) != "" {
		parsed, err := time.Parse("15:04", strings.TrimSpace(*req.ExecutionTime))
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Format jam pelaksanaan tidak valid (HH:MM)"})
			return nil, nil, false
		}
		executionTime = &models.ClockTime{Hour: parsed.Hour(), Minute: parsed.Minute(), Valid: true}
	}

	if req.TimeRequire == nil || *req.TimeRequire < 1 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Estimasi waktu wajib diisi minimal 1 menit"})
		return nil, nil, false
	}

	// Ambang waktu
	if (req.LowerThreshold == nil) != (req.UpperThreshold == nil) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Ambang waktu bawah dan atas harus diisi bersama"})
		return nil, nil, false
	}
	if req.LowerThreshold != nil && req.UpperThreshold != nil {
		if *req.LowerThreshold < 0 || *req.UpperThreshold < 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Ambang waktu tidak boleh negatif"})
			return nil, nil, false
		}
		if *req.LowerThreshold > *req.UpperThreshold {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Ambang waktu bawah tidak boleh lebih besar dari ambang atas"})
			return nil, nil, false
		}
	}

	// Periode
	if !models.OptionValueValid(models.TaskPeriodOptions, req.Period) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Periode task tidak valid"})
		return nil, nil, false
	}

	if req.IsActive == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Status aktif wajib diisi"})
		return nil, nil, false
	}
	if req.IsMandatory == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Status task wajib wajib diisi"})
		return nil, nil, false
	}

	// Biaya
	fixedCost, ok := validateCostPayload(c, req.FixedCost, "Fixed cost")
	if !ok {
		return nil, nil, false
	}
	variableCost, ok := validateCostPayload(c, req.VariableCost, "Variable cost")
	if !ok {
		return nil, nil, false
	}

	// Field tambahan
	fields, ok := validateAdditionalFields(c, req.AdditionalFields)
	if !ok {
		return nil, nil, false
	}

	task := &models.Task{
		TaskCategoryID: req.TaskCategoryID,
		BMCStatus:      req.BMCStatus,
		SortOrder:      req.SortOrder,
		Name:           name,
		Description:    description,
		ExecutionTime:  executionTime,
		TimeRequire:    *req.TimeRequire,
		LowerThreshold: req.LowerThreshold,
		UpperThreshold: req.UpperThreshold,
		Period:         req.Period,
		IsActive:       *req.IsActive,
		IsMandatory:    *req.IsMandatory,
		FixedCost:      models.ConfiguredFixedCostBreakdown(fixedCost),
		VariableCost:   models.NormalizeCostBreakdown(variableCost),
	}

	return task, fields, true
}

func validateCostPayload(c *gin.Context, payload *costPayload, label string) (models.CostBreakdown, bool) {
	if payload == nil || payload.Man == nil || payload.Machine == nil || payload.Method == nil || payload.Material == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": label + " wajib diisi lengkap (man, machine, method, material)"})
		return nil, false
	}

	values := map[string]int{
		"man":      *payload.Man,
		"machine":  *payload.Machine,
		"method":   *payload.Method,
		"material": *payload.Material,
	}

	for component, value := range values {
		if value < 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": label + " " + component + " tidak boleh negatif"})
			return nil, false
		}
	}

	return models.CostBreakdown(values), true
}

func validateAdditionalFields(c *gin.Context, fields []additionalFieldPayload) ([]additionalFieldPayload, bool) {
	for index := range fields {
		field := &fields[index]

		field.Label = strings.TrimSpace(field.Label)
		if field.Label == "" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Label field tambahan wajib diisi"})
			return nil, false
		}
		if len([]rune(field.Label)) > 255 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Label field tambahan maksimal 255 karakter"})
			return nil, false
		}

		if !models.OptionValueValid(models.TaskInputTypeOptions, field.InputType) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Tipe input field tambahan tidak valid"})
			return nil, false
		}
		if !models.OptionValueValid(models.TaskShowWhenOptions, field.ShowWhen) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Waktu tampil field tambahan tidak valid"})
			return nil, false
		}
		if field.IsRequired == nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Status wajib field tambahan harus diisi"})
			return nil, false
		}

		for _, option := range field.Options {
			if len([]rune(strings.TrimSpace(option))) > 255 {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Opsi field tambahan maksimal 255 karakter"})
				return nil, false
			}
		}
	}

	return fields, true
}

func syncTaskRoles(tx *gorm.DB, taskID int64, roleIDs []int64) error {
	if err := tx.Exec("DELETE FROM task_roles WHERE task_id = ?", taskID).Error; err != nil {
		return err
	}

	for _, roleID := range roleIDs {
		if err := tx.Exec(
			"INSERT INTO task_roles (task_id, role_id, created_at, updated_at) VALUES (?, ?, now(), now())",
			taskID, roleID,
		).Error; err != nil {
			return err
		}
	}

	return nil
}

func syncTaskAdditionalFields(tx *gorm.DB, taskID int64, fields []additionalFieldPayload) error {
	incomingIDs := make([]int64, 0, len(fields))
	for _, field := range fields {
		if field.ID != nil && *field.ID > 0 {
			incomingIDs = append(incomingIDs, *field.ID)
		}
	}

	del := tx.Where("task_id = ?", taskID)
	if len(incomingIDs) > 0 {
		del = del.Where("id NOT IN ?", incomingIDs)
	}
	if err := del.Delete(&models.TaskAdditionalField{}).Error; err != nil {
		return err
	}

	for index, field := range fields {
		fieldName, err := uniqueTaskFieldName(tx, taskID, field.Label, derefInt64(field.ID))
		if err != nil {
			return err
		}

		data := map[string]any{
			"task_id":     taskID,
			"label":       field.Label,
			"field_name":  fieldName,
			"input_type":  field.InputType,
			"show_when":   field.ShowWhen,
			"is_required": *field.IsRequired,
			"sort_order":  index,
			"options":     taskFieldOptions(field),
			"updated_at":  time.Now(),
		}

		if field.ID != nil && *field.ID > 0 {
			result := tx.Model(&models.TaskAdditionalField{}).
				Where("id = ? AND task_id = ?", *field.ID, taskID).
				Updates(data)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return createTaskAdditionalField(tx, data)
			}
			continue
		}

		if err := createTaskAdditionalField(tx, data); err != nil {
			return err
		}
	}

	return nil
}

func createTaskAdditionalField(tx *gorm.DB, data map[string]any) error {
	data["uuid"] = uuid.NewString()
	data["created_at"] = time.Now()
	return tx.Table("task_additional_fields").Create(data).Error
}

func uniqueTaskFieldName(tx *gorm.DB, taskID int64, label string, ignoreFieldID int64) (string, error) {
	base := slug.MakeSeparator(label, "_")
	if base == "" {
		base = "field_" + strings.ToLower(uuid.NewString()[:8])
	}

	candidate := base
	for suffix := 2; ; suffix++ {
		query := tx.Table("task_additional_fields").
			Where("task_id = ? AND field_name = ?", taskID, candidate)
		if ignoreFieldID > 0 {
			query = query.Where("id <> ?", ignoreFieldID)
		}

		var count int64
		if err := query.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}

		candidate = base + "_" + strconv.Itoa(suffix)
	}
}

func taskFieldOptions(field additionalFieldPayload) models.StringList {
	if field.InputType != "select" && field.InputType != "radio" && field.InputType != "checkbox" {
		return nil
	}

	options := make(models.StringList, 0, len(field.Options))
	for _, option := range field.Options {
		if trimmed := strings.TrimSpace(option); trimmed != "" {
			options = append(options, trimmed)
		}
	}

	if len(options) == 0 {
		return nil
	}
	return options
}

func findTask(c *gin.Context) (*models.Task, bool) {
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Task tidak ditemukan"})
		return nil, false
	}

	var task models.Task
	err = AppDeps.DB.WithContext(c.Request.Context()).First(&task, taskID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Task tidak ditemukan"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat task"})
		return nil, false
	}

	return &task, true
}

func reloadTask(c *gin.Context, taskID int64) *models.Task {
	var task models.Task
	err := AppDeps.DB.WithContext(c.Request.Context()).
		Preload("TaskCategory").
		Preload("Roles").
		Preload("AdditionalFields", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order").Order("id")
		}).
		First(&task, taskID).Error
	if err != nil {
		return nil
	}
	return &task
}

func transformTask(task *models.Task) gin.H {
	fixedCost := task.FixedCostBreakdown()
	variableCost := task.VariableCostBreakdown()
	firstRole := (*models.Role)(nil)
	if len(task.Roles) > 0 {
		firstRole = &task.Roles[0]
	}

	response := gin.H{
		"id":                           task.ID,
		"uuid":                         task.UUID,
		"task_category_id":             task.TaskCategoryID,
		"bmc_status":                   task.BMCStatus,
		"bmc_status_label":             models.OptionLabel(models.TaskBmcStatusOptions, task.BMCStatus),
		"sort_order":                   task.SortOrder,
		"name":                         task.Name,
		"description":                  task.Description,
		"execution_time":               task.ExecutionTime.String(),
		"time_require":                 task.TimeRequire,
		"lower_time_threshold_minutes": task.LowerThreshold,
		"upper_time_threshold_minutes": task.UpperThreshold,
		"period":                       task.Period,
		"period_label":                 models.OptionLabel(models.TaskPeriodOptions, task.Period),
		"is_active":                    task.IsActive,
		"is_mandatory":                 task.IsMandatory,
		"fixed_cost":                   fixedCost,
		"fixed_cost_total":             models.CostTotal(fixedCost),
		"variable_cost":                variableCost,
		"variable_cost_total":          models.CostTotal(variableCost),
		"role_id":                      nil,
		"role_ids":                     roleIDs(task.Roles),
		"created_at":                   task.CreatedAt,
		"updated_at":                   task.UpdatedAt,
	}

	if firstRole != nil {
		response["role_id"] = firstRole.ID
		response["role"] = roleSummary(*firstRole)
	} else {
		response["role"] = nil
	}

	roles := make([]gin.H, 0, len(task.Roles))
	for _, role := range task.Roles {
		roles = append(roles, roleSummary(role))
	}
	response["roles"] = roles

	if task.TaskCategory != nil {
		response["task_category"] = gin.H{
			"id":   task.TaskCategory.ID,
			"name": task.TaskCategory.Name,
			"slug": task.TaskCategory.Slug,
		}
	} else {
		response["task_category"] = nil
	}

	fields := make([]gin.H, 0, len(task.AdditionalFields))
	for _, field := range task.AdditionalFields {
		options := field.Options
		if options == nil {
			options = models.StringList{}
		}
		fields = append(fields, gin.H{
			"id":               field.ID,
			"uuid":             field.UUID,
			"label":            field.Label,
			"field_name":       field.FieldName,
			"input_type":       field.InputType,
			"input_type_label": models.OptionLabel(models.TaskInputTypeOptions, field.InputType),
			"show_when":        field.ShowWhen,
			"show_when_label":  models.OptionLabel(models.TaskShowWhenOptions, field.ShowWhen),
			"is_required":      field.IsRequired,
			"sort_order":       field.SortOrder,
			"options":          options,
		})
	}
	response["additional_fields"] = fields

	return response
}

func roleSummary(role models.Role) gin.H {
	return gin.H{
		"id":          role.ID,
		"name":        role.Name,
		"slug":        role.Slug,
		"level":       role.Level,
		"level_label": models.RoleLevelLabel(role.Level),
	}
}

func roleIDs(roles []models.Role) []int64 {
	ids := make([]int64, 0, len(roles))
	for _, role := range roles {
		ids = append(ids, role.ID)
	}
	return ids
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
