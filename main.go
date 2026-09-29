package main

import (
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/cache"
	"agrinaspangan/ebitda-api/internal/crypto"
	"agrinaspangan/ebitda-api/internal/database"
	"agrinaspangan/ebitda-api/internal/kdkmp"
	"agrinaspangan/ebitda-api/internal/lark"
	"agrinaspangan/ebitda-api/internal/meetingminutes"
	"agrinaspangan/ebitda-api/internal/passkey"
	"agrinaspangan/ebitda-api/internal/sarpras"
	"agrinaspangan/ebitda-api/internal/server"
	"agrinaspangan/ebitda-api/internal/session"
	"agrinaspangan/ebitda-api/internal/storage"
	"agrinaspangan/ebitda-api/internal/taskreport"
	"agrinaspangan/ebitda-api/internal/token"
	"agrinaspangan/ebitda-api/internal/twofactor"
)

// @title                     EBITDA Max APN API
// @version                   1.0
// @description               REST API aplikasi EBITDA Max APN (refactor Go + Next.js) — scope manager KDKMP & superadmin.
// @host                      localhost:4000
// @schemes                   http
// @securityDefinitions.apikey CookieAuth
// @in                        cookie
// @name                      ebitda_access
// @securityDefinitions.apikey BearerAuth
// @in                        header
// @name                      Authorization
// @description               Isi dengan: Bearer {access_token}
func main() {
	config.LoadEnv()

	redisClient := cache.Connect()
	db := database.Connect()

	refreshTTL := time.Duration(config.GetEnvInt("REFRESH_TTL_HOURS", 168)) * time.Hour
	accessCookie := config.GetEnv("ACCESS_COOKIE", "ebitda_access")
	refreshCookie := config.GetEnv("REFRESH_COOKIE", "ebitda_refresh")
	sessionSecure := config.GetEnv("APP_ENV", "local") == "production"
	sessionManager := session.NewManager(redisClient, refreshTTL)
	refreshStore := session.NewRefreshStore(redisClient, refreshTTL)

	tokenService := token.NewService(
		config.GetEnv("JWT_SECRET", "dev-jwt-secret-change-me"),
		config.GetEnv("JWT_ISSUER", "ebitda-max-apn"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 60))*time.Minute,
	)

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
	larkClient, err := lark.NewClient(lark.Config{
		Enabled:     config.GetEnvBool("LARK_SSO_ENABLED", false),
		AppID:       config.GetEnv("LARK_APP_ID", ""),
		AppSecret:   config.GetEnv("LARK_APP_SECRET", ""),
		BaseURL:     config.GetEnv("LARK_BASE_URL", "https://open.larksuite.com"),
		AuthURL:     config.GetEnv("LARK_AUTHORIZATION_URL", "https://accounts.larksuite.com/open-apis/authen/v1/authorize"),
		RedirectURI: config.GetEnv("LARK_REDIRECT_URI", "http://localhost:4000/api/v1/auth/lark/callback"),
		FrontendURL: config.GetEnv("LARK_FRONTEND_URL", "http://localhost:3000"),
		Scopes:      config.GetEnv("LARK_SCOPES", "contact:user.email:readonly"),
	})
	if err != nil {
		log.Fatalf("failed to initialize Lark SSO: %v", err)
	}

	deps := server.Deps{
		DB:            db,
		Redis:         redisClient,
		Minio:         minioClient,
		Files:         files,
		Session:       sessionManager,
		Refresh:       refreshStore,
		Tokens:        tokenService,
		TwoFactor:     twofactor.NewService(db, sessionManager, appKey, "EBITDA Max APN"),
		Passkey:       passkeyService,
		Selection:     kdkmp.NewSelectionService(db),
		Allocation:    kdkmp.NewAllocationService(db),
		TaskReports:   taskreport.NewDocumentService(files),
		Meetings:      meetingminutes.NewService(db, files),
		Lark:          larkClient,
		AccessCookie:  accessCookie,
		RefreshCookie: refreshCookie,
		SessionSecure: sessionSecure,
		CORSOrigins:   splitOrigins(config.GetEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	router := server.NewRouter(deps)

	startSarprasScheduler(db)

	port := config.GetEnv("APP_PORT", "4000")
	log.Printf("api server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

// startSarprasScheduler mengaktifkan sinkronisasi titik sarpras berkala bila
// SARPRAS_SCHEDULER_ENABLED=true dan token portal tersedia.
func startSarprasScheduler(db *gorm.DB) {
	if !config.GetEnvBool("SARPRAS_SCHEDULER_ENABLED", false) {
		return
	}

	interval := config.GetEnv("SARPRAS_SYNC_INTERVAL", "@every 15m")
	token := config.GetEnv("PORTAL_PEMBANGUNAN_SARPRAS_TOKEN", "")
	if token == "" {
		log.Print("scheduler sarpras tidak aktif: PORTAL_PEMBANGUNAN_SARPRAS_TOKEN kosong")
		return
	}

	baseURL := config.GetEnv("PORTAL_PEMBANGAN_BASE_URL", "https://portalkdkmp.id")
	service := sarpras.NewSyncService(db, sarpras.NewClient(baseURL, token))
	if _, err := sarpras.StartScheduler(db, service, interval, log.Default()); err != nil {
		log.Printf("gagal memulai scheduler sarpras: %v", err)
		return
	}

	log.Printf("scheduler sarpras aktif (%s)", interval)
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
