package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/session"
	"agrinaspangan/ebitda-api/internal/twofactor"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginHandler godoc
//
//	@Summary      Login
//	@Description  Autentikasi email & kata sandi. Mengembalikan data user beserta token pair JWT (`access_token`, `refresh_token`) dan menyetel cookie HttpOnly `ebitda_access` + `ebitda_refresh` untuk browser. Bila 2FA aktif: `{two_factor_required: true, challenge_token}` untuk dilanjutkan ke /auth/two-factor-challenge.
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

	user, err := findUserByCredentials(c, req.Email, req.Password)
	if err != nil {
		respondLoginError(c, err)
		return
	}

	// 2FA aktif: tahan login, minta kode verifikasi dulu.
	if AppDeps.TwoFactor.Enabled(user) {
		challengeToken, err := AppDeps.TwoFactor.CreateChallenge(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai verifikasi 2FA"})
			return
		}

		setTwoFactorCookie(c, challengeToken, int(twofactor.ChallengeTTL.Seconds()))
		c.JSON(http.StatusOK, gin.H{
			"two_factor_required": true,
			"challenge_token":     challengeToken,
		})
		return
	}

	accessToken, refreshToken, expiresAt, err := issueTokenPair(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat sesi"})
		return
	}

	applyAuthCookies(c, accessToken, refreshToken)

	c.JSON(http.StatusOK, authResponseWithTokens(user, accessToken, refreshToken, expiresAt))
}

// LogoutHandler godoc
//
//	@Summary      Logout
//	@Description  Mencabut refresh token (cookie) dan menghapus cookie access + refresh.
//	@Tags         Auth
//	@Accept       json
//	@Produce      json
//	@Param        payload  body      refreshTokenRequest  false  "Refresh token untuk klien API (opsional bila memakai cookie)"
//	@Success      200  {object}  map[string]string
//	@Router       /api/v1/auth/logout [post]
func LogoutHandler(c *gin.Context) {
	ctx := c.Request.Context()
	revoked := false

	if refreshToken, err := c.Cookie(AppDeps.RefreshCookie); err == nil && refreshToken != "" {
		_ = AppDeps.Refresh.Revoke(ctx, refreshToken)
		revoked = true
	}

	if !revoked {
		var req refreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil && strings.TrimSpace(req.RefreshToken) != "" {
			_ = AppDeps.Refresh.Revoke(ctx, strings.TrimSpace(req.RefreshToken))
		}
	}

	clearAuthCookies(c)
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

func setAccessCookie(c *gin.Context, rawToken string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		AppDeps.AccessCookie,
		rawToken,
		maxAge,
		"/",
		"",
		AppDeps.SessionSecure,
		true, // HttpOnly
	)
}

func clearAccessCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		AppDeps.AccessCookie,
		"",
		-1,
		"/",
		"",
		AppDeps.SessionSecure,
		true,
	)
}

func setRefreshCookie(c *gin.Context, refreshToken string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		AppDeps.RefreshCookie,
		refreshToken,
		maxAge,
		"/",
		"",
		AppDeps.SessionSecure,
		true, // HttpOnly
	)
}

func clearAuthCookies(c *gin.Context) {
	clearAccessCookie(c)
	setRefreshCookie(c, "", -1)
}

// issueAuthSession menerbitkan token pair dan menyimpannya di cookie HttpOnly.
// Dipakai alur yang tidak perlu mengembalikan token di body (mis. ganti password).
func issueAuthSession(c *gin.Context, userID int64) error {
	accessToken, refreshToken, _, err := issueTokenPair(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	applyAuthCookies(c, accessToken, refreshToken)
	return nil
}

// applyAuthCookies menulis cookie access + refresh untuk browser.
func applyAuthCookies(c *gin.Context, accessToken string, refreshToken string) {
	setAccessCookie(c, accessToken, int(AppDeps.Tokens.AccessTTL().Seconds()))
	setRefreshCookie(c, refreshToken, int(AppDeps.Refresh.TTL().Seconds()))
}

// authResponseWithTokens menggabungkan data user dengan token pair JWT.
func authResponseWithTokens(user *models.User, accessToken string, refreshToken string, expiresAt time.Time) gin.H {
	response := userResponse(user)
	response["token_type"] = "Bearer"
	response["access_token"] = accessToken
	response["expires_in"] = int(time.Until(expiresAt).Seconds())
	response["refresh_token"] = refreshToken
	return response
}

// issueTokenPair menerbitkan access + refresh token untuk klien API (Body/Bearer).
func issueTokenPair(ctx context.Context, userID int64) (string, string, time.Time, error) {
	familyID, err := session.NewToken()
	if err != nil {
		return "", "", time.Time{}, err
	}

	refreshToken, err := AppDeps.Refresh.Create(ctx, userID, familyID)
	if err != nil {
		return "", "", time.Time{}, err
	}

	accessToken, expiresAt, err := AppDeps.Tokens.IssueAccess(userID, familyID)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return accessToken, refreshToken, expiresAt, nil
}

// findUserByCredentials mencari user berdasarkan email + kata sandi.
func findUserByCredentials(c *gin.Context, email string, password string) (*models.User, error) {
	var user models.User
	err := AppDeps.DB.WithContext(c.Request.Context()).
		Preload("Role").
		Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errInvalidCredentials
		}
		return nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, errInvalidCredentials
	}

	return &user, nil
}

var errInvalidCredentials = errors.New("kredensial tidak valid")

func respondLoginError(c *gin.Context, err error) {
	if errors.Is(err, errInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Email atau kata sandi salah"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": "Terjadi kesalahan pada server"})
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
