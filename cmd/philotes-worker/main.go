// Package main provides the entry point for the Philotes CDC Worker service.
// The worker handles Change Data Capture from PostgreSQL sources and writes
// data to Apache Iceberg tables via Lakekeeper.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/janovincze/philotes/internal/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config) error {
	fmt.Printf("Starting Philotes CDC Worker v%s\n", cfg.Version)
	fmt.Println("Worker initialization not yet implemented")

	// Wait for shutdown signal
	<-ctx.Done()
	return nil
}
