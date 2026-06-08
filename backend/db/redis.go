package db

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_ADDR")
	}
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	var client *redis.Client
	if strings.HasPrefix(redisAddr, "redis://") || strings.HasPrefix(redisAddr, "rediss://") {
		opts, err := redis.ParseURL(redisAddr)
		if err != nil {
			log.Fatalf("Critical: Failed to parse Redis URL: %v", err)
		}
		client = redis.NewClient(opts)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: "",
			DB:       0,
		})
	}

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Critical: Redis is unreachable. Error: %v", err)
	}

	RedisClient = client
	log.Println("✅ Redis (Cache & Rate Limiter) Connected")
}
