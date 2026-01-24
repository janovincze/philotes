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

	"github.com/janovincze/philotes/internal/cdc/checkpoint"
	"github.com/janovincze/philotes/internal/cdc/pipeline"
	"github.com/janovincze/philotes/internal/cdc/source/postgres"
	"github.com/janovincze/philotes/internal/config"
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

	// Create and run the pipeline
	pipelineCfg := pipeline.Config{
		CheckpointInterval: cfg.CDC.Checkpoint.Interval,
		CheckpointEnabled:  cfg.CDC.Checkpoint.Enabled,
	}

	p := pipeline.New(reader, checkpointMgr, pipelineCfg, logger)

	logger.Info("CDC pipeline configured",
		"source_host", cfg.CDC.Source.Host,
		"source_port", cfg.CDC.Source.Port,
		"source_database", cfg.CDC.Source.Database,
		"replication_slot", cfg.CDC.Replication.SlotName,
		"checkpoint_enabled", cfg.CDC.Checkpoint.Enabled,
		"checkpoint_interval", cfg.CDC.Checkpoint.Interval,
	)

	if err := p.Run(ctx); err != nil {
		return fmt.Errorf("pipeline error: %w", err)
	}

	logger.Info("CDC worker stopped gracefully")
	return nil
}
