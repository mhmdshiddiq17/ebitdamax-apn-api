package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/slug"
)

const (
	userPerPageDefault   = 15
	maxRegionalAssign    = 25
	maxSKDocumentSize    = 10 << 20 // 10 MB
	skDocumentObjectPath = "manager-sk"
)

type regionalAssignmentPayload struct {
	ScopeLevel    string  `json:"scope_level"`
	Provinsi      string  `json:"provinsi"`
	KotaKabupaten *string `json:"kota_kabupaten"`
	Kecamatan     *string `json:"kecamatan"`
}

type userPayload struct {
	Domain               string                      `json:"domain"`
	RoleID               int64                       `json:"role_id"`
	Name                 string                      `json:"name"`
	Email                string                      `json:"email"`
	Password             string                      `json:"password"`
	PasswordConfirmation string                      `json:"password_confirmation"`
	SDMKdkmpEntryID      *int64                      `json:"sdm_kdkmp_entry_id"`
	RegionalAssignments  []regionalAssignmentPayload `json:"regional_assignments"`
}

type skDocumentMetadata struct {
	Disk         string `json:"disk"`
	Path         string `json:"path"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
	UploadedBy   int64  `json:"uploaded_by"`
	UploadedAt   string `json:"uploaded_at"`
}

// ListUsersHandler godoc
//
//	@Summary      Daftar user
//	@Description  Daftar user domain KDKMP (superadmin) dengan pencarian, filter role, urutan, dan paginasi.
//	@Tags         Users
//	@Produce      json
//	@Security     CookieAuth
//	@Param        search     query  string  false  "Cari nama/username/email"
//	@Param        role_id    query  int     false  "Filter role"
//	@Param        sort       query  string  false  "Urutkan: name|email|created_at"
//	@Param        direction  query  string  false  "asc|desc"
//	@Param        page       query  int     false  "Halaman"
//	@Success      200        {object}  map[string]any
//	@Failure      401        {object}  map[string]string
//	@Failure      403        {object}  map[string]string
//	@Router       /api/v1/users [get]
func ListUsersHandler(c *gin.Context) {
	domain := domainFromQuery(c, models.RoleDomainKdkmp)
	search := strings.TrimSpace(c.Query("search"))
	sort := c.DefaultQuery("sort", "name")
	direction := "asc"
	if strings.ToLower(c.Query("direction")) == "desc" {
		direction = "desc"
	}

	allowedSorts := map[string]bool{"name": true, "email": true, "created_at": true}
	if !allowedSorts[sort] {
		sort = "name"
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	query := AppDeps.DB.WithContext(c.Request.Context()).
		Model(&models.User{}).
		Joins("JOIN roles ON roles.id = users.role_id").
		Where("roles.domain = ?", domain)

	if roleID, err := strconv.ParseInt(c.Query("role_id"), 10, 64); err == nil && roleID > 0 {
		query = query.Where("users.role_id = ?", roleID)
	}

	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"(LOWER(users.name) LIKE ? OR LOWER(users.username) LIKE ? OR LOWER(users.email) LIKE ?)",
			pattern, pattern, pattern,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat user"})
		return
	}

	var users []models.User
	err := query.
		Select("users.*").
		Preload("Role").
		Preload("RegionalAssignments").
		Preload("SDMKdkmpEntry").
		Order("users." + sort + " " + direction).
		Order("users.id").
		Offset((page - 1) * userPerPageDefault).
		Limit(userPerPageDefault).
		Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat user"})
		return
	}

	data := make([]gin.H, 0, len(users))
	for i := range users {
		data = append(data, transformUser(&users[i]))
	}

	totalPages := int((total + int64(userPerPageDefault) - 1) / int64(userPerPageDefault))

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":        page,
			"per_page":    userPerPageDefault,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// CreateUserHandler godoc
//
//	@Summary      Tambah user
//	@Description  Membuat user domain KDKMP; manager wajib terhubung ke data KDKMP, manager wilayah wajib punya cakupan.
//	@Tags         Users
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      userPayload  true  "Data user"
//	@Success      201      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/users [post]
func CreateUserHandler(c *gin.Context) {
	var req userPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	role, ok := validateUserPayload(c, &req, 0, true)
	if !ok {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memproses kata sandi"})
		return
	}

	email := normalizeEmail(req.Email)
	username, err := slug.UniqueColumn(AppDeps.DB, "users", "username", slug.Make(strings.TrimSpace(req.Name)), 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat username"})
		return
	}

	now := time.Now()
	user := models.User{
		RoleID:          &role.ID,
		SDMKdkmpEntryID: req.SDMKdkmpEntryID,
		Name:            strings.TrimSpace(req.Name),
		Username:        &username,
		Email:           email,
		EmailVerifiedAt: &now,
		Password:        string(hash),
	}

	err = AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return syncRegionalAssignments(tx, &user, role, req.RegionalAssignments)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan user"})
		return
	}

	reloaded := reloadUser(c, user.ID)
	if reloaded == nil {
		c.JSON(http.StatusCreated, transformUser(&user))
		return
	}

	c.JSON(http.StatusCreated, transformUser(reloaded))
}

// UpdateUserHandler godoc
//
//	@Summary      Ubah user
//	@Description  Mengubah data user KDKMP; kata sandi opsional.
//	@Tags         Users
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id       path      int          true  "ID user"
//	@Param        payload  body      userPayload  true  "Data user"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      403      {object}  map[string]string
//	@Failure      404      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/users/{id} [put]
func UpdateUserHandler(c *gin.Context) {
	user, ok := findKdkmpUser(c)
	if !ok {
		return
	}

	var req userPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	role, ok := validateUserPayload(c, &req, user.ID, false)
	if !ok {
		return
	}

	updates := map[string]any{
		"role_id":            role.ID,
		"name":               strings.TrimSpace(req.Name),
		"email":              normalizeEmail(req.Email),
		"sdm_kdkmp_entry_id": req.SDMKdkmpEntryID,
	}

	if strings.TrimSpace(req.Password) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memproses kata sandi"})
			return
		}
		updates["password"] = string(hash)
	}

	err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return err
		}
		user.RoleID = &role.ID
		user.Name = strings.TrimSpace(req.Name)
		user.Email = normalizeEmail(req.Email)
		user.SDMKdkmpEntryID = req.SDMKdkmpEntryID
		return syncRegionalAssignments(tx, user, role, req.RegionalAssignments)
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui user"})
		return
	}

	reloaded := reloadUser(c, user.ID)
	if reloaded == nil {
		c.JSON(http.StatusOK, transformUser(user))
		return
	}

	c.JSON(http.StatusOK, transformUser(reloaded))
}

// DeleteUserHandler godoc
//
//	@Summary      Hapus user
//	@Description  Menghapus user KDKMP (tidak dapat menghapus akun sendiri).
//	@Tags         Users
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id  path  int  true  "ID user"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Failure      422  {object}  map[string]string
//	@Router       /api/v1/users/{id} [delete]
func DeleteUserHandler(c *gin.Context) {
	user, ok := findKdkmpUser(c)
	if !ok {
		return
	}

	current := middleware.CurrentUser(c)
	if current != nil && current.ID == user.ID {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Tidak dapat menghapus akun sendiri."})
		return
	}

	if err := AppDeps.DB.WithContext(c.Request.Context()).Delete(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghapus user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User berhasil dihapus."})
}

// RegionOptionsHandler godoc
//
//	@Summary      Opsi wilayah
//	@Description  Kombinasi wilayah distinct dari KDKMP yang dikelola manager (untuk editor cakupan wilayah).
//	@Tags         Users
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Router       /api/v1/region-options [get]
func RegionOptionsHandler(c *gin.Context) {
	type regionRow struct {
		Provinsi      string  `json:"provinsi"`
		KotaKabupaten *string `json:"kota_kabupaten"`
		Kecamatan     *string `json:"kecamatan"`
	}

	var regions []regionRow = make([]regionRow, 0)
	err := AppDeps.DB.WithContext(c.Request.Context()).
		Table("sdm_kdkmp_entries AS s").
		Select("DISTINCT s.provinsi, s.kota_kabupaten, s.kecamatan").
		Joins("JOIN users u ON u.sdm_kdkmp_entry_id = s.id").
		Joins("JOIN roles r ON r.id = u.role_id AND r.domain = ? AND r.slug = ?", models.RoleDomainKdkmp, models.RoleSlugManager).
		Where("s.provinsi IS NOT NULL AND s.provinsi <> ''").
		Order("s.provinsi, s.kota_kabupaten, s.kecamatan").
		Scan(&regions).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat opsi wilayah"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"regions": regions})
}

// KdkmpOptionsHandler godoc
//
//	@Summary      Opsi data KDKMP
//	@Description  Daftar data KDKMP untuk dihubungkan ke akun manager (beserta pemilik saat ini).
//	@Tags         Users
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Router       /api/v1/kdkmp-options [get]
func KdkmpOptionsHandler(c *gin.Context) {
	type kdkmpRow struct {
		ID                   int64   `json:"id"`
		NIK                  *string `json:"nik"`
		NamaKoperasi         *string `json:"nama_koperasi"`
		Provinsi             *string `json:"provinsi"`
		KotaKabupaten        *string `json:"kota_kabupaten"`
		Kecamatan            *string `json:"kecamatan"`
		Desa                 *string `json:"desa"`
		AssignedManagerUseID *int64  `json:"assigned_manager_user_id"`
	}

	var entries []kdkmpRow = make([]kdkmpRow, 0)
	err := AppDeps.DB.WithContext(c.Request.Context()).
		Table("sdm_kdkmp_entries AS s").
		Select("s.id, s.nik, s.nama_koperasi, s.provinsi, s.kota_kabupaten, s.kecamatan, s.desa, u.id AS assigned_manager_use_id").
		Joins("LEFT JOIN users u ON u.sdm_kdkmp_entry_id = s.id").
		Order("s.nama_koperasi").
		Scan(&entries).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat data KDKMP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"kdkmp": entries})
}

// UploadManagerSKDocumentHandler godoc
//
//	@Summary      Unggah SK Manager
//	@Description  Mengunggah dokumen SK (PDF, maks 10 MB) untuk akun manager KDKMP. Dokumen lama diganti.
//	@Tags         Users
//	@Accept       multipart/form-data
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id                  path  int   true  "ID user manager"
//	@Param        manager_sk_document  formData  file  true  "Dokumen SK (PDF)"
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Failure      422  {object}  map[string]string
//	@Router       /api/v1/users/{id}/manager-sk-document [post]
func UploadManagerSKDocumentHandler(c *gin.Context) {
	user, ok := findKdkmpUser(c)
	if !ok {
		return
	}
	if !user.IsKdkmpManager() {
		c.JSON(http.StatusForbidden, gin.H{"message": "Dokumen SK hanya untuk akun manager KDKMP."})
		return
	}

	file, err := c.FormFile("manager_sk_document")
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Dokumen SK Manager wajib dipilih."})
		return
	}
	if file.Size > maxSKDocumentSize {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Ukuran dokumen SK Manager maksimal 10 MB."})
		return
	}
	if contentType(file) != "application/pdf" || !strings.HasSuffix(strings.ToLower(file.Filename), ".pdf") {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Dokumen SK Manager harus berupa file PDF."})
		return
	}

	objectName := fmt.Sprintf("%s/%d/%s.pdf", skDocumentObjectPath, user.ID, uuid.NewString())

	contents, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membaca dokumen"})
		return
	}
	defer func() { _ = contents.Close() }()

	ctx := c.Request.Context()
	if err := AppDeps.Files.Put(ctx, objectName, contents, file.Size, "application/pdf"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengunggah dokumen"})
		return
	}

	current := middleware.CurrentUser(c)
	var uploadedBy int64
	if current != nil {
		uploadedBy = current.ID
	}

	metadata := skDocumentMetadata{
		Disk:         "minio",
		Path:         objectName,
		OriginalName: file.Filename,
		MimeType:     "application/pdf",
		Size:         file.Size,
		UploadedBy:   uploadedBy,
		UploadedAt:   time.Now().Format(time.RFC3339),
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		_ = AppDeps.Files.Remove(ctx, objectName)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan metadata dokumen"})
		return
	}

	previous := parseSKDocument(user.ManagerSKDocument)
	encoded := string(payload)

	if err := AppDeps.DB.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("manager_sk_document", encoded).Error; err != nil {
		_ = AppDeps.Files.Remove(ctx, objectName)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan dokumen"})
		return
	}

	if previous != nil && previous.Path != "" && previous.Path != objectName {
		_ = AppDeps.Files.Remove(ctx, previous.Path)
	}

	user.ManagerSKDocument = &encoded

	c.JSON(http.StatusOK, gin.H{"message": "Dokumen SK Manager berhasil diunggah.", "manager_sk_document": skDocumentResponse(user)})
}

// PreviewManagerSKDocumentHandler godoc
//
//	@Summary      Pratinjau SK Manager
//	@Description  Menampilkan dokumen SK Manager (superadmin atau pemilik akun manager).
//	@Tags         Users
//	@Produce      application/pdf
//	@Security     CookieAuth
//	@Param        id  path  int  true  "ID user manager"
//	@Success      200  {file}  binary
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/users/{id}/manager-sk-document [get]
func PreviewManagerSKDocumentHandler(c *gin.Context) {
	user, ok := findKdkmpUser(c)
	if !ok {
		return
	}
	if !user.IsKdkmpManager() {
		c.JSON(http.StatusNotFound, gin.H{"message": "Dokumen tidak ditemukan"})
		return
	}

	current := middleware.CurrentUser(c)
	allowed := current != nil && (current.IsSuperadmin() || current.ID == user.ID)
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"message": "Anda tidak memiliki akses"})
		return
	}

	metadata := parseSKDocument(user.ManagerSKDocument)
	if metadata == nil || metadata.Path == "" {
		c.JSON(http.StatusNotFound, gin.H{"message": "Dokumen tidak ditemukan"})
		return
	}

	object, err := AppDeps.Files.Stream(c.Request.Context(), metadata.Path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Dokumen tidak ditemukan"})
		return
	}
	defer func() { _ = object.Close() }()

	filename := metadata.OriginalName
	if filename == "" {
		filename = "sk-manager.pdf"
	}

	c.DataFromReader(http.StatusOK, metadata.Size, "application/pdf", object, map[string]string{
		"Content-Disposition":    fmt.Sprintf("inline; filename=%q", filename),
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, max-age=300",
	})
}

// validateUserPayload memvalidasi payload user dan mengembalikan role target.
// excludeUserID > 0 saat update; passwordRequired true saat create.
func validateUserPayload(c *gin.Context, req *userPayload, excludeUserID int64, passwordRequired bool) (*models.Role, bool) {
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		domain = models.RoleDomainKdkmp
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama wajib diisi"})
		return nil, false
	}
	if len([]rune(name)) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama maksimal 255 karakter"})
		return nil, false
	}

	email := normalizeEmail(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Format email tidak valid"})
		return nil, false
	}
	if len(email) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Email maksimal 255 karakter"})
		return nil, false
	}

	password := strings.TrimSpace(req.Password)
	if passwordRequired && password == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi wajib diisi"})
		return nil, false
	}
	if password != "" {
		if len(password) < 8 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi minimal 8 karakter"})
			return nil, false
		}
		if password != req.PasswordConfirmation {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Konfirmasi kata sandi tidak cocok"})
			return nil, false
		}
	}

	db := AppDeps.DB.WithContext(c.Request.Context())

	var duplicateEmail int64
	emailQuery := db.Model(&models.User{}).Where("LOWER(email) = ?", email)
	if excludeUserID > 0 {
		emailQuery = emailQuery.Where("id <> ?", excludeUserID)
	}
	if err := emailQuery.Count(&duplicateEmail).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa email"})
		return nil, false
	}
	if duplicateEmail > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Email sudah digunakan akun lain"})
		return nil, false
	}

	var role models.Role
	err := db.First(&role, req.RoleID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Role tidak ditemukan"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat role"})
		return nil, false
	}
	if role.Domain != domain {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Role tidak sesuai domain user"})
		return nil, false
	}

	isManager := role.Slug == models.RoleSlugManager
	isRegionalManager := role.Slug == models.RoleSlugRegionalManager

	// Penautan data KDKMP hanya untuk role manager.
	if !isManager {
		if req.SDMKdkmpEntryID != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data KDKMP hanya dapat diberikan kepada role Manager."})
			return nil, false
		}
	} else {
		if req.SDMKdkmpEntryID == nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Manager wajib dihubungkan ke satu data KDKMP."})
			return nil, false
		}

		available, err := isKdkmpEntryAvailable(db, *req.SDMKdkmpEntryID, excludeUserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa data KDKMP"})
			return nil, false
		}
		if !available {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data KDKMP tersebut sudah terhubung ke akun Manager lain."})
			return nil, false
		}
	}

	// Cakupan wilayah hanya untuk role Manager Wilayah dan wajib ada.
	if !isRegionalManager && len(req.RegionalAssignments) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Cakupan wilayah hanya dapat diberikan kepada role Manager Wilayah."})
		return nil, false
	}
	if isRegionalManager && len(req.RegionalAssignments) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Manager Wilayah wajib memiliki minimal satu cakupan wilayah."})
		return nil, false
	}
	if len(req.RegionalAssignments) > maxRegionalAssign {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Cakupan wilayah maksimal 25 baris."})
		return nil, false
	}

	seen := make(map[string]bool, len(req.RegionalAssignments))
	for _, assignment := range req.RegionalAssignments {
		scope := strings.TrimSpace(assignment.ScopeLevel)
		if !models.IsValidRegionalScope(scope) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Scope level cakupan wilayah tidak valid"})
			return nil, false
		}

		provinsi := strings.TrimSpace(assignment.Provinsi)
		kota := trimmedOrEmpty(assignment.KotaKabupaten)
		kecamatan := trimmedOrEmpty(assignment.Kecamatan)

		if provinsi == "" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Provinsi wajib diisi untuk cakupan wilayah."})
			return nil, false
		}
		if (scope == models.RegionalScopeRegency || scope == models.RegionalScopeDistrict) && kota == "" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kabupaten/Kota wajib diisi untuk cakupan ini."})
			return nil, false
		}
		if scope == models.RegionalScopeDistrict && kecamatan == "" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kecamatan wajib diisi untuk cakupan kecamatan."})
			return nil, false
		}

		key := strings.Join([]string{scope, provinsi, kota, kecamatan}, "|")
		if seen[key] {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Cakupan wilayah tersebut sudah ditambahkan."})
			return nil, false
		}
		seen[key] = true

		monitored, err := regionHasManagedKdkmp(db, scope, provinsi, kota, kecamatan)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memeriksa cakupan wilayah"})
			return nil, false
		}
		if !monitored {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Cakupan wilayah tidak memiliki data KDKMP yang dapat dimonitor."})
			return nil, false
		}
	}

	return &role, true
}

// syncRegionalAssignments mengganti seluruh cakupan wilayah user sesuai payload.
func syncRegionalAssignments(tx *gorm.DB, user *models.User, role *models.Role, assignments []regionalAssignmentPayload) error {
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserRegionalAssignment{}).Error; err != nil {
		return err
	}

	if role.Slug != models.RoleSlugRegionalManager {
		return nil
	}

	for _, assignment := range assignments {
		scope := strings.TrimSpace(assignment.ScopeLevel)
		provinsi := strings.TrimSpace(assignment.Provinsi)

		var kota *string
		if scope != models.RegionalScopeProvince {
			if value := trimmedOrEmpty(assignment.KotaKabupaten); value != "" {
				kota = &value
			}
		}

		var kecamatan *string
		if scope == models.RegionalScopeDistrict {
			if value := trimmedOrEmpty(assignment.Kecamatan); value != "" {
				kecamatan = &value
			}
		}

		record := models.UserRegionalAssignment{
			UserID:        user.ID,
			ScopeLevel:    scope,
			Provinsi:      provinsi,
			KotaKabupaten: kota,
			Kecamatan:     kecamatan,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
	}

	return nil
}

func regionHasManagedKdkmp(db *gorm.DB, scope string, provinsi string, kota string, kecamatan string) (bool, error) {
	query := db.Table("sdm_kdkmp_entries AS s").
		Joins("JOIN users u ON u.sdm_kdkmp_entry_id = s.id").
		Joins("JOIN roles r ON r.id = u.role_id AND r.domain = ? AND r.slug = ?", models.RoleDomainKdkmp, models.RoleSlugManager).
		Where("s.provinsi = ?", provinsi)

	if scope != models.RegionalScopeProvince {
		query = query.Where("s.kota_kabupaten = ?", kota)
	}
	if scope == models.RegionalScopeDistrict {
		query = query.Where("s.kecamatan = ?", kecamatan)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func isKdkmpEntryAvailable(db *gorm.DB, entryID int64, excludeUserID int64) (bool, error) {
	var count int64
	query := db.Model(&models.User{}).Where("sdm_kdkmp_entry_id = ?", entryID)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func findKdkmpUser(c *gin.Context) (*models.User, bool) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User tidak ditemukan"})
		return nil, false
	}

	var user models.User
	err = AppDeps.DB.WithContext(c.Request.Context()).
		Preload("Role").
		Preload("RegionalAssignments").
		Preload("SDMKdkmpEntry").
		First(&user, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "User tidak ditemukan"})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat user"})
		return nil, false
	}

	if user.Role == nil || user.Role.Domain != models.RoleDomainKdkmp {
		c.JSON(http.StatusNotFound, gin.H{"message": "User tidak ditemukan"})
		return nil, false
	}

	return &user, true
}

func reloadUser(c *gin.Context, userID int64) *models.User {
	var user models.User
	err := AppDeps.DB.WithContext(c.Request.Context()).
		Preload("Role").
		Preload("RegionalAssignments").
		Preload("SDMKdkmpEntry").
		First(&user, userID).Error
	if err != nil {
		return nil
	}
	return &user
}

func transformUser(user *models.User) gin.H {
	response := gin.H{
		"id":                       user.ID,
		"role_id":                  user.RoleID,
		"sdm_kdkmp_entry_id":       user.SDMKdkmpEntryID,
		"name":                     user.Name,
		"username":                 user.Username,
		"email":                    user.Email,
		"email_verified_at":        user.EmailVerifiedAt,
		"has_completed_onboarding": user.HasCompletedOnboarding,
		"created_at":               user.CreatedAt,
		"updated_at":               user.UpdatedAt,
		"manager_sk_document":      skDocumentResponse(user),
	}

	if user.Role != nil {
		response["role"] = gin.H{
			"id":          user.Role.ID,
			"name":        user.Role.Name,
			"slug":        user.Role.Slug,
			"level":       user.Role.Level,
			"level_label": models.RoleLevelLabel(user.Role.Level),
			"domain":      user.Role.Domain,
		}
	} else {
		response["role"] = nil
	}

	assignments := make([]gin.H, 0, len(user.RegionalAssignments))
	for _, assignment := range user.RegionalAssignments {
		assignments = append(assignments, gin.H{
			"id":             assignment.ID,
			"scope_level":    assignment.ScopeLevel,
			"provinsi":       assignment.Provinsi,
			"kota_kabupaten": assignment.KotaKabupaten,
			"kecamatan":      assignment.Kecamatan,
		})
	}
	response["regional_assignments"] = assignments

	if user.SDMKdkmpEntry != nil {
		response["kdkmp"] = gin.H{
			"id":                       user.SDMKdkmpEntry.ID,
			"nik":                      user.SDMKdkmpEntry.NIK,
			"nama_koperasi":            user.SDMKdkmpEntry.NamaKoperasi,
			"provinsi":                 user.SDMKdkmpEntry.Provinsi,
			"kota_kabupaten":           user.SDMKdkmpEntry.KotaKabupaten,
			"kecamatan":                user.SDMKdkmpEntry.Kecamatan,
			"desa":                     user.SDMKdkmpEntry.Desa,
			"assigned_manager_user_id": user.ID,
		}
	} else {
		response["kdkmp"] = nil
	}

	return response
}

func skDocumentResponse(user *models.User) gin.H {
	metadata := parseSKDocument(user.ManagerSKDocument)
	if metadata == nil || !user.IsKdkmpManager() {
		return nil
	}

	return gin.H{
		"name":        metadata.OriginalName,
		"size":        metadata.Size,
		"uploaded_at": metadata.UploadedAt,
		"preview_url": fmt.Sprintf("/users/%d/manager-sk-document", user.ID),
	}
}

func parseSKDocument(raw *string) *skDocumentMetadata {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}

	var metadata skDocumentMetadata
	if err := json.Unmarshal([]byte(*raw), &metadata); err != nil {
		return nil
	}
	return &metadata
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func trimmedOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func contentType(file *multipart.FileHeader) string {
	if file == nil {
		return ""
	}
	return file.Header.Get("Content-Type")
}
