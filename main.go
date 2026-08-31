package main

import (
	"log"

	"agrinaspangan/ebitda-api/config"
	"agrinaspangan/ebitda-api/internal/cache"
	"agrinaspangan/ebitda-api/internal/database"
	"agrinaspangan/ebitda-api/internal/server"
	"agrinaspangan/ebitda-api/internal/storage"
)

// @title         EBITDA Max APN API
// @version       1.0
// @description   REST API aplikasi EBITDA Max APN (refactor Go + Next.js). Semua endpoint bisnis berada di bawah /api/v1.
// @host          localhost:4000
// @schemes       http
func main() {
	config.LoadEnv()

	deps := server.Deps{
		DB:    database.Connect(),
		Redis: cache.Connect(),
		Minio: storage.Connect(),
	}

	router := server.NewRouter(deps)

	port := config.GetEnv("APP_PORT", "4000")
	log.Printf("api server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
