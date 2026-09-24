package main

import (
	"log"
	"strings"
	"time"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/cache"
	"agrinaspangan/ebitda-api/internal/crypto"
	"agrinaspangan/ebitda-api/internal/database"
	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/passkey"
	"agrinaspangan/ebitda-api/internal/server"
	"agrinaspangan/ebitda-api/internal/session"
	"agrinaspangan/ebitda-api/internal/storage"
	"agrinaspangan/ebitda-api/internal/taskreport"
	"agrinaspangan/ebitda-api/internal/twofactor"
)

// @title                     EBITDA Max APN API
// @version                   1.0
// @description               REST API aplikasi EBITDA Max APN (refactor Go + Next.js) — scope manager KDKMP & superadmin.
// @host                      localhost:4000
// @schemes                   http
// @securityDefinitions.apikey CookieAuth
// @in                        cookie
// @name                      ebitda_session
func main() {
	config.LoadEnv()

	redisClient := cache.Connect()
	db := database.Connect()

	sessionTTL := time.Duration(config.GetEnvInt("SESSION_TTL_HOURS", 168)) * time.Hour
	sessionCookie := config.GetEnv("SESSION_COOKIE", "ebitda_session")
	sessionSecure := config.GetEnv("APP_ENV", "local") == "production"
	sessionManager := session.NewManager(redisClient, sessionTTL)

	appKey := crypto.KeyFromSecret(config.GetEnv("APP_KEY", "dev-app-key-change-me"))

	passkeyService, err := passkey.NewService(db, sessionManager, passkey.Config{
		RPID:          config.GetEnv("WEBAUTHN_RP_ID", "localhost"),
		RPDisplayName: config.GetEnv("WEBAUTHN_RP_DISPLAY_NAME", "EBITDA Max APN"),
		RPOrigins:     splitOrigins(config.GetEnv("WEBAUTHN_RP_ORIGINS", "http://localhost:3000")),
	})
	if err != nil {
		log.Fatalf("failed to initialize webauthn: %v", err)
	}

	minioClient := storage.Connect()
	files := storage.NewFiles(minioClient, config.GetEnv("MINIO_BUCKET", "ebitdamax"))

	deps := server.Deps{
		DB:            db,
		Redis:         redisClient,
		Minio:         minioClient,
		Files:         files,
		Session:       sessionManager,
		TwoFactor:     twofactor.NewService(db, sessionManager, appKey, "EBITDA Max APN"),
		Passkey:       passkeyService,
		Selection:     kdkmp.NewSelectionService(db),
		Allocation:    kdkmp.NewAllocationService(db),
		TaskReports:   taskreport.NewDocumentService(files),
		SessionCookie: sessionCookie,
		SessionTTL:    sessionTTL,
		SessionSecure: sessionSecure,
		CORSOrigins:   splitOrigins(config.GetEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	router := server.NewRouter(deps)

	port := config.GetEnv("APP_PORT", "4000")
	log.Printf("api server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func splitOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
