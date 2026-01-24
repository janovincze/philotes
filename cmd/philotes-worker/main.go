// Package main provides the entry point for the Philotes CDC Worker service.
// The worker handles Change Data Capture from PostgreSQL sources and writes
// data to Apache Iceberg tables via Lakekeeper.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/janovincze/philotes/internal/cdc/buffer"
	"github.com/janovincze/philotes/internal/cdc/checkpoint"
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
			SourceID:        fmt.Sprintf("postgres-%s", cfg.CDC.Source.Database),
			BatchSize:       cfg.CDC.BatchSize,
			FlushInterval:   cfg.CDC.FlushInterval,
			Retention:       cfg.CDC.Buffer.Retention,
			CleanupInterval: cfg.CDC.Buffer.CleanupInterval,
		}

		batchProcessor = buffer.NewBatchProcessor(
			bufferMgr,
			writer.BatchHandler(icebergWriter),
			batchCfg,
			logger,
		)

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
	}

	p := pipeline.New(reader, checkpointMgr, bufferMgr, pipelineCfg, logger)

	logger.Info("CDC pipeline configured",
		"source_host", cfg.CDC.Source.Host,
		"source_port", cfg.CDC.Source.Port,
		"source_database", cfg.CDC.Source.Database,
		"replication_slot", cfg.CDC.Replication.SlotName,
		"checkpoint_enabled", cfg.CDC.Checkpoint.Enabled,
		"checkpoint_interval", cfg.CDC.Checkpoint.Interval,
		"buffer_enabled", cfg.CDC.Buffer.Enabled,
		"iceberg_enabled", batchProcessor != nil,
	)

	if err := p.Run(ctx); err != nil {
		return fmt.Errorf("pipeline error: %w", err)
	}

	logger.Info("CDC worker stopped gracefully")
	return nil
}
