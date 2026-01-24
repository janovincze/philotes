// Package main provides the entry point for the Philotes CLI tool.
// The CLI provides commands for managing CDC pipelines from the command line.
package main

import (
	"fmt"
	"os"

	"github.com/janovincze/philotes/internal/config"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	cmd := os.Args[1]
	switch cmd {
	case "version", "-v", "--version":
		fmt.Printf("philotes version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	case "status":
		return cmdStatus()
	case "pipelines":
		return cmdPipelines()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		return fmt.Errorf("unknown command: %s", cmd)
	}
	return nil
}

func printUsage() {
	fmt.Println(`Philotes CLI - CDC Pipeline Management

Usage:
  philotes <command> [options]

Commands:
  version     Show version information
  status      Show system status
  pipelines   List and manage pipelines
  help        Show this help message

Use "philotes <command> --help" for more information about a command.`)
}

func cmdStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Philotes Status\n")
	fmt.Printf("---------------\n")
	fmt.Printf("API URL: %s\n", cfg.API.BaseURL)
	fmt.Println("Status check not yet implemented")
	return nil
}

func cmdPipelines() error {
	fmt.Println("Pipeline management not yet implemented")
	return nil
}
