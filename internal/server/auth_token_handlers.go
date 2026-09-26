package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/session"
)

// refreshTokenRequest adalah payload refresh/logout untuk klien API (Bearer flow).
type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshHandler godoc
//
//	@Summary      Perbarui access token
//	@Description  Menukar refresh token (dari cookie atau body `refresh_token`) dengan access token baru.
//	@Description  Refresh token lama dirotasi (single-use); pemakaian ulang token lama mencabut seluruh sesi user.
//	@Tags         Auth
//	@Accept       json
//	@Produce      json
//	@Param        payload  body      refreshTokenRequest  false  "Refresh token untuk klien API (opsional bila memakai cookie)"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Router       /api/v1/auth/refresh [post]
func RefreshHandler(c *gin.Context) {
	ctx := c.Request.Context()

	refreshToken := ""
	fromCookie := false

	if cookie, err := c.Cookie(AppDeps.RefreshCookie); err == nil && cookie != "" {
		refreshToken = cookie
		fromCookie = true
	} else {
		var req refreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			refreshToken = strings.TrimSpace(req.RefreshToken)
		}
	}

	if refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Sesi tidak ditemukan, silakan masuk kembali"})
		return
	}

	newRefreshToken, data, err := AppDeps.Refresh.Rotate(ctx, refreshToken)
	if err != nil {
		clearAuthCookies(c)
		if errors.Is(err, session.ErrReplay) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Refresh token sudah pernah dipakai; seluruh sesi dicabut. Silakan masuk kembali",
			})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Sesi sudah berakhir, silakan masuk kembali"})
		return
	}

	accessToken, expiresAt, err := AppDeps.Tokens.IssueAccess(data.UserID, data.FamilyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menerbitkan token"})
		return
	}

	if fromCookie {
		setAccessCookie(c, accessToken, int(AppDeps.Tokens.AccessTTL().Seconds()))
		setRefreshCookie(c, newRefreshToken, int(AppDeps.Refresh.TTL().Seconds()))
	}

	c.JSON(http.StatusOK, gin.H{
		"token_type":    "Bearer",
		"access_token":  accessToken,
		"expires_in":    int(time.Until(expiresAt).Seconds()),
		"refresh_token": newRefreshToken,
	})
}

// LogoutAllHandler godoc
//
//	@Summary      Logout semua perangkat
//	@Description  Mencabut seluruh refresh token milik user yang sedang login.
//	@Tags         Auth
//	@Produce      json
//	@Security     CookieAuth
//	@Security     BearerAuth
//	@Success      200  {object}  map[string]string
//	@Failure      401  {object}  map[string]string
//	@Router       /api/v1/auth/logout-all [post]
func LogoutAllHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	if err := AppDeps.Refresh.RevokeUser(c.Request.Context(), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mencabut seluruh sesi"})
		return
	}

	clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "Semua sesi berhasil dicabut"})
}
