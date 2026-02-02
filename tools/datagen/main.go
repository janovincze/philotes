// Package main provides a CLI tool for generating continuous test data
// in the Philotes e-commerce sample database.
//
// Usage:
//
//	go run tools/datagen/main.go --rate 10 --duration 60s
//	go run tools/datagen/main.go --rate 5 --count 100
//	go run tools/datagen/main.go --mode mixed --rate 20
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

// Configuration flags
var (
	host     = flag.String("host", getEnv("PHILOTES_SOURCE_HOST", "localhost"), "Database host")
	port     = flag.String("port", getEnv("PHILOTES_SOURCE_PORT", "5433"), "Database port")
	user     = flag.String("user", getEnv("PHILOTES_SOURCE_USER", "source"), "Database user")
	password = flag.String("password", getEnv("PHILOTES_SOURCE_PASSWORD", "source"), "Database password")
	dbname   = flag.String("dbname", getEnv("PHILOTES_SOURCE_DB", "source"), "Database name")
	rate     = flag.Int("rate", 10, "Operations per second")
	duration = flag.Duration("duration", 0, "Duration to run (0 = unlimited)")
	count    = flag.Int("count", 0, "Total operations to perform (0 = unlimited)")
	mode     = flag.String("mode", "insert", "Operation mode: insert, update, delete, mixed")
	verbose  = flag.Bool("verbose", false, "Verbose output")
)

// Counters for statistics
var (
	insertCount int64
	updateCount int64
	deleteCount int64
	errorCount  int64
)

func main() {
	flag.Parse()

	// Validate mode
	validModes := map[string]bool{"insert": true, "update": true, "delete": true, "mixed": true}
	if !validModes[*mode] {
		log.Fatalf("Invalid mode: %s. Valid modes: insert, update, delete, mixed", *mode)
	}

	// Connect to database
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		*host, *port, *user, *password, *dbname,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Connected to database at %s:%s/%s", *host, *port, *dbname)
	log.Printf("Mode: %s, Rate: %d ops/sec", *mode, *rate)
	if *duration > 0 {
		log.Printf("Duration: %s", *duration)
	}
	if *count > 0 {
		log.Printf("Count: %d operations", *count)
	}

	// Setup context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\nShutting down...")
		cancel()
	}()

	// Apply duration limit if set
	if *duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

	// Start generating data
	generator := NewDataGenerator(db)
	startTime := time.Now()

	ticker := time.NewTicker(time.Second / time.Duration(*rate))
	defer ticker.Stop()

	var totalOps int64

	for {
		select {
		case <-ctx.Done():
			printStats(startTime)
			return
		case <-ticker.C:
			if *count > 0 && totalOps >= int64(*count) {
				printStats(startTime)
				return
			}

			totalOps++
			go func() {
				var opErr error
				switch *mode {
				case "insert":
					opErr = generator.Insert(ctx)
				case "update":
					opErr = generator.Update(ctx)
				case "delete":
					opErr = generator.Delete(ctx)
				case "mixed":
					opErr = generator.Mixed(ctx)
				}

				if opErr != nil {
					atomic.AddInt64(&errorCount, 1)
					if *verbose {
						log.Printf("Error: %v", opErr)
					}
				}
			}()

			// Print periodic stats
			if totalOps%100 == 0 {
				printStats(startTime)
			}
		}
	}
}

// DataGenerator generates random data for the e-commerce database
type DataGenerator struct {
	db         *sql.DB
	firstNames []string
	lastNames  []string
	domains    []string
}

// NewDataGenerator creates a new data generator
func NewDataGenerator(db *sql.DB) *DataGenerator {
	return &DataGenerator{
		db: db,
		firstNames: []string{
			"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda",
			"William", "Elizabeth", "David", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
			"Thomas", "Sarah", "Charles", "Karen", "Christopher", "Nancy", "Daniel", "Lisa",
		},
		lastNames: []string{
			"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
			"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
			"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
		},
		domains: []string{
			"example.com", "test.com", "sample.org", "demo.net", "fake.io",
		},
	}
}

// Insert creates a new random customer
func (g *DataGenerator) Insert(ctx context.Context) error {
	firstName := g.firstNames[rand.Intn(len(g.firstNames))]
	lastName := g.lastNames[rand.Intn(len(g.lastNames))]
	domain := g.domains[rand.Intn(len(g.domains))]
	email := fmt.Sprintf("%s.%s.%d@%s", firstName, lastName, rand.Int63(), domain)
	phone := fmt.Sprintf("+1-555-%04d", rand.Intn(10000))
	loyaltyTier := []string{"bronze", "silver", "gold"}[rand.Intn(3)]
	metadata := fmt.Sprintf(`{"loyalty_tier": %q, "generated": true}`, loyaltyTier)

	_, err := g.db.ExecContext(ctx, `
		INSERT INTO customers (first_name, last_name, email, phone, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, firstName, lastName, email, phone, metadata)

	if err == nil {
		atomic.AddInt64(&insertCount, 1)
		if *verbose {
			log.Printf("INSERT: %s %s <%s>", firstName, lastName, email)
		}
	}

	return err
}

// Update modifies a random existing customer
func (g *DataGenerator) Update(ctx context.Context) error {
	// Get a random customer ID
	var customerID string
	err := g.db.QueryRowContext(ctx, `
		SELECT id FROM customers
		WHERE metadata->>'generated' = 'true'
		ORDER BY RANDOM()
		LIMIT 1
	`).Scan(&customerID)

	if err != nil {
		// No generated customers to update, insert instead
		return g.Insert(ctx)
	}

	// Update the customer
	newPhone := fmt.Sprintf("+1-555-%04d", rand.Intn(10000))
	_, err = g.db.ExecContext(ctx, `
		UPDATE customers SET phone = $1, updated_at = NOW()
		WHERE id = $2
	`, newPhone, customerID)

	if err == nil {
		atomic.AddInt64(&updateCount, 1)
		if *verbose {
			log.Printf("UPDATE: customer %s", customerID)
		}
	}

	return err
}

// Delete removes a random generated customer
func (g *DataGenerator) Delete(ctx context.Context) error {
	// Get a random generated customer ID
	var customerID string
	err := g.db.QueryRowContext(ctx, `
		SELECT id FROM customers
		WHERE metadata->>'generated' = 'true'
		ORDER BY RANDOM()
		LIMIT 1
	`).Scan(&customerID)

	if err != nil {
		// No generated customers to delete, insert instead
		return g.Insert(ctx)
	}

	// Delete the customer
	_, err = g.db.ExecContext(ctx, `
		DELETE FROM customers WHERE id = $1
	`, customerID)

	if err == nil {
		atomic.AddInt64(&deleteCount, 1)
		if *verbose {
			log.Printf("DELETE: customer %s", customerID)
		}
	}

	return err
}

// Mixed performs a random operation (60% insert, 30% update, 10% delete)
func (g *DataGenerator) Mixed(ctx context.Context) error {
	r := rand.Float32()
	switch {
	case r < 0.6:
		return g.Insert(ctx)
	case r < 0.9:
		return g.Update(ctx)
	default:
		return g.Delete(ctx)
	}
}

func printStats(startTime time.Time) {
	elapsed := time.Since(startTime)
	inserts := atomic.LoadInt64(&insertCount)
	updates := atomic.LoadInt64(&updateCount)
	deletes := atomic.LoadInt64(&deleteCount)
	errors := atomic.LoadInt64(&errorCount)
	total := inserts + updates + deletes

	rate := float64(total) / elapsed.Seconds()

	log.Printf("Stats: %d total (%.1f/s) | I:%d U:%d D:%d | Errors:%d | Elapsed:%s",
		total, rate, inserts, updates, deletes, errors, elapsed.Round(time.Second))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
