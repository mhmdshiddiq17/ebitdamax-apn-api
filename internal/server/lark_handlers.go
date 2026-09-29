package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"agrinaspangan/ebitda-api/internal/lark"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/session"
)

const (
	larkBrowserCookie = "ebitda_lark_browser"
	larkH5Cookie      = "ebitda_lark_h5"
	larkBrowserTTL    = 5 * time.Minute
	larkH5TTL         = 3 * time.Minute
)

type larkH5Request struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// LarkConfigHandler godoc
//
//	@Summary      Konfigurasi publik SSO Lark dan state H5 sekali pakai
//	@Tags         Auth
//	@Produce      json
//	@Success      200  {object}  map[string]any
//	@Router       /api/v1/auth/lark/config [get]
func LarkConfigHandler(c *gin.Context) {
	if AppDeps.Lark == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}
	state, err := session.NewToken()
	if err != nil || AppDeps.Redis.Set(c.Request.Context(), "lark:h5:"+state, "1", larkH5TTL).Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai login Lark"})
		return
	}
	setLarkCookie(c, larkH5Cookie, state, int(larkH5TTL.Seconds()))
	c.JSON(http.StatusOK, gin.H{
		"enabled": true, "app_id": AppDeps.Lark.Config.AppID,
		"scopes": strings.Fields(AppDeps.Lark.Config.Scopes), "state": state,
	})
}

// LarkRedirectHandler godoc
//
//	@Summary      Mulai OAuth Lark di browser
//	@Tags         Auth
//	@Success      302
//	@Router       /api/v1/auth/lark/redirect [get]
func LarkRedirectHandler(c *gin.Context) {
	if AppDeps.Lark == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "SSO Lark belum dikonfigurasi"})
		return
	}
	state, err := session.NewToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai login Lark"})
		return
	}
	verifier, err := session.NewToken()
	if err != nil || AppDeps.Redis.Set(c.Request.Context(), "lark:browser:"+state, verifier, larkBrowserTTL).Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memulai login Lark"})
		return
	}
	challenge := sha256.Sum256([]byte(verifier))
	setLarkCookie(c, larkBrowserCookie, state, int(larkBrowserTTL.Seconds()))
	c.Redirect(http.StatusFound, AppDeps.Lark.AuthorizationURL(state, base64.RawURLEncoding.EncodeToString(challenge[:])))
}

// LarkCallbackHandler godoc
//
//	@Summary      Selesaikan OAuth Lark di browser
//	@Tags         Auth
//	@Success      302
//	@Router       /api/v1/auth/lark/callback [get]
func LarkCallbackHandler(c *gin.Context) {
	if AppDeps.Lark == nil {
		failLarkRedirect(c)
		return
	}
	state := c.Query("state")
	verifier, ok := consumeLarkState(c, larkBrowserCookie, "lark:browser:", state)
	if !ok || c.Query("error") != "" || c.Query("code") == "" {
		failLarkRedirect(c)
		return
	}
	identity, err := AppDeps.Lark.ExchangeBrowser(c.Request.Context(), c.Query("code"), verifier)
	if err != nil || completeLarkLogin(c, identity) != nil {
		failLarkRedirect(c)
		return
	}
	c.Redirect(http.StatusFound, strings.TrimRight(AppDeps.Lark.Config.FrontendURL, "/")+"/dashboard")
}

// LarkH5Handler godoc
//
//	@Summary      Selesaikan login otomatis dari klien Lark
//	@Tags         Auth
//	@Accept       json
//	@Produce      json
//	@Param        payload  body      larkH5Request  true  "Kode dan state H5"
//	@Success      200      {object}  map[string]any
//	@Failure      401      {object}  map[string]string
//	@Router       /api/v1/auth/lark/h5 [post]
func LarkH5Handler(c *gin.Context) {
	if AppDeps.Lark == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "SSO Lark belum dikonfigurasi"})
		return
	}
	var req larkH5Request
	if err := c.ShouldBindJSON(&req); err != nil {
		failLarkH5(c)
		return
	}
	if _, ok := consumeLarkState(c, larkH5Cookie, "lark:h5:", req.State); !ok {
		failLarkH5(c)
		return
	}
	identity, err := AppDeps.Lark.ExchangeH5(c.Request.Context(), req.Code)
	if err != nil || completeLarkLogin(c, identity) != nil {
		failLarkH5(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Berhasil masuk"})
}

func consumeLarkState(c *gin.Context, cookieName, prefix, state string) (string, bool) {
	cookie, _ := c.Cookie(cookieName)
	setLarkCookie(c, cookieName, "", -1)
	if len(state) != 64 || len(cookie) != 64 || subtle.ConstantTimeCompare([]byte(state), []byte(cookie)) != 1 {
		return "", false
	}
	value, err := AppDeps.Redis.GetDel(c.Request.Context(), prefix+state).Result()
	return value, err == nil && value != ""
}

func setLarkCookie(c *gin.Context, name, value string, age int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, age, "/api/v1/auth/lark", "", AppDeps.SessionSecure, true)
}

func failLarkRedirect(c *gin.Context) {
	clearAuthCookies(c)
	setLarkCookie(c, larkBrowserCookie, "", -1)
	frontend := "http://localhost:3000"
	if AppDeps.Lark != nil {
		frontend = strings.TrimRight(AppDeps.Lark.Config.FrontendURL, "/")
	}
	c.Redirect(http.StatusFound, frontend+"/login?error="+url.QueryEscape("Login Lark gagal atau akun tidak memiliki akses."))
}

func failLarkH5(c *gin.Context) {
	clearAuthCookies(c)
	setLarkCookie(c, larkH5Cookie, "", -1)
	c.JSON(http.StatusUnauthorized, gin.H{"message": "Login Lark gagal atau akun tidak memiliki akses."})
}

func completeLarkLogin(c *gin.Context, identity lark.Identity) error {
	user, err := resolveLarkUser(c, identity)
	if err != nil {
		return err
	}
	return issueAuthSession(c, user.ID)
}

func resolveLarkUser(c *gin.Context, identity lark.Identity) (*models.User, error) {
	if identity.OpenID == "" || identity.Email == "" {
		return nil, lark.ErrIdentity
	}
	var user models.User
	err := AppDeps.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		err := tx.Preload("Role").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("lark_open_id = ?", identity.OpenID).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var matches []models.User
			err = tx.Preload("Role").Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("LOWER(email) = ?", identity.Email).Limit(2).Find(&matches).Error
			if err == nil && len(matches) != 1 {
				return errors.New("email akun tidak unik atau tidak ditemukan")
			}
			if err == nil {
				user = matches[0]
			}
		}
		if err != nil {
			return err
		}
		if !canBindLark(&user, identity) {
			return errors.New("akun Lark tidak diizinkan")
		}
		updates := map[string]any{}
		if user.LarkOpenID == nil {
			updates["lark_open_id"] = identity.OpenID
		}
		if user.EmailVerifiedAt == nil {
			updates["email_verified_at"] = time.Now()
		}
		if len(updates) > 0 {
			return tx.Model(&user).Updates(updates).Error
		}
		return nil
	})
	return &user, err
}

func canBindLark(user *models.User, identity lark.Identity) bool {
	return user.IsKdkmpManager() && strings.EqualFold(user.Email, identity.Email) &&
		(user.LarkOpenID == nil || *user.LarkOpenID == identity.OpenID)
}

// ResetLarkIdentityHandler godoc
//
//	@Summary      Lepas koneksi Lark milik user KDKMP
//	@Tags         Users
//	@Security     CookieAuth
//	@Param        id  path  int  true  "ID user"
//	@Success      200  {object}  map[string]string
//	@Router       /api/v1/users/{id}/lark-identity [delete]
func ResetLarkIdentityHandler(c *gin.Context) {
	user, ok := findKdkmpUser(c)
	if !ok {
		return
	}
	if err := AppDeps.DB.WithContext(c.Request.Context()).Model(user).Update("lark_open_id", nil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal melepas koneksi Lark"})
		return
	}
	if err := AppDeps.Refresh.RevokeUser(c.Request.Context(), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Koneksi Lark dilepas, namun sesi belum dicabut"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Koneksi Lark berhasil dilepas"})
}
