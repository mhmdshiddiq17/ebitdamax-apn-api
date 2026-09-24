package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/twofactor"
)

const twoFactorCookie = "ebitda_2fa"

type twoFactorEnableRequest struct {
	CurrentPassword string `json:"current_password"`
}

type twoFactorCodeRequest struct {
	Code string `json:"code"`
}

type twoFactorPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
}

// TwoFactorStatusHandler godoc
//
//	@Summary      Status 2FA
//	@Description  Mengetahui apakah 2FA aktif untuk user yang sedang login.
//	@Tags         Two-Factor
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]bool
//	@Failure      401  {object}  map[string]string
//	@Router       /api/v1/two-factor [get]
func TwoFactorStatusHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": AppDeps.TwoFactor.Enabled(user),
		"pending": user.TwoFactorSecret != nil && user.TwoFactorConfirmedAt == nil,
	})
}

// TwoFactorEnableHandler godoc
//
//	@Summary      Mulai aktifkan 2FA
//	@Description  Membuat secret TOTP baru (belum aktif sampai dikonfirmasi kode). Wajib kata sandi saat ini.
//	@Tags         Two-Factor
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      twoFactorEnableRequest  true  "Kata sandi saat ini"
//	@Success      200      {object}  map[string]string
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/two-factor/enable [post]
func TwoFactorEnableHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	var req twoFactorEnableRequest
	if err := c.ShouldBindJSON(&req); err != nil || !verifyCurrentPassword(user, req.CurrentPassword) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi saat ini salah"})
		return
	}

	secret, uri, err := AppDeps.TwoFactor.Begin(c.Request.Context(), user)
	if err != nil {
		if errors.Is(err, twofactor.ErrAlreadyEnabled) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "2FA sudah aktif"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai aktivasi 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret, "otpauth_uri": uri})
}

// TwoFactorConfirmHandler godoc
//
//	@Summary      Konfirmasi 2FA
//	@Description  Memverifikasi kode TOTP pertama dan mengaktifkan 2FA; mengembalikan recovery codes (tampil sekali).
//	@Tags         Two-Factor
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      twoFactorCodeRequest  true  "Kode TOTP"
//	@Success      200      {object}  map[string][]string
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/two-factor/confirm [post]
func TwoFactorConfirmHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	var req twoFactorCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	codes, err := AppDeps.TwoFactor.Confirm(c.Request.Context(), user, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, twofactor.ErrInvalidCode):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kode tidak valid"})
		case errors.Is(err, twofactor.ErrNotEnabled):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Mulai aktivasi 2FA terlebih dahulu"})
		case errors.Is(err, twofactor.ErrSecretUnsupported):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Secret 2FA tidak dapat dibaca, ulangi aktivasi"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengaktifkan 2FA"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"recovery_codes": codes})
}

// TwoFactorDisableHandler godoc
//
//	@Summary      Nonaktifkan 2FA
//	@Description  Menonaktifkan 2FA dan menghapus secret + recovery codes. Wajib kata sandi saat ini.
//	@Tags         Two-Factor
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      twoFactorPasswordRequest  true  "Kata sandi saat ini"
//	@Success      200      {object}  map[string]string
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/two-factor [delete]
func TwoFactorDisableHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	var req twoFactorPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || !verifyCurrentPassword(user, req.CurrentPassword) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi saat ini salah"})
		return
	}

	if err := AppDeps.TwoFactor.Disable(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menonaktifkan 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA dinonaktifkan"})
}

// TwoFactorRecoveryCodesHandler godoc
//
//	@Summary      Buat ulang recovery codes
//	@Description  Mengganti seluruh recovery codes 2FA. Wajib kata sandi saat ini.
//	@Tags         Two-Factor
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        payload  body      twoFactorPasswordRequest  true  "Kata sandi saat ini"
//	@Success      200      {object}  map[string][]string
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/two-factor/recovery-codes [post]
func TwoFactorRecoveryCodesHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	var req twoFactorPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || !verifyCurrentPassword(user, req.CurrentPassword) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kata sandi saat ini salah"})
		return
	}

	codes, err := AppDeps.TwoFactor.RegenerateRecoveryCodes(c.Request.Context(), user)
	if err != nil {
		if errors.Is(err, twofactor.ErrNotEnabled) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "2FA belum aktif"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat recovery codes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recovery_codes": codes})
}

// TwoFactorChallengeHandler godoc
//
//	@Summary      Verifikasi 2FA saat login
//	@Description  Menyelesaikan login yang tertahan 2FA memakai kode TOTP atau recovery code.
//	@Tags         Auth
//	@Accept       json
//	@Produce      json
//	@Param        payload  body      twoFactorCodeRequest  true  "Kode TOTP atau recovery code"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Failure      422      {object}  map[string]string
//	@Router       /api/v1/auth/two-factor-challenge [post]
func TwoFactorChallengeHandler(c *gin.Context) {
	token, err := c.Cookie(twoFactorCookie)
	if err != nil || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Sesi verifikasi tidak ditemukan, silakan masuk ulang"})
		return
	}

	challenge, err := AppDeps.TwoFactor.ResolveChallenge(c.Request.Context(), token)
	if err != nil {
		clearTwoFactorCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Sesi verifikasi sudah berakhir, silakan masuk ulang"})
		return
	}

	var req twoFactorCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Data tidak valid"})
		return
	}

	var user models.User
	err = AppDeps.DB.WithContext(c.Request.Context()).
		Preload("Role").
		First(&user, challenge.UserID).Error
	if err != nil {
		clearTwoFactorCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Akun tidak ditemukan"})
		return
	}

	if err := AppDeps.TwoFactor.Verify(c.Request.Context(), &user, req.Code); err != nil {
		failErr := AppDeps.TwoFactor.FailChallenge(c.Request.Context(), token, challenge)
		if errors.Is(failErr, twofactor.ErrTooManyTries) {
			clearTwoFactorCookie(c)
			c.JSON(http.StatusTooManyRequests, gin.H{"message": "Terlalu banyak percobaan, silakan masuk ulang"})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Kode tidak valid"})
		return
	}

	sessionID, err := AppDeps.Session.Create(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat sesi"})
		return
	}

	_ = AppDeps.TwoFactor.ClearChallenge(c.Request.Context(), token)
	clearTwoFactorCookie(c)
	setSessionCookie(c, sessionID, int(AppDeps.SessionTTL.Seconds()))

	c.JSON(http.StatusOK, userResponse(&user))
}

func verifyCurrentPassword(user *models.User, password string) bool {
	if password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil
}

func setTwoFactorCookie(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(twoFactorCookie, token, maxAge, "/", "", AppDeps.SessionSecure, true)
}

func clearTwoFactorCookie(c *gin.Context) {
	setTwoFactorCookie(c, "", -1)
}
