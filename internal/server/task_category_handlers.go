package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/slug"
)

const (
	taskCategorySortDefault    = "name"
	taskCategoryPerPageDefault = 15
)

type taskCategoryPayload struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// ListTaskCategoriesHandler godoc
//
//	@Summary      Daftar kategori task
//	@Description  Daftar kategori task dengan pencarian, urutan, paginasi, dan jumlah task.
//	@Tags         Task Categories
//	@Produce      json
//	@Security     CookieAuth
//	@Param        search     query  string  false  "Cari nama/slug/deskripsi"
//	@Param        sort       query  string  false  "Urutkan: name|created_at"
//	@Param        direction  query  string  false  "asc|desc"
//	@Param        page       query  int     false  "Halaman"
//	@Success      200        {object}  map[string]any
//	@Failure      401        {object}  map[string]string
//	@Failure      403        {object}  map[string]string
//	@Router       /api/v1/task-categories [get]
func ListTaskCategoriesHandler(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	sort := c.DefaultQuery("sort", taskCategorySortDefault)
	direction := "asc"
	if strings.ToLower(c.Query("direction")) == "desc" {
		direction = "desc"
	}

	allowedSorts := map[string]bool{"name": true, "created_at": true}
	if !allowedSorts[sort] {
		sort = taskCategorySortDefault
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	query := AppDeps.DB.WithContext(c.Request.Context()).
		Model(&models.TaskCategory{}).
		Select("task_categories.*, (SELECT COUNT(*) FROM tasks WHERE tasks.task_category_id = task_categories.id) AS tasks_count")

	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(task_categories.name) LIKE ? OR LOWER(task_categories.slug) LIKE ? OR LOWER(task_categories.description) LIKE ?",
			pattern, pattern, pattern,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat kategori task"})
		return
	}

	type categoryRow struct {
		models.TaskCategory
		TasksCount int64 `gorm:"column:tasks_count" json:"tasks_count"`
	}

	var rows []categoryRow
	err := query.
		Order("task_categories." + sort + " " + direction).
		Order("task_categories.id").
		Offset((page - 1) * taskCategoryPerPageDefault).
		Limit(taskCategoryPerPageDefault).
		Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat kategori task"})
		return
	}

	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, taskCategoryResponse(&row.TaskCategory, row.TasksCount))
	}

	totalPages := int((total + int64(taskCategoryPerPageDefault) - 1) / int64(taskCategoryPerPageDefault))

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":        page,
			"per_page":    taskCategoryPerPageDefault,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// CreateTaskCategoryHandler godoc
//
//	@Summary      Tambah kategori task
//	@Description  Membuat kategori task (slug otomatis unik dari nama).
//	@Tags         Task Categories
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      taskCategoryPayload  true  "Data kategori"
//	@Success      201      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/task-categories [post]
func CreateTaskCategoryHandler(c *gin.Context) {
	var req taskCategoryPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	category, ok := validateTaskCategoryPayload(c, &req, 0)
	if !ok {
		return
	}

	category.UUID = uuid.NewString()

	if err := AppDeps.DB.WithContext(c.Request.Context()).Create(category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan kategori task"})
		return
	}

	c.JSON(http.StatusCreated, taskCategoryResponse(category, 0))
}

// UpdateTaskCategoryHandler godoc
//
//	@Summary      Ubah kategori task
//	@Description  Mengubah nama dan deskripsi kategori; slug ikut diperbarui.
//	@Tags         Task Categories
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id       path      int                  true  "ID kategori"
//	@Param        payload  body      taskCategoryPayload  true  "Data kategori"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      404      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/task-categories/{id} [put]
func UpdateTaskCategoryHandler(c *gin.Context) {
	category, ok := findTaskCategory(c)
	if !ok {
		return
	}

	var req taskCategoryPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	updated, ok := validateTaskCategoryPayload(c, &req, category.ID)
	if !ok {
		return
	}

	category.Name = updated.Name
	category.Slug = updated.Slug
	category.Description = updated.Description

	if err := AppDeps.DB.WithContext(c.Request.Context()).Save(category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui kategori task"})
		return
	}

	var tasksCount int64
	_ = AppDeps.DB.Table("tasks").Where("task_category_id = ?", category.ID).Count(&tasksCount).Error

	c.JSON(http.StatusOK, taskCategoryResponse(category, tasksCount))
}

// DeleteTaskCategoryHandler godoc
//
//	@Summary      Hapus kategori task
//	@Description  Menghapus kategori; ditolak bila masih dipakai task.
//	@Tags         Task Categories
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id  path  int  true  "ID kategori"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Failure      409  {object}  map[string]string
//	@Router       /api/v1/task-categories/{id} [delete]
func DeleteTaskCategoryHandler(c *gin.Context) {
	category, ok := findTaskCategory(c)
	if !ok {
		return
	}

	var tasksCount int64
	if err := AppDeps.DB.Table("tasks").Where("task_category_id = ?", category.ID).Count(&tasksCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa pemakaian kategori"})
		return
	}
	if tasksCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "Kategori task tidak dapat dihapus karena sudah digunakan oleh task."})
		return
	}

	if err := AppDeps.DB.WithContext(c.Request.Context()).Delete(category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghapus kategori task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kategori task berhasil dihapus."})
}

func validateTaskCategoryPayload(c *gin.Context, req *taskCategoryPayload, excludeID int64) (*models.TaskCategory, bool) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama kategori wajib diisi"})
		return nil, false
	}
	if len([]rune(name)) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama kategori maksimal 255 karakter"})
		return nil, false
	}

	var description *string
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		if trimmed != "" {
			description = &trimmed
		}
	}

	db := AppDeps.DB.WithContext(c.Request.Context())

	var duplicate int64
	query := db.Model(&models.TaskCategory{}).Where("LOWER(name) = ?", strings.ToLower(name))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&duplicate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa nama kategori"})
		return nil, false
	}
	if duplicate > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama kategori sudah digunakan"})
		return nil, false
	}

	categorySlug, err := slug.Unique(db, "task_categories", slug.Make(name), excludeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat slug kategori"})
		return nil, false
	}

	return &models.TaskCategory{
		Name:        name,
		Slug:        categorySlug,
		Description: description,
	}, true
}

func findTaskCategory(c *gin.Context) (*models.TaskCategory, bool) {
	categoryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Kategori task tidak ditemukan"})
		return nil, false
	}

	var category models.TaskCategory
	err = AppDeps.DB.WithContext(c.Request.Context()).First(&category, categoryID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Kategori task tidak ditemukan"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat kategori task"})
		return nil, false
	}

	return &category, true
}

func taskCategoryResponse(category *models.TaskCategory, tasksCount int64) gin.H {
	return gin.H{
		"id":          category.ID,
		"uuid":        category.UUID,
		"name":        category.Name,
		"slug":        category.Slug,
		"description": category.Description,
		"tasks_count": tasksCount,
		"created_at":  category.CreatedAt,
		"updated_at":  category.UpdatedAt,
	}
}
