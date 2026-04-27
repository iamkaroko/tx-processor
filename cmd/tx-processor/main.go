// cmd/tx-processor/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	icache "tx-processor/internal/cache"
	rds "tx-processor/internal/cache/redis"
	"tx-processor/internal/config"
	"tx-processor/internal/db"
	"tx-processor/internal/handler"
	"tx-processor/internal/repository"
	"tx-processor/internal/server"
	"tx-processor/internal/service"

	"github.com/redis/go-redis/v9"
)

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.New()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	database, err := db.NewPostgresDB(&cfg.DatabaseConfig)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer database.Close()

	repo := repository.NewAnalyticsRepo(database)

	var analyticsCache icache.AnalyticsCache
	if cfg.RedisConfig.RedisEnabled {
		redisClient := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisConfig.RedisAddr,
			Password: cfg.RedisConfig.RedisPw,
			DB:       cfg.RedisConfig.RedisDB,
		})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		defer redisClient.Close()
		analyticsCache = rds.NewRedisAnalyticsCache(redisClient)
	}

	svc := service.NewAnalytics(repo, analyticsCache)
	h := handler.New(svc, logger)
	srv := server.New(cfg.Port, h, logger)

	return srv.Start(ctx)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
