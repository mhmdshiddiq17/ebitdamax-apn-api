package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/middleware"
	"agrinaspangan/ebitda-api/internal/models"
	"agrinaspangan/ebitda-api/internal/passkey"
	"agrinaspangan/ebitda-api/internal/session"
	"agrinaspangan/ebitda-api/internal/storage"
	"agrinaspangan/ebitda-api/internal/twofactor"

	_ "agrinaspangan/ebitda-api/docs"
)

// Deps berisi dependensi global yang dipakai handler.
type Deps struct {
	DB            *gorm.DB
	Redis         *redis.Client
	Minio         *minio.Client
	Files         *storage.Files
	Session       *session.Manager
	TwoFactor     *twofactor.Service
	Passkey       *passkey.Service
	SessionCookie string
	SessionTTL    time.Duration
	SessionSecure bool
	CORSOrigins   []string
}

// NewRouter merakit semua route aplikasi.
func NewRouter(deps Deps) *gin.Engine {
	if config.GetEnv("APP_ENV", "local") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	AppDeps = deps

	router := gin.Default()
	router.Use(middleware.CORS(deps.CORSOrigins))

	auth := &middleware.Auth{
		DB:      deps.DB,
		Session: deps.Session,
		Cookie:  deps.SessionCookie,
	}

	api := router.Group("/api/v1")
	{
		api.GET("/ping", PingHandler)

		api.POST("/auth/login", LoginHandler)
		api.POST("/auth/logout", LogoutHandler)
		api.POST("/auth/two-factor-challenge", TwoFactorChallengeHandler)
		api.POST("/auth/passkey/options", BeginPasskeyLoginHandler)
		api.POST("/auth/passkey/login", FinishPasskeyLoginHandler)

		protected := api.Group("")
		protected.Use(auth.Required())
		{
			protected.GET("/auth/me", MeHandler)
			protected.PATCH("/profile", UpdateProfileHandler)
			protected.PUT("/profile/password", UpdatePasswordHandler)
			protected.GET("/two-factor", TwoFactorStatusHandler)
			protected.POST("/two-factor/enable", TwoFactorEnableHandler)
			protected.POST("/two-factor/confirm", TwoFactorConfirmHandler)
			protected.POST("/two-factor/recovery-codes", TwoFactorRecoveryCodesHandler)
			protected.DELETE("/two-factor", TwoFactorDisableHandler)
			protected.GET("/passkeys", ListPasskeysHandler)
			protected.POST("/passkeys/register/options", BeginPasskeyRegistrationHandler)
			protected.POST("/passkeys/register", FinishPasskeyRegistrationHandler)
			protected.DELETE("/passkeys/:id", DeletePasskeyHandler)
			protected.POST("/users/complete-onboarding", CompleteOnboardingHandler)
			protected.GET("/users/:id/manager-sk-document", PreviewManagerSKDocumentHandler)
		}

		admin := protected.Group("")
		admin.Use(middleware.RequireLevels(models.RoleLevelSuperadmin))
		{
			admin.GET("/roles", ListRolesHandler)
			admin.POST("/roles", CreateRoleHandler)
			admin.PUT("/roles/:id", UpdateRoleHandler)
			admin.DELETE("/roles/:id", DeleteRoleHandler)

			admin.GET("/users", ListUsersHandler)
			admin.POST("/users", CreateUserHandler)
			admin.PUT("/users/:id", UpdateUserHandler)
			admin.DELETE("/users/:id", DeleteUserHandler)
			admin.POST("/users/:id/manager-sk-document", UploadManagerSKDocumentHandler)

			admin.GET("/region-options", RegionOptionsHandler)
			admin.GET("/kdkmp-options", KdkmpOptionsHandler)
		}
	}

	router.GET("/healthz", HealthzHandler)

	// Swagger UI: http://localhost:4000/swagger/index.html
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
