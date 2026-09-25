package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/twofactor"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginHandler godoc
//
//	@Summary      Login
//	@Description  Autentikasi dengan email & kata sandi; membuat session cookie `ebitda_session`.
//	@Tags         Auth
//	@Accept       json
//	@Produce      json
//	@Param        payload  body      loginRequest  true  "Kredensial"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/auth/login [post]
func LoginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Email dan kata sandi wajib diisi"})
		return
	}

	var user models.User
	err := AppDeps.DB.WithContext(c.Request.Context()).
		Preload("Role").
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(req.Email))).
		First(&user).Error

	invalidCredentials := gin.H{"message": "Email atau kata sandi salah"}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, invalidCredentials)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Terjadi kesalahan pada server"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, invalidCredentials)
		return
	}

	// 2FA aktif: tahan login, minta kode verifikasi dulu.
	if AppDeps.TwoFactor.Enabled(&user) {
		token, err := AppDeps.TwoFactor.CreateChallenge(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai verifikasi 2FA"})
			return
		}

		setTwoFactorCookie(c, token, int(twofactor.ChallengeTTL.Seconds()))
		c.JSON(http.StatusOK, gin.H{"two_factor_required": true})
		return
	}

	if err := replaceSession(c, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat sesi"})
		return
	}

	c.JSON(http.StatusOK, userResponse(&user))
}

// LogoutHandler godoc
//
//	@Summary      Logout
//	@Description  Menghapus session di Redis dan cookie.
//	@Tags         Auth
//	@Produce      json
//	@Success      200  {object}  map[string]string
//	@Router       /api/v1/auth/logout [post]
func LogoutHandler(c *gin.Context) {
	if sessionID, err := c.Cookie(AppDeps.SessionCookie); err == nil && sessionID != "" {
		_ = AppDeps.Session.Destroy(c.Request.Context(), sessionID)
	}

	clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil keluar"})
}

// MeHandler godoc
//
//	@Summary      Profil pengguna saat ini
//	@Description  Mengembalikan user yang sedang login (beserta role).
//	@Tags         Auth
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Router       /api/v1/auth/me [get]
func MeHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}
	c.JSON(http.StatusOK, userResponse(user))
}

func setSessionCookie(c *gin.Context, sessionID string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		AppDeps.SessionCookie,
		sessionID,
		maxAge,
		"/",
		"",
		AppDeps.SessionSecure,
		true, // HttpOnly
	)
}

func clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		AppDeps.SessionCookie,
		"",
		-1,
		"/",
		"",
		AppDeps.SessionSecure,
		true,
	)
}

// replaceSession mengganti sesi aktif agar ID sesi sebelum autentikasi atau
// perubahan kredensial tidak dapat dipakai kembali.
func replaceSession(c *gin.Context, userID int64) error {
	if sessionID, err := c.Cookie(AppDeps.SessionCookie); err == nil && sessionID != "" {
		_ = AppDeps.Session.Destroy(c.Request.Context(), sessionID)
	}

	sessionID, err := AppDeps.Session.Create(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	setSessionCookie(c, sessionID, int(AppDeps.SessionTTL.Seconds()))
	return nil
}

func userResponse(user *models.User) gin.H {
	response := gin.H{
		"id":                       user.ID,
		"name":                     user.Name,
		"username":                 user.Username,
		"email":                    user.Email,
		"email_verified_at":        user.EmailVerifiedAt,
		"sdm_kdkmp_entry_id":       user.SDMKdkmpEntryID,
		"has_completed_onboarding": user.HasCompletedOnboarding,
		"two_factor_enabled":       user.TwoFactorConfirmedAt != nil,
		"manager_sk_document":      skDocumentResponse(user),
	}

	if user.Role != nil {
		response["role"] = gin.H{
			"id":     user.Role.ID,
			"name":   user.Role.Name,
			"slug":   user.Role.Slug,
			"level":  user.Role.Level,
			"domain": user.Role.Domain,
		}
	} else {
		response["role"] = nil
	}

	return response
}
