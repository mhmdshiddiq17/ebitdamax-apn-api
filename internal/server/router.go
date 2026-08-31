package server

import (
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/config"

	_ "agrinaspangan/ebitda-api/docs"
)

// Deps berisi dependensi global yang dipakai handler.
type Deps struct {
	DB    *gorm.DB
	Redis *redis.Client
	Minio *minio.Client
}

// NewRouter merakit semua route aplikasi.
func NewRouter(deps Deps) *gin.Engine {
	if config.GetEnv("APP_ENV", "local") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	AppDeps = deps

	router := gin.Default()

	api := router.Group("/api/v1")
	{
		api.GET("/ping", PingHandler)
	}

	router.GET("/healthz", HealthzHandler)

	// Swagger UI: http://localhost:4000/swagger/index.html
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}
