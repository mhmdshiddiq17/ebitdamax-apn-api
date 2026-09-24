package middleware

import (
	"errors"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/session"
)

// ContextUserKey adalah key user di gin.Context.
const ContextUserKey = "auth_user"

// Auth adalah middleware autentikasi berbasis session cookie + Redis.
type Auth struct {
	DB      *gorm.DB
	Session *session.Manager
	Cookie  string
}

// Required menolak request tanpa session valid dan menaruh user di context.
func (a *Auth) Required() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(a.Cookie)
		if err != nil || sessionID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
			return
		}

		data, err := a.Session.Get(c.Request.Context(), sessionID)
		if err != nil {
			if !errors.Is(err, session.ErrNotFound) {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Gagal membaca sesi"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Sesi tidak valid atau sudah berakhir"})
			return
		}

		var user models.User
		err = a.DB.WithContext(c.Request.Context()).
			Preload("Role").
			First(&user, data.UserID).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Gagal memuat akun"})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Akun tidak ditemukan"})
			return
		}

		if err := a.Session.Touch(c.Request.Context(), sessionID); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Gagal memperbarui sesi"})
			return
		}

		c.Set(ContextUserKey, &user)
		c.Next()
	}
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
