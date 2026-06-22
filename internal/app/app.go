package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"dbchangepakets/internal/config"
	"dbchangepakets/internal/repository/mongodb"
	"dbchangepakets/internal/repository/postgres"
	"dbchangepakets/internal/user"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Run bootstraps the database connections, sets up routing, and handles graceful shutdowns.
func Run(ctx context.Context, logger *slog.Logger) error {
	// 1. Load typed hierarchical configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var repo user.UserRepository
	var cleanups []func(context.Context) error

	// Run registered cleanups in LIFO order on function exit
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := cleanups[i](cleanupCtx); err != nil {
				logger.Error("cleanup failed", "index", i, "error", err)
			}
			cancel()
		}
	}()

	// 2. Select database and establish connection pool/client
	switch cfg.DBType {
	case "postgres":
		logger.Info("initializing postgres connection pool...", "uri", cfg.PG.URI)
		db, err := sql.Open("postgres", cfg.PG.URI)
		if err != nil {
			return fmt.Errorf("postgres open: %w", err)
		}

		db.SetMaxOpenConns(cfg.PG.MaxOpenConns)
		db.SetMaxIdleConns(cfg.PG.MaxIdleConns)
		db.SetConnMaxLifetime(time.Duration(cfg.PG.ConnMaxLifetime) * time.Minute)

		pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
		err = db.PingContext(pingCtx)
		pingCancel()
		if err != nil {
			_ = db.Close()
			return fmt.Errorf("postgres ping: %w", err)
		}

		cleanups = append(cleanups, func(c context.Context) error {
			logger.Info("closing postgres connection pool")
			return db.Close()
		})

		if err := postgres.RunMigrations(db); err != nil {
			return fmt.Errorf("postgres migrations: %w", err)
		}
		repo = postgres.NewUserRepository(db)

	case "mongodb":
		logger.Info("initializing mongodb connection...", "uri", cfg.Mongo.URI)
		clientOptions := options.Client().ApplyURI(cfg.Mongo.URI)

		connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
		client, err := mongo.Connect(connectCtx, clientOptions)
		connectCancel()
		if err != nil {
			return fmt.Errorf("mongo connect: %w", err)
		}

		pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
		err = client.Ping(pingCtx, nil)
		pingCancel()
		if err != nil {
			_ = client.Disconnect(ctx)
			return fmt.Errorf("mongo ping: %w", err)
		}

		cleanups = append(cleanups, func(c context.Context) error {
			logger.Info("disconnecting mongodb client")
			return client.Disconnect(c)
		})

		mongoRepo := mongodb.NewUserRepository(client, cfg.Mongo.DBName, "users")
		if err := mongoRepo.EnsureIndexes(ctx); err != nil {
			return fmt.Errorf("mongo indexes: %w", err)
		}
		repo = mongoRepo

	default:
		return fmt.Errorf("unsupported database type: %q", cfg.DBType)
	}

	// 3. Inject Repository into the Service and prepare Router
	userService := user.NewService(repo)
	mux := http.NewServeMux()
	
	userHandler := user.NewHandler(userService, logger)
	userHandler.RegisterRoutes(mux)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 4. Start Server blockingly with graceful listener handler
	srvErrChan := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErrChan <- err
		}
	}()

	// 5. Monitor termination signals
	select {
	case err := <-srvErrChan:
		return fmt.Errorf("http server failure: %w", err)
	case <-ctx.Done():
		logger.Info("initiating graceful shutdown of http server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}

	return nil
}
