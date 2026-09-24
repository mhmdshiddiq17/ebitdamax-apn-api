package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
)

// CompleteOnboardingHandler godoc
//
//	@Summary      Selesaikan onboarding
//	@Description  Menandai onboarding manager KDKMP sebagai selesai (hanya role manager domain kdkmp).
//	@Tags         Users
//	@Produce      json
//	@Security     CookieAuth
//	@Success      200  {object}  map[string]any
//	@Failure      401  {object}  map[string]string
//	@Failure      403  {object}  map[string]string
//	@Router       /api/v1/users/complete-onboarding [post]
func CompleteOnboardingHandler(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Belum masuk"})
		return
	}

	if !user.IsKdkmpManager() {
		c.JSON(http.StatusForbidden, gin.H{"message": "Anda tidak memiliki akses"})
		return
	}

	if user.HasCompletedOnboarding {
		c.JSON(http.StatusOK, gin.H{"message": "Onboarding sudah selesai", "has_completed_onboarding": true})
		return
	}

	err := AppDeps.DB.WithContext(c.Request.Context()).
		Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("has_completed_onboarding", true).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan status onboarding"})
		return
	}

	user.HasCompletedOnboarding = true

	c.JSON(http.StatusOK, gin.H{"message": "Onboarding selesai", "has_completed_onboarding": true})
}
