package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FahadAkash/100M-Systems/internal/api"
	"github.com/FahadAkash/100M-Systems/internal/config"
	"github.com/FahadAkash/100M-Systems/internal/middleware"
	"github.com/FahadAkash/100M-Systems/internal/shortener"
	"github.com/FahadAkash/100M-Systems/internal/storage"
	"github.com/FahadAkash/100M-Systems/pkg/events"
	"github.com/FahadAkash/100M-Systems/pkg/logger"
	"github.com/FahadAkash/100M-Systems/pkg/metrics"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	log := logger.New()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	store, rdb, cleanup, err := createStore(cfg, log)
	if err != nil {
		log.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	var analyticsStore storage.AnalyticsStore
	if as, ok := store.(storage.AnalyticsStore); ok {
		analyticsStore = as
	}

	var producer events.ClickProducer = &events.NoopProducer{}
	if cfg.KafkaEnabled {
		kp := events.NewKafkaProducer(cfg.KafkaBrokers, cfg.KafkaTopic, 5000)
		producer = kp
		defer func() {
			if err := kp.Close(); err != nil {
				log.Error("error closing kafka producer", "error", err)
			}
		}()
		log.Info("kafka click event producer initialized", "brokers", cfg.KafkaBrokers, "topic", cfg.KafkaTopic)

		if analyticsStore != nil {
			consumer := events.NewKafkaConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, analyticsStore.SaveClickBatch)
			consumer.Start(ctx)
			defer func() {
				if err := consumer.Close(); err != nil {
					log.Error("error closing kafka consumer", "error", err)
				}
			}()
			log.Info("kafka click event consumer worker started", "topic", cfg.KafkaTopic, "group_id", cfg.KafkaGroupID)
		}
	}

	service := shortener.NewService(store)
	handler := api.NewHandler(service, producer, analyticsStore, cfg.BaseURL, log)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shorten", handler.Shorten)
	mux.HandleFunc("GET /api/urls/{code}/analytics", handler.GetAnalytics)
	mux.HandleFunc("GET /api/urls/{code}", handler.GetURL)
	mux.HandleFunc("GET /{code}", handler.Redirect)
	mux.Handle("GET /metrics", metrics.Handler())

	var root http.Handler = mux
	if cfg.SimulateLatencyEnabled {
		sim := middleware.NewLatencySimulator(true, cfg.SimulateLatencyMinMs, cfg.SimulateLatencyMaxMs)
		root = sim.Middleware(root)
		log.Info("server latency simulation enabled", "min_ms", cfg.SimulateLatencyMinMs, "max_ms", cfg.SimulateLatencyMaxMs)
	}
	if cfg.RateLimitEnabled {
		limiter := middleware.NewRateLimiter(rdb, cfg.RateLimitMaxRequests, cfg.RateLimitWindowSeconds)
		root = limiter.Limit(root)
		log.Info("rate limiting enabled", "max_requests", cfg.RateLimitMaxRequests, "window_seconds", cfg.RateLimitWindowSeconds)
	}
	root = middleware.CORS(root)
	root = metrics.Middleware(root)
	root = middleware.Logging(log)(root)
	root = middleware.Recover(log)(root)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("server starting", "port", cfg.Port, "mode", cfg.StorageMode)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		_ = server.Close()
	}

	log.Info("server stopped")
}

func createStore(
	cfg config.Config,
	log *slog.Logger,
) (storage.Store, *redis.Client, func(), error) {
	noopCleanup := func() {}

	switch cfg.StorageMode {
	case "memory":
		log.Info("using in-memory storage")
		return storage.NewMemoryStore(), nil, noopCleanup, nil

	case "redis":
		rdb, err := initRedis(cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		log.Info("using Redis storage")
		return storage.NewRedisStore(rdb, 0), rdb, func() { _ = rdb.Close() }, nil

	case "postgres":
		db, err := initPostgres(cfg)
		if err != nil {
			return nil, nil, nil, err
		}

		// Optionally enable Redis cache if configured
		if cfg.RedisURL != "" {
			if rdb, err := initRedis(cfg); err == nil {
				log.Info("using PostgreSQL storage with Redis caching")
				cachedStore := storage.NewCachedStore(storage.NewPostgresStore(db), rdb, 24*time.Hour)
				return cachedStore, rdb, func() {
					_ = db.Close()
					_ = rdb.Close()
				}, nil
			} else {
				log.Warn("redis cache initialization failed, falling back to postgres only", "error", err)
			}
		}

		log.Info("using PostgreSQL storage")
		return storage.NewPostgresStore(db), nil, func() { _ = db.Close() }, nil

	case "cached":
		db, err := initPostgres(cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		rdb, err := initRedis(cfg)
		if err != nil {
			_ = db.Close()
			return nil, nil, nil, err
		}

		log.Info("using PostgreSQL storage with Redis cache layer")
		cachedStore := storage.NewCachedStore(storage.NewPostgresStore(db), rdb, 24*time.Hour)
		return cachedStore, rdb, func() {
			_ = db.Close()
			_ = rdb.Close()
		}, nil

	default:
		return nil, nil, nil, errors.New("unsupported STORAGE_MODE: " + cfg.StorageMode)
	}
}

func initPostgres(cfg config.Config) (*sql.DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required for postgres mode")
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	// Ensure database schema is migrated
	schema := `
	CREATE TABLE IF NOT EXISTS urls (
		id BIGSERIAL PRIMARY KEY,
		code VARCHAR(32) NOT NULL UNIQUE,
		original_url TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMPTZ
	);
	ALTER TABLE urls ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
	ALTER TABLE urls ALTER COLUMN code TYPE VARCHAR(32);
	CREATE INDEX IF NOT EXISTS idx_urls_code ON urls (code);
	CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls (expires_at);

	CREATE TABLE IF NOT EXISTS url_clicks (
		id BIGSERIAL PRIMARY KEY,
		code VARCHAR(32) NOT NULL,
		original_url TEXT NOT NULL,
		ip VARCHAR(64) NOT NULL DEFAULT '',
		user_agent TEXT NOT NULL DEFAULT '',
		referer TEXT NOT NULL DEFAULT '',
		clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_url_clicks_code ON url_clicks (code);
	CREATE INDEX IF NOT EXISTS idx_url_clicks_clicked_at ON url_clicks (clicked_at);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func initRedis(cfg config.Config) (*redis.Client, error) {
	if cfg.RedisURL == "" {
		return nil, errors.New("REDIS_URL or REDIS_HOST is required for redis mode")
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}

	return rdb, nil
}
