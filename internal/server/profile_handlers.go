package server

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

type updateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type updatePasswordRequest struct {
	CurrentPassword      string `json:"current_password"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

// UpdateProfileHandler godoc
//
//	@Summary      Perbarui profil
//	@Description  Mengubah nama dan email user yang sedang login.
//	@Tags         Profile
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      updateProfileRequest  true  "Data profil"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/profile [patch]
func UpdateProfileHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama wajib diisi"})
		return
	}
	if len([]rune(name)) > 255 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Nama maksimal 255 karakter"})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Format email tidak valid"})
		return
	}

	var emailTaken int64
	if err := AppDeps.DB.Model(&models.User{}).
		Where("LOWER(email) = ? AND id <> ?", email, user.ID).
		Count(&emailTaken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Terjadi kesalahan pada server"})
		return
	}
	if emailTaken > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Email sudah digunakan akun lain"})
		return
	}

	if err := AppDeps.DB.Model(&models.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{"name": name, "email": email}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan profil"})
		return
	}

	user.Name = name
	user.Email = email

	c.JSON(http.StatusOK, userResponse(user))
}

// UpdatePasswordHandler godoc
//
//	@Summary      Ganti kata sandi
//	@Description  Mengganti kata sandi user yang sedang login (wajib kata sandi saat ini).
//	@Tags         Profile
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      updatePasswordRequest  true  "Data kata sandi"
//	@Success      200      {object}  map[string]string
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/profile/password [put]
func UpdatePasswordHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	var req updatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	if req.CurrentPassword == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi saat ini wajib diisi"})
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi baru minimal 8 karakter"})
		return
	}
	if req.Password != req.PasswordConfirmation {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Konfirmasi kata sandi tidak cocok"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)) != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi saat ini salah"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memproses kata sandi"})
		return
	}

	if err := AppDeps.DB.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("password", string(hash)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan kata sandi"})
		return
	}
	if err := AppDeps.Session.DestroyUserSessions(c.Request.Context(), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Kata sandi diperbarui, tetapi sesi lama tidak dapat dicabut"})
		return
	}
	if err := replaceSession(c, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Kata sandi diperbarui, tetapi sesi baru gagal dibuat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kata sandi berhasil diperbarui"})
}
