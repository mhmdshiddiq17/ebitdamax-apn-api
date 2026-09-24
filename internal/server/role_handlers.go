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
	roleSortDefault    = "name"
	rolePerPageDefault = 15
)

type rolePayload struct {
	Domain string `json:"domain"`
	Name   string `json:"name"`
	Level  string `json:"level"`
}

type roleListRow struct {
	models.Role
	UsersCount int64 `gorm:"column:users_count" json:"users_count"`
}

var validRoleLevels = map[string]bool{
	models.RoleLevelStaff:      true,
	models.RoleLevelManager:    true,
	models.RoleLevelSuperadmin: true,
}

var validRoleDomains = map[string]bool{
	models.RoleDomainApn:   true,
	models.RoleDomainKdkmp: true,
}

// ListRolesHandler godoc
//
//	@Summary      Daftar role
//	@Description  Daftar role per domain (default kdkmp) dengan pencarian, urutan, dan paginasi.
//	@Tags         Roles
//	@Produce      json
//	@Security     CookieAuth
//	@Param        domain     query  string  false  "Domain (apn|kdkmp)"
//	@Param        search     query  string  false  "Cari nama/slug/level"
//	@Param        sort       query  string  false  "Urutkan: name|level|created_at"
//	@Param        direction  query  string  false  "asc|desc"
//	@Param        page       query  int     false  "Halaman"
//	@Success      200        {object}  map[string]any
//	@Failure      401        {object}  map[string]string
//	@Failure      403        {object}  map[string]string
//	@Router       /api/v1/roles [get]
func ListRolesHandler(c *gin.Context) {
	domain := domainFromQuery(c, models.RoleDomainKdkmp)

	search := strings.TrimSpace(c.Query("search"))
	sort := c.DefaultQuery("sort", roleSortDefault)
	direction := "asc"
	if strings.ToLower(c.Query("direction")) == "desc" {
		direction = "desc"
	}

	allowedSorts := map[string]bool{"name": true, "level": true, "created_at": true}
	if !allowedSorts[sort] {
		sort = roleSortDefault
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	query := AppDeps.DB.WithContext(c.Request.Context()).
		Model(&models.Role{}).
		Select("roles.*, (SELECT COUNT(*) FROM users WHERE users.role_id = roles.id) AS users_count").
		Where("roles.domain = ?", domain)

	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(roles.name) LIKE ? OR LOWER(roles.slug) LIKE ? OR LOWER(roles.level) LIKE ?",
			pattern, pattern, pattern,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat role"})
		return
	}

	var rows []roleListRow
	err := query.
		Order("roles." + sort + " " + direction).
		Order("roles.id").
		Offset((page - 1) * rolePerPageDefault).
		Limit(rolePerPageDefault).
		Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat role"})
		return
	}

	data := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		data = append(data, roleResponse(&row.Role, row.UsersCount))
	}

	totalPages := int((total + int64(rolePerPageDefault) - 1) / int64(rolePerPageDefault))

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":        page,
			"per_page":    rolePerPageDefault,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// CreateRoleHandler godoc
//
//	@Summary      Tambah role
//	@Description  Membuat role baru (slug dibuat otomatis dari nama, dijamin unik).
//	@Tags         Roles
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      rolePayload  true  "Data role"
//	@Success      201      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/roles [post]
func CreateRoleHandler(c *gin.Context) {
	var req rolePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	role, ok := validateRolePayload(c, req, 0)
	if !ok {
		return
	}

	role.UUID = uuid.NewString()

	if err := AppDeps.DB.WithContext(c.Request.Context()).Create(role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan role"})
		return
	}

	c.JSON(http.StatusCreated, roleResponse(role, 0))
}

// UpdateRoleHandler godoc
//
//	@Summary      Ubah role
//	@Description  Mengubah nama dan level role; slug ikut diperbarui.
//	@Tags         Roles
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id       path      int          true  "ID role"
//	@Param        payload  body      rolePayload  true  "Data role"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      404      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/roles/{id} [put]
func UpdateRoleHandler(c *gin.Context) {
	role, ok := findRole(c)
	if !ok {
		return
	}

	var req rolePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	updated, ok := validateRolePayload(c, req, role.ID)
	if !ok {
		return
	}

	role.Name = updated.Name
	role.Slug = updated.Slug
	role.Level = updated.Level
	role.Domain = updated.Domain

	if err := AppDeps.DB.WithContext(c.Request.Context()).Save(role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui role"})
		return
	}

	var usersCount int64
	_ = AppDeps.DB.Model(&models.User{}).Where("role_id = ?", role.ID).Count(&usersCount).Error

	c.JSON(http.StatusOK, roleResponse(role, usersCount))
}

// DeleteRoleHandler godoc
//
//	@Summary      Hapus role
//	@Description  Menghapus role; ditolak bila masih dipakai user atau task.
//	@Tags         Roles
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id  path  int  true  "ID role"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Failure      409  {object}  map[string]string
//	@Router       /api/v1/roles/{id} [delete]
func DeleteRoleHandler(c *gin.Context) {
	role, ok := findRole(c)
	if !ok {
		return
	}

	var usersCount int64
	if err := AppDeps.DB.Model(&models.User{}).Where("role_id = ?", role.ID).Count(&usersCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa pemakaian role"})
		return
	}
	if usersCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "Role tidak dapat dihapus karena sudah digunakan oleh user."})
		return
	}

	var tasksCount int64
	if err := AppDeps.DB.Table("task_roles").Where("role_id = ?", role.ID).Count(&tasksCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa pemakaian role"})
		return
	}
	if tasksCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"message": "Role tidak dapat dihapus karena sudah digunakan oleh task."})
		return
	}

	if err := AppDeps.DB.WithContext(c.Request.Context()).Delete(role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghapus role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role berhasil dihapus."})
}

// validateRolePayload memvalidasi payload dan mengembalikan role siap simpan
// (name unik, slug unik, level & domain valid). excludeID > 0 saat update.
func validateRolePayload(c *gin.Context, req rolePayload, excludeID int64) (*models.Role, bool) {
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		domain = models.RoleDomainKdkmp
	}
	if !validRoleDomains[domain] {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Domain role tidak valid"})
		return nil, false
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama role wajib diisi"})
		return nil, false
	}
	if len([]rune(name)) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama role maksimal 255 karakter"})
		return nil, false
	}

	level := strings.ToLower(strings.TrimSpace(req.Level))
	if !validRoleLevels[level] {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Level role tidak valid"})
		return nil, false
	}

	db := AppDeps.DB.WithContext(c.Request.Context())

	var duplicate int64
	query := db.Model(&models.Role{}).Where("LOWER(name) = ?", strings.ToLower(name))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&duplicate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa nama role"})
		return nil, false
	}
	if duplicate > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama role sudah digunakan"})
		return nil, false
	}

	roleSlug, err := slug.Unique(db, "roles", slug.Make(name), excludeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat slug role"})
		return nil, false
	}

	return &models.Role{
		Name:   name,
		Slug:   roleSlug,
		Level:  level,
		Domain: domain,
	}, true
}

func findRole(c *gin.Context) (*models.Role, bool) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Role tidak ditemukan"})
		return nil, false
	}

	var role models.Role
	err = AppDeps.DB.WithContext(c.Request.Context()).First(&role, roleID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Role tidak ditemukan"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat role"})
		return nil, false
	}

	return &role, true
}

func roleResponse(role *models.Role, usersCount int64) gin.H {
	return gin.H{
		"id":          role.ID,
		"uuid":        role.UUID,
		"domain":      role.Domain,
		"name":        role.Name,
		"slug":        role.Slug,
		"level":       role.Level,
		"level_label": models.RoleLevelLabel(role.Level),
		"users_count": usersCount,
		"created_at":  role.CreatedAt,
		"updated_at":  role.UpdatedAt,
	}
}

func domainFromQuery(c *gin.Context, fallback string) string {
	domain := strings.ToLower(strings.TrimSpace(c.Query("domain")))
	if !validRoleDomains[domain] {
		return fallback
	}
	return domain
}
