// Package main provides the entry point for the Philotes CDC Worker service.
// The worker handles Change Data Capture from PostgreSQL sources and writes
// data to Apache Iceberg tables via Lakekeeper.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/janovincze/philotes/internal/cdc/buffer"
	"github.com/janovincze/philotes/internal/cdc/checkpoint"
	"github.com/janovincze/philotes/internal/cdc/deadletter"
	"github.com/janovincze/philotes/internal/cdc/health"
	"github.com/janovincze/philotes/internal/cdc/pipeline"
	"github.com/janovincze/philotes/internal/cdc/source/postgres"
	"github.com/janovincze/philotes/internal/config"
	"github.com/janovincze/philotes/internal/iceberg/catalog"
	"github.com/janovincze/philotes/internal/iceberg/writer"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig.String())
		cancel()
	}()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("worker failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	logger.Info("starting Philotes CDC Worker",
		"version", cfg.Version,
		"environment", cfg.Environment,
	)

	// Create health manager
	healthMgr := health.NewManager(health.DefaultManagerConfig(), logger)

	// Start health server if enabled
	var healthServer *health.Server
	if cfg.CDC.Health.Enabled {
		healthServer = health.NewServer(healthMgr, health.ServerConfig{
			ListenAddr:   cfg.CDC.Health.ListenAddr,
			ReadTimeout:  cfg.CDC.Health.ReadinessTimeout,
			WriteTimeout: cfg.CDC.Health.ReadinessTimeout * 2,
		}, logger)

		go func() {
			if err := healthServer.Start(); err != nil && err != http.ErrServerClosed {
				logger.Error("health server failed", "error", err)
			}
		}()
		defer healthServer.Stop(context.Background())

		logger.Info("health server started", "addr", cfg.CDC.Health.ListenAddr)
	}

	// Create the PostgreSQL source reader
	readerCfg := postgres.Config{
		ConnectionURL:   cfg.CDC.Source.URL(),
		SlotName:        cfg.CDC.Replication.SlotName,
		PublicationName: cfg.CDC.Replication.PublicationName,
		Tables:          cfg.CDC.Replication.Tables,
		EventBufferSize: cfg.CDC.BufferSize,
	}
	readerCfg.Name = fmt.Sprintf("postgres-%s", cfg.CDC.Source.Database)

	reader, err := postgres.New(readerCfg, logger)
	if err != nil {
		return fmt.Errorf("create source reader: %w", err)
	}

	// Create the checkpoint manager
	var checkpointMgr checkpoint.Manager
	if cfg.CDC.Checkpoint.Enabled {
		cpCfg := checkpoint.PostgresConfig{
			DSN:          cfg.Database.DSN(),
			MaxOpenConns: cfg.Database.MaxOpenConns,
			MaxIdleConns: cfg.Database.MaxIdleConns,
		}

		checkpointMgr, err = checkpoint.NewPostgresManager(ctx, cpCfg, logger)
		if err != nil {
			return fmt.Errorf("create checkpoint manager: %w", err)
		}
		defer checkpointMgr.Close()
	}

	// Create the buffer manager
	var bufferMgr buffer.Manager
	var db *sql.DB
	if cfg.CDC.Buffer.Enabled {
		bufCfg := buffer.Config{
			Enabled:         true,
			DSN:             cfg.Database.DSN(),
			MaxOpenConns:    cfg.Database.MaxOpenConns,
			MaxIdleConns:    cfg.Database.MaxIdleConns,
			Retention:       cfg.CDC.Buffer.Retention,
			CleanupInterval: cfg.CDC.Buffer.CleanupInterval,
		}

		bufferMgr, err = buffer.NewPostgresManager(ctx, bufCfg, logger)
		if err != nil {
			return fmt.Errorf("create buffer manager: %w", err)
		}
		defer bufferMgr.Close()

		// Get the database connection for DLQ (reuse buffer manager's connection)
		db, err = sql.Open("postgres", cfg.Database.DSN())
		if err != nil {
			return fmt.Errorf("open database for DLQ: %w", err)
		}
		defer db.Close()

		// Register buffer health check
		healthMgr.Register(health.NewDatabaseChecker("buffer-database", func(ctx context.Context) error {
			return db.PingContext(ctx)
		}))
	}

	// Create the dead-letter queue manager if enabled
	var dlqMgr deadletter.Manager
	if cfg.CDC.DeadLetter.Enabled && db != nil {
		dlqCfg := deadletter.PostgresConfig{
			Retention: cfg.CDC.DeadLetter.Retention,
		}
		dlqMgr = deadletter.NewPostgresManager(db, dlqCfg, logger)
		logger.Info("dead-letter queue enabled", "retention", cfg.CDC.DeadLetter.Retention)
	}

	// Create the Iceberg writer and batch processor if buffering is enabled
	var batchProcessor *buffer.BatchProcessor
	if cfg.CDC.Buffer.Enabled && bufferMgr != nil {
		// Create Iceberg writer
		writerCfg := writer.Config{
			Catalog: catalog.Config{
				CatalogURL: cfg.Iceberg.CatalogURL,
				Warehouse:  cfg.Iceberg.Warehouse,
			},
			S3: writer.S3Config{
				Endpoint:  cfg.Storage.Endpoint,
				AccessKey: cfg.Storage.AccessKey,
				SecretKey: cfg.Storage.SecretKey,
				UseSSL:    cfg.Storage.UseSSL,
			},
			Bucket:           cfg.Storage.Bucket,
			WarehousePath:    "warehouse",
			DefaultNamespace: "cdc",
		}

		icebergWriter, err := writer.NewIcebergWriter(writerCfg, logger)
		if err != nil {
			return fmt.Errorf("create iceberg writer: %w", err)
		}
		defer icebergWriter.Close()

		// Create batch processor with Iceberg handler
		batchCfg := buffer.BatchConfig{
			SourceID:             fmt.Sprintf("postgres-%s", cfg.CDC.Source.Database),
			BatchSize:            cfg.CDC.BatchSize,
			FlushInterval:        cfg.CDC.FlushInterval,
			Retention:            cfg.CDC.Buffer.Retention,
			CleanupInterval:      cfg.CDC.Buffer.CleanupInterval,
			RetryMaxAttempts:     cfg.CDC.Retry.MaxAttempts,
			RetryInitialInterval: cfg.CDC.Retry.InitialInterval,
			RetryMaxInterval:     cfg.CDC.Retry.MaxInterval,
			RetryMultiplier:      cfg.CDC.Retry.Multiplier,
			DLQEnabled:           cfg.CDC.DeadLetter.Enabled,
			DLQRetention:         cfg.CDC.DeadLetter.Retention,
		}

		batchProcessor = buffer.NewBatchProcessor(
			bufferMgr,
			writer.BatchHandler(icebergWriter),
			batchCfg,
			logger,
		)

		// Set DLQ manager if enabled
		if dlqMgr != nil {
			batchProcessor.SetDeadLetterManager(dlqMgr)
		}

		// Start the batch processor
		if err := batchProcessor.Start(ctx); err != nil {
			return fmt.Errorf("start batch processor: %w", err)
		}
		defer batchProcessor.Stop(context.Background())

		logger.Info("Iceberg writer configured",
			"catalog_url", cfg.Iceberg.CatalogURL,
			"warehouse", cfg.Iceberg.Warehouse,
			"storage_endpoint", cfg.Storage.Endpoint,
			"bucket", cfg.Storage.Bucket,
		)
	}

	// Create and run the pipeline
	pipelineCfg := pipeline.Config{
		CheckpointInterval: cfg.CDC.Checkpoint.Interval,
		CheckpointEnabled:  cfg.CDC.Checkpoint.Enabled,
		BufferEnabled:      cfg.CDC.Buffer.Enabled,
		RetryPolicy: pipeline.RetryPolicy{
			MaxAttempts:     cfg.CDC.Retry.MaxAttempts,
			InitialInterval: cfg.CDC.Retry.InitialInterval,
			MaxInterval:     cfg.CDC.Retry.MaxInterval,
			Multiplier:      cfg.CDC.Retry.Multiplier,
			Jitter:          true,
		},
		BackpressureConfig: pipeline.BackpressureConfig{
			Enabled:       cfg.CDC.Backpressure.Enabled,
			HighWatermark: cfg.CDC.Backpressure.HighWatermark,
			LowWatermark:  cfg.CDC.Backpressure.LowWatermark,
			CheckInterval: cfg.CDC.Backpressure.CheckInterval,
		},
	}

	p := pipeline.New(reader, checkpointMgr, bufferMgr, pipelineCfg, logger)

	// Setup backpressure controller if enabled and buffer manager exists
	if cfg.CDC.Backpressure.Enabled && bufferMgr != nil {
		bpController := pipeline.NewBackpressureController(
			pipeline.BackpressureConfig{
				Enabled:       true,
				HighWatermark: cfg.CDC.Backpressure.HighWatermark,
				LowWatermark:  cfg.CDC.Backpressure.LowWatermark,
				CheckInterval: cfg.CDC.Backpressure.CheckInterval,
			},
			func(ctx context.Context) (int, error) {
				stats, err := bufferMgr.Stats(ctx)
				if err != nil {
					return 0, err
				}
				return int(stats.UnprocessedEvents), nil
			},
			nil, // state machine will be set internally
			logger,
		)
		p.SetBackpressureController(bpController)
		logger.Info("backpressure controller enabled",
			"high_watermark", cfg.CDC.Backpressure.HighWatermark,
			"low_watermark", cfg.CDC.Backpressure.LowWatermark,
		)
	}

	// Register pipeline health check
	healthMgr.Register(p.HealthChecker())

	logger.Info("CDC pipeline configured",
		"source_host", cfg.CDC.Source.Host,
		"source_port", cfg.CDC.Source.Port,
		"source_database", cfg.CDC.Source.Database,
		"replication_slot", cfg.CDC.Replication.SlotName,
		"checkpoint_enabled", cfg.CDC.Checkpoint.Enabled,
		"checkpoint_interval", cfg.CDC.Checkpoint.Interval,
		"buffer_enabled", cfg.CDC.Buffer.Enabled,
		"iceberg_enabled", batchProcessor != nil,
		"dlq_enabled", cfg.CDC.DeadLetter.Enabled,
		"health_enabled", cfg.CDC.Health.Enabled,
		"backpressure_enabled", cfg.CDC.Backpressure.Enabled,
	)

	if err := p.Run(ctx); err != nil {
		return fmt.Errorf("pipeline error: %w", err)
	}

	logger.Info("CDC worker stopped gracefully")
	return nil
}
