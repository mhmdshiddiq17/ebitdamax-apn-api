package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/passkey"
)

const passkeyCookie = "ebitda_passkey"

type passkeyLoginOptionsRequest struct {
	Email string `json:"email"`
}

// ListPasskeysHandler godoc
//
//	@Summary      Daftar passkey
//	@Description  Menampilkan passkey milik user yang sedang login.
//	@Tags         Passkeys
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Router       /api/v1/passkeys [get]
func ListPasskeysHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	passkeys, err := AppDeps.Passkey.List(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat passkey"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"passkeys": passkeys})
}

// BeginPasskeyRegistrationHandler godoc
//
//	@Summary      Mulai registrasi passkey
//	@Description  Mengembalikan opsi WebAuthn (publicKey) untuk membuat passkey baru; menyimpan challenge di cookie `ebitda_passkey`.
//	@Tags         Passkeys
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Failure      503  {object}  map[string]string
//	@Router       /api/v1/passkeys/register/options [post]
func BeginPasskeyRegistrationHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	creation, token, err := AppDeps.Passkey.BeginRegistration(c.Request.Context(), user)
	if err != nil {
		if errors.Is(err, passkey.ErrDisabled) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Passkey belum dikonfigurasi"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai registrasi passkey"})
		return
	}

	setPasskeyCookie(c, token, int((5 * 60)))
	c.JSON(http.StatusOK, creation)
}

// FinishPasskeyRegistrationHandler godoc
//
//	@Summary      Selesaikan registrasi passkey
//	@Description  Memverifikasi respons attestation dan menyimpan passkey. Nama perangkat opsional via query `name`.
//	@Tags         Passkeys
//	@Accept       json
//	@Produce      json
//	@Security     CookieAuth
//	@Param        name  query  string  false  "Nama perangkat"
//	@Success      201   {object}  map[string]any
//	@Failure      401   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      503   {object}  map[string]string
//	@Router       /api/v1/passkeys/register [post]
func FinishPasskeyRegistrationHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	token, _ := c.Cookie(passkeyCookie)
	if token == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Sesi registrasi tidak ditemukan, ulangi"})
		return
	}

	registered, err := AppDeps.Passkey.FinishRegistration(c.Request.Context(), user, token, c.Request, c.Query("name"))
	if err != nil {
		switch {
		case errors.Is(err, passkey.ErrChallengeNotFound):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Sesi registrasi sudah berakhir, ulangi"})
		case errors.Is(err, passkey.ErrAlreadyRegistered):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Passkey ini sudah terdaftar"})
		case errors.Is(err, passkey.ErrDisabled):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Passkey belum dikonfigurasi"})
		default:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Verifikasi passkey gagal"})
		}
		return
	}

	clearPasskeyCookie(c)
	c.JSON(http.StatusCreated, gin.H{"passkey": registered})
}

// DeletePasskeyHandler godoc
//
//	@Summary      Hapus passkey
//	@Description  Menghapus passkey milik user yang sedang login.
//	@Tags         Passkeys
//	@Produce      json
//	@Security     CookieAuth
//	@Param        id  path  int  true  "ID passkey"
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /api/v1/passkeys/{id} [delete]
func DeletePasskeyHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	passkeyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Passkey tidak ditemukan"})
		return
	}

	if err := AppDeps.Passkey.Delete(c.Request.Context(), user.ID, passkeyID); err != nil {
		if errors.Is(err, passkey.ErrPasskeyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Passkey tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menghapus passkey"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Passkey dihapus"})
}

// BeginPasskeyLoginHandler godoc
//
//	@Summary      Mulai login passkey
//	@Description  Mengembalikan opsi WebAuthn (publicKey) untuk login passkey berdasarkan email; menyimpan challenge di cookie `ebitda_passkey`.
//	@Tags         Auth
//	@Accept       json
//	@Produce      json
//	@Param        payload  body      passkeyLoginOptionsRequest  true  "Email"
//	@Success      200      {object}  map[string]any
//	@Failure      422      {object}  map[string]string
//	@Failure      503      {object}  map[string]string
//	@Router       /api/v1/auth/passkey/options [post]
func BeginPasskeyLoginHandler(c *gin.Context) {
	var req passkeyLoginOptionsRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Email wajib diisi"})
		return
	}

	assertion, token, err := AppDeps.Passkey.BeginLogin(c.Request.Context(), req.Email)
	if err != nil {
		switch {
		case errors.Is(err, passkey.ErrNoPasskey):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Passkey tidak tersedia untuk akun ini"})
		case errors.Is(err, passkey.ErrDisabled):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Passkey belum dikonfigurasi"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai login passkey"})
		}
		return
	}

	setPasskeyCookie(c, token, int((5 * 60)))
	c.JSON(http.StatusOK, assertion)
}

// FinishPasskeyLoginHandler godoc
//
//	@Summary      Selesaikan login passkey
//	@Description  Memverifikasi respons assertion, membuat sesi login, lalu mengembalikan data user + token pair JWT (sekaligus cookie HttpOnly untuk browser).
//	@Tags         Auth
//	@Accept       json
//	@Produce      json
//	@Success      200  {object}  map[string]any
//	@Failure      422  {object}  map[string]string
//	@Failure      503  {object}  map[string]string
//	@Router       /api/v1/auth/passkey/login [post]
func FinishPasskeyLoginHandler(c *gin.Context) {
	token, _ := c.Cookie(passkeyCookie)
	if token == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Sesi login tidak ditemukan, ulangi"})
		return
	}

	user, err := AppDeps.Passkey.FinishLogin(c.Request.Context(), token, c.Request)
	if err != nil {
		switch {
		case errors.Is(err, passkey.ErrChallengeNotFound):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Sesi login sudah berakhir, ulangi"})
		case errors.Is(err, passkey.ErrNoPasskey):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Passkey tidak tersedia untuk akun ini"})
		case errors.Is(err, passkey.ErrDisabled):
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "Passkey belum dikonfigurasi"})
		default:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Verifikasi passkey gagal"})
		}
		return
	}

	accessToken, refreshToken, expiresAt, err := issueTokenPair(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat sesi"})
		return
	}

	applyAuthCookies(c, accessToken, refreshToken)

	clearPasskeyCookie(c)

	c.JSON(http.StatusOK, authResponseWithTokens(user, accessToken, refreshToken, expiresAt))
}

func setPasskeyCookie(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(passkeyCookie, token, maxAge, "/", "", AppDeps.SessionSecure, true)
}

func clearPasskeyCookie(c *gin.Context) {
	setPasskeyCookie(c, "", -1)
}
