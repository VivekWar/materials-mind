package db

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func InitPostgres() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("POSTGRES_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://postgres:password123@localhost:5432/materialmind"
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Critical: Failed to parse Postgres config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Critical: Failed to initialize Postgres pool: %v", err)
	}

	if err = pool.Ping(context.Background()); err != nil {
		log.Fatalf("Critical: Postgres is unreachable. Is Docker running? Error: %v", err)
	}

	Pool = pool
	log.Println("✅ Postgres (HNSW Vector DB) Connected")
}
