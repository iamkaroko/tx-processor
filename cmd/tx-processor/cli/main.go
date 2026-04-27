// cmd/tx-processor/cli/main.go
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	icache "tx-processor/internal/cache"
	rds "tx-processor/internal/cache/redis"
	"tx-processor/internal/config"
	"tx-processor/internal/db"
	"tx-processor/internal/processor"
	"tx-processor/internal/repository"
	"tx-processor/internal/service"

	"github.com/redis/go-redis/v9"
)

const (
	defaultWorkers       = 10
	defaultBatchSize     = 500
	defaultChannelBuffer = 10000
)

func main() {
	filePath := flag.String("file", "", "Path to the NDJSON file (required)")
	workerCount := flag.Int("workers", defaultWorkers, "Number of concurrent workers")
	batchSize := flag.Int("batch", defaultBatchSize, "Batch size per worker flush")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Usage: cli -file=sample_transactions.json [-workers=10] [-batch=500]")
		os.Exit(1)
	}

	if err := processFile(*filePath, *workerCount, *batchSize); err != nil {
		log.Fatal(err)
	}
}

func processFile(filePath string, workerCount, batchSize int) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("starting transaction processor",
		"file", filePath, "workers", workerCount, "batch_size", batchSize)

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbConn, err := db.NewPostgresDB(&cfg.DatabaseConfig)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer dbConn.Close()

	repo := repository.NewAnalyticsRepo(dbConn)

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

	analyticsSvc := service.NewAnalytics(repo, analyticsCache)
	proc := processor.New(logger)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		defer signal.Stop(sig)
		<-sig
		logger.Warn("interrupt received, shutting down...")
		cancel()
	}()

	lines := make(chan string, defaultChannelBuffer)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var wg sync.WaitGroup
	start := time.Now()
	totalLines := 0

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := proc.ProcessStream(ctx, id, lines, batchSize); err != nil {
				logger.Error("worker failed", "id", id, "error", err)
				cancel()
			}
		}(i)
	}

FeedLoop:
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			break FeedLoop
		default:
			lines <- scanner.Text()
			totalLines++
		}
	}
	close(lines)
	wg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	logger.Info("flushing to database...")
	if err := analyticsSvc.UpdateAnalytics(ctx, proc.Snapshot()); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	elapsed := time.Since(start).Seconds()
	snapshot := proc.Snapshot()

	logger.Info("processing complete",
		"transactions", totalLines,
		"elapsed_sec", fmt.Sprintf("%.3f", elapsed),
		"throughput_tps", fmt.Sprintf("%.0f", float64(totalLines)/elapsed),
		"unique_users", len(snapshot))

	return scanner.Err()
}
