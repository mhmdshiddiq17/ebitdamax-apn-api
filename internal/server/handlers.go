package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"agrinaspangan/ebitda-api/config"
)

// AppDeps menyimpan dependensi global yang di-set saat router dibuat.
var AppDeps Deps

// PingHandler godoc
//
//	@Summary      Ping
//	@Description  Cek apakah API hidup
//	@Tags         Health
//	@Produce      json
//	@Success      200  {object}  map[string]string
//	@Router       /api/v1/ping [get]
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// HealthzHandler godoc
//
//	@Summary      Health check
//	@Description  Status koneksi database, redis, dan minio
//	@Tags         Health
//	@Produce      json
//	@Success      200  {object}  map[string]string
//	@Failure      503  {object}  map[string]string
//	@Router       /healthz [get]
func HealthzHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	status := gin.H{"status": "ok"}

	sqlDB, err := AppDeps.DB.DB()
	if err == nil {
		err = sqlDB.PingContext(ctx)
	}
	status["database"] = boolToStatus(err == nil)

	err = AppDeps.Redis.Ping(ctx).Err()
	status["redis"] = boolToStatus(err == nil)

	_, err = AppDeps.Minio.BucketExists(ctx, config.GetEnv("MINIO_BUCKET", "ebitdamax"))
	status["minio"] = boolToStatus(err == nil)

	if status["database"] == "down" || status["redis"] == "down" || status["minio"] == "down" {
		status["status"] = "degraded"
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}

	c.JSON(http.StatusOK, status)
}

func boolToStatus(up bool) string {
	if up {
		return "up"
	}
	return "down"
}
