package db

import (
	"context"
	"log"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/config"
	gen "github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/db/gen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	DB  *pgxpool.Pool
	Ctx = context.Background()
	Q   *gen.Queries
)

func Connect(cfg config.Config) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DBURL)
	if err != nil {
		log.Fatalf("❌ Failed to parse DB config: %v", err)
	}

	// 🚫 Disable prepared statement caching
	poolConfig.ConnConfig.StatementCacheCapacity = 0

	// 🧩 Force simple protocol (no prepared statements)
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(Ctx, poolConfig)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v", err)
	}

	if err := pool.Ping(Ctx); err != nil {
		log.Fatalf("❌ Could not ping database: %v", err)
	}

	DB = pool
	Q = gen.New(pool)
	log.Println("✅ Connected to Postgres successfully (no prepared statement caching)")
}
func LocalConnect(cfg config.Config) {
	poolConfig, err := pgxpool.ParseConfig(cfg.LOCALDBURL)
	if err != nil {
		log.Fatalf("❌ Failed to parse DB config: %v", err)
	}

	// 🚫 Disable prepared statement caching
	poolConfig.ConnConfig.StatementCacheCapacity = 0

	// 🧩 Force simple protocol (no prepared statements)
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(Ctx, poolConfig)
	if err != nil {
		log.Fatalf("❌ Unable to connect to database: %v", err)
	}

	if err := pool.Ping(Ctx); err != nil {
		log.Fatalf("❌ Could not ping database: %v", err)
	}

	DB = pool
	Q = gen.New(pool)
	log.Println("✅ Connected to Postgres successfully (no prepared statement caching)")
}

func Close() {
	if DB != nil {
		DB.Close()
		log.Println("🛑 Database connection closed")
	}
}
