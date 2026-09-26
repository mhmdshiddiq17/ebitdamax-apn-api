package middleware

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/session"
	"agrinaspangan/ebitda-api/internal/token"
)

// ContextUserKey adalah key user di gin.Context.
const ContextUserKey = "auth_user"

// Auth adalah middleware autentikasi berbasis JWT access token
// (cookie HttpOnly untuk web, header Authorization: Bearer untuk API client).
// Access token yang kedaluwarsa diperbarui otomatis dari refresh cookie
// (tanpa rotasi, agar aman terhadap request paralel); rotasi refresh terjadi
// di endpoint POST /auth/refresh.
type Auth struct {
	DB            *gorm.DB
	Tokens        *token.Service
	Refresh       *session.RefreshStore
	Cookie        string
	RefreshCookie string
	Secure        bool
}

// Required menolak request tanpa access token valid dan menaruh user di context.
func (a *Auth) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		bearer := bearerToken(c)
		isBearer := bearer != ""
		raw := bearer
		if raw == "" {
			if cookie, err := c.Cookie(a.Cookie); err == nil {
				raw = cookie
			}
		}

		var claims *token.Claims
		verifyErr := error(nil)

		if raw != "" {
			claims, verifyErr = a.Tokens.Verify(raw)
		}

		if claims == nil {
			// Auto-refresh hanya untuk alur cookie web (Bearer dikelola klien).
			if !isBearer {
				if refreshed, ok := a.refreshFromCookie(c); ok {
					claims = refreshed
				}
			}
		}

		if claims == nil {
			message := "Belum masuk"
			if raw != "" {
				message = "Sesi tidak valid"
				if errors.Is(verifyErr, token.ErrExpired) {
					message = "Sesi sudah berakhir, silakan masuk kembali"
				}
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": message})
			return
		}

		var user models.User
		err := a.DB.WithContext(c.Request.Context()).
			Preload("Role").
			First(&user, claims.UserID).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat akun"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Akun tidak ditemukan"})
			return
		}

		c.Set(ContextUserKey, &user)
		c.Next()
	}
}

// refreshFromCookie menerbitkan access token baru dari refresh cookie yang valid
// dan memperbarui cookie access pada response.
func (a *Auth) refreshFromCookie(c *gin.Context) (*token.Claims, bool) {
	if a.Refresh == nil || a.RefreshCookie == "" {
		return nil, false
	}

	refreshToken, err := c.Cookie(a.RefreshCookie)
	if err != nil || refreshToken == "" {
		return nil, false
	}

	ctx := c.Request.Context()
	data, err := a.Refresh.Get(ctx, refreshToken)
	if err != nil {
		return nil, false
	}

	_ = a.Refresh.Touch(ctx, refreshToken)

	raw, _, err := a.Tokens.IssueAccess(data.UserID, data.FamilyID)
	if err != nil {
		return nil, false
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(a.Cookie, raw, int(a.Tokens.AccessTTL().Seconds()), "/", "", a.Secure, true)

	return &token.Claims{UserID: data.UserID, SessionID: data.FamilyID}, true
}

// RequireLevels membatasi akses berdasarkan level role (mirror middleware role.level).
func RequireLevels(levels ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil || user.Role == nil || !slices.Contains(levels, user.Role.Level) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Anda tidak memiliki akses"})
			return
		}
		c.Next()
	}
}

// CurrentUser mengambil user terautentikasi dari context (nil jika tidak ada).
func CurrentUser(c *gin.Context) *models.User {
	value, ok := c.Get(ContextUserKey)
	if !ok {
		return nil
	}
	user, ok := value.(*models.User)
	if !ok {
		return nil
	}
	return user
}

// bearerToken mengambil token dari header Authorization: Bearer <token>.
func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
