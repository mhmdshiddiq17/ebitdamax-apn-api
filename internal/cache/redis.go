package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"agrinaspangan/ebitda-api/config"
)

// Connect membuat client Redis (dipakai untuk session store & cache).
func Connect() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     config.GetEnv("REDIS_HOST", "127.0.0.1") + ":" + config.GetEnv("REDIS_PORT", "6379"),
		Password: config.GetEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}

	log.Println("redis connected")
	return client
}
