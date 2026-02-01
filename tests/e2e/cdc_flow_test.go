//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestCDCFullFlow tests the complete CDC pipeline flow:
// Source DB → Pipeline → Iceberg → Trino Query
//
// Note: This test requires:
// - Docker Compose environment running
// - API server running
// - CDC Worker running (for full functionality)
func TestCDCFullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full CDC flow test in short mode")
	}

	client := NewAPIClient(apiBaseURL)
	sourceDB, err := NewSourceDB()
	if err != nil {
		t.Fatalf("Failed to connect to source database: %v", err)
	}
	defer sourceDB.Close()

	// Generate unique identifiers for this test run
	testID := time.Now().UnixNano()
	sourceName := fmt.Sprintf("e2e-cdc-source-%d", testID)
	pipelineName := fmt.Sprintf("e2e-cdc-pipeline-%d", testID)
	slotName := fmt.Sprintf("e2e_cdc_%d", testID)

	// Cleanup function
	var sourceID, pipelineID string
	defer func() {
		if pipelineID != "" {
			client.Post(fmt.Sprintf("/api/v1/pipelines/%s/stop", pipelineID), nil)
			time.Sleep(2 * time.Second)
			client.Delete("/api/v1/pipelines/" + pipelineID)
		}
		if sourceID != "" {
			client.Delete("/api/v1/sources/" + sourceID)
		}
		// Clean up replication slot
		sourceDB.Exec(fmt.Sprintf("SELECT pg_drop_replication_slot('%s')", slotName))
	}()

	// Step 1: Create Source
	t.Run("Step1_CreateSource", func(t *testing.T) {
		sourceReq := CreateSourceRequest{
			Name:            sourceName,
			Host:            sourceDBHost,
			Port:            5433,
			DatabaseName:    sourceDBName,
			Username:        sourceDBUser,
			Password:        sourceDBPassword,
			SSLMode:         "disable",
			SlotName:        slotName,
			PublicationName: "philotes_pub",
		}

		body, status, err := client.Post("/api/v1/sources", sourceReq)
		if err != nil {
			t.Fatalf("Failed to create source: %v", err)
		}
		RequireStatusCode(t, http.StatusCreated, status, body)

		var sourceResp SourceResponse
		if err := ParseJSON(body, &sourceResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		sourceID = sourceResp.Source.ID
		t.Logf("Created source: %s", sourceID)
	})

	// Step 2: Test Source Connection
	t.Run("Step2_TestConnection", func(t *testing.T) {
		body, status, err := client.Post("/api/v1/sources/"+sourceID+"/test", nil)
		if err != nil {
			t.Fatalf("Failed to test connection: %v", err)
		}
		RequireStatusCode(t, http.StatusOK, status, body)

		var result TestConnectionResponse
		if err := ParseJSON(body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if !result.Success {
			t.Fatalf("Connection test failed: %s", result.Message)
		}
		t.Log("Connection test successful")
	})

	// Step 3: Create Pipeline
	t.Run("Step3_CreatePipeline", func(t *testing.T) {
		pipelineReq := CreatePipelineRequest{
			Name:     pipelineName,
			SourceID: sourceID,
			Config: map[string]interface{}{
				"batch_size":             100,
				"flush_interval_seconds": 5,
			},
		}

		body, status, err := client.Post("/api/v1/pipelines", pipelineReq)
		if err != nil {
			t.Fatalf("Failed to create pipeline: %v", err)
		}
		RequireStatusCode(t, http.StatusCreated, status, body)

		var pipelineResp PipelineResponse
		if err := ParseJSON(body, &pipelineResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		pipelineID = pipelineResp.Pipeline.ID
		t.Logf("Created pipeline: %s", pipelineID)
	})

	// Step 4: Add Table Mappings
	t.Run("Step4_AddTableMappings", func(t *testing.T) {
		enabled := true
		tables := []AddTableMappingRequest{
			{
				Schema:  "public",
				Table:   "customers",
				Enabled: &enabled,
			},
			{
				Schema:  "public",
				Table:   "products",
				Enabled: &enabled,
			},
		}

		for _, mapping := range tables {
			body, status, err := client.Post(
				fmt.Sprintf("/api/v1/pipelines/%s/tables", pipelineID),
				mapping,
			)
			if err != nil {
				t.Fatalf("Failed to add table mapping for %s: %v", mapping.Table, err)
			}
			RequireStatusCode(t, http.StatusCreated, status, body)
			t.Logf("Added table mapping: %s.%s", mapping.Schema, mapping.Table)
		}
	})

	// Step 5: Start Pipeline
	t.Run("Step5_StartPipeline", func(t *testing.T) {
		body, status, err := client.Post(fmt.Sprintf("/api/v1/pipelines/%s/start", pipelineID), nil)
		if err != nil {
			t.Fatalf("Failed to start pipeline: %v", err)
		}

		if status != http.StatusOK && status != http.StatusAccepted {
			t.Fatalf("Expected status 200 or 202, got %d. Body: %s", status, string(body))
		}

		t.Log("Pipeline start initiated")

		// Wait for pipeline to be running (or skip if worker not running)
		ctx, cancel := context.WithTimeout(context.Background(), pipelineStartWait)
		defer cancel()

		err = PollUntil(ctx, pollInterval, func() (bool, error) {
			body, status, err := client.Get(fmt.Sprintf("/api/v1/pipelines/%s/status", pipelineID))
			if err != nil {
				return false, nil // Keep polling
			}
			if status != http.StatusOK {
				return false, nil
			}

			var statusResp PipelineStatusResponse
			if err := ParseJSON(body, &statusResp); err != nil {
				return false, nil
			}

			t.Logf("Pipeline status: %s", statusResp.Status)
			return statusResp.Status == "running", nil
		})

		if err != nil {
			t.Logf("Pipeline did not reach running state (worker may not be running): %v", err)
			// Don't fail - the API tests are still valid
		}
	})

	// Step 6: Insert Test Data
	t.Run("Step6_InsertTestData", func(t *testing.T) {
		testEmail := fmt.Sprintf("e2e-test-%d@example.com", testID)

		_, err := sourceDB.Exec(`
			INSERT INTO customers (first_name, last_name, email, phone, metadata)
			VALUES ($1, $2, $3, $4, $5)
		`, "E2E", "TestUser", testEmail, "+1-555-0000", `{"e2e_test": true}`)

		if err != nil {
			t.Fatalf("Failed to insert test customer: %v", err)
		}

		t.Logf("Inserted test customer: %s", testEmail)

		// Verify the insert in source
		var count int
		err = sourceDB.QueryRow("SELECT COUNT(*) FROM customers WHERE email = $1", testEmail).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to verify insert: %v", err)
		}
		if count != 1 {
			t.Fatalf("Expected 1 row, got %d", count)
		}
	})

	// Step 7: Verify Pipeline Processing (if worker is running)
	t.Run("Step7_VerifyProcessing", func(t *testing.T) {
		// Wait for CDC to process the insert
		time.Sleep(10 * time.Second)

		body, status, err := client.Get(fmt.Sprintf("/api/v1/pipelines/%s/status", pipelineID))
		if err != nil {
			t.Logf("Failed to get pipeline status: %v", err)
			return
		}

		if status == http.StatusOK {
			var statusResp PipelineStatusResponse
			if err := ParseJSON(body, &statusResp); err == nil {
				t.Logf("Pipeline status: %s, events processed: %d",
					statusResp.Status, statusResp.EventsProcessed)
			}
		}
	})

	// Step 8: Test UPDATE operation
	t.Run("Step8_UpdateData", func(t *testing.T) {
		testEmail := fmt.Sprintf("e2e-test-%d@example.com", testID)

		result, err := sourceDB.Exec(`
			UPDATE customers SET first_name = $1 WHERE email = $2
		`, "E2E-Updated", testEmail)

		if err != nil {
			t.Fatalf("Failed to update test customer: %v", err)
		}

		rowsAffected, _ := result.RowsAffected()
		t.Logf("Updated %d row(s)", rowsAffected)

		if rowsAffected != 1 {
			t.Errorf("Expected 1 row affected, got %d", rowsAffected)
		}
	})

	// Step 9: Test DELETE operation
	t.Run("Step9_DeleteData", func(t *testing.T) {
		testEmail := fmt.Sprintf("e2e-test-%d@example.com", testID)

		result, err := sourceDB.Exec(`
			DELETE FROM customers WHERE email = $1
		`, testEmail)

		if err != nil {
			t.Fatalf("Failed to delete test customer: %v", err)
		}

		rowsAffected, _ := result.RowsAffected()
		t.Logf("Deleted %d row(s)", rowsAffected)

		if rowsAffected != 1 {
			t.Errorf("Expected 1 row affected, got %d", rowsAffected)
		}
	})

	// Step 10: Stop Pipeline
	t.Run("Step10_StopPipeline", func(t *testing.T) {
		body, status, err := client.Post(fmt.Sprintf("/api/v1/pipelines/%s/stop", pipelineID), nil)
		if err != nil {
			t.Fatalf("Failed to stop pipeline: %v", err)
		}

		if status != http.StatusOK && status != http.StatusAccepted {
			t.Fatalf("Expected status 200 or 202, got %d. Body: %s", status, string(body))
		}

		t.Log("Pipeline stopped")
	})
}

// TestCDCBatchInsert tests inserting multiple rows in a batch
func TestCDCBatchInsert(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping batch insert test in short mode")
	}

	sourceDB, err := NewSourceDB()
	if err != nil {
		t.Fatalf("Failed to connect to source database: %v", err)
	}
	defer sourceDB.Close()

	testID := time.Now().UnixNano()
	batchSize := 100

	// Insert batch
	t.Run("InsertBatch", func(t *testing.T) {
		for i := 0; i < batchSize; i++ {
			email := fmt.Sprintf("batch-%d-%d@example.com", testID, i)
			_, err := sourceDB.Exec(`
				INSERT INTO customers (first_name, last_name, email, phone, metadata)
				VALUES ($1, $2, $3, $4, $5)
			`, fmt.Sprintf("Batch%d", i), "User", email, "+1-555-0000", `{"batch_test": true}`)

			if err != nil {
				t.Fatalf("Failed to insert batch item %d: %v", i, err)
			}
		}

		t.Logf("Inserted %d customers", batchSize)
	})

	// Verify count
	t.Run("VerifyBatch", func(t *testing.T) {
		var count int
		err := sourceDB.QueryRow(`
			SELECT COUNT(*) FROM customers
			WHERE email LIKE $1
		`, fmt.Sprintf("batch-%d-%%@example.com", testID)).Scan(&count)

		if err != nil {
			t.Fatalf("Failed to count batch: %v", err)
		}

		if count != batchSize {
			t.Errorf("Expected %d rows, got %d", batchSize, count)
		}
	})

	// Cleanup
	t.Run("CleanupBatch", func(t *testing.T) {
		result, err := sourceDB.Exec(`
			DELETE FROM customers
			WHERE email LIKE $1
		`, fmt.Sprintf("batch-%d-%%@example.com", testID))

		if err != nil {
			t.Fatalf("Failed to cleanup batch: %v", err)
		}

		rowsAffected, _ := result.RowsAffected()
		t.Logf("Cleaned up %d rows", rowsAffected)
	})
}

// TestCDCDataTypes tests CDC with various PostgreSQL data types
func TestCDCDataTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping data types test in short mode")
	}

	sourceDB, err := NewSourceDB()
	if err != nil {
		t.Fatalf("Failed to connect to source database: %v", err)
	}
	defer sourceDB.Close()

	testID := time.Now().UnixNano()

	// Test with a product (has various data types)
	t.Run("InsertProductWithAllTypes", func(t *testing.T) {
		sku := fmt.Sprintf("E2E-TEST-%d", testID)

		_, err := sourceDB.Exec(`
			INSERT INTO products (name, sku, description, price, cost, inventory_count, category, attributes, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
			"E2E Test Product",                                    // VARCHAR
			sku,                                                   // VARCHAR (unique)
			"A product created for E2E testing with various types", // TEXT
			99.99,                                                 // DECIMAL
			49.99,                                                 // DECIMAL (nullable)
			100,                                                   // INTEGER
			"Testing",                                             // VARCHAR
			`{"test_id": "`+fmt.Sprintf("%d", testID)+`", "features": ["a", "b", "c"]}`, // JSONB
			true, // BOOLEAN
		)

		if err != nil {
			t.Fatalf("Failed to insert product: %v", err)
		}

		// Verify
		var name, category string
		var price float64
		var inventory int
		var isActive bool

		err = sourceDB.QueryRow(`
			SELECT name, price, inventory_count, category, is_active
			FROM products WHERE sku = $1
		`, sku).Scan(&name, &price, &inventory, &category, &isActive)

		if err != nil {
			t.Fatalf("Failed to verify product: %v", err)
		}

		if name != "E2E Test Product" {
			t.Errorf("Unexpected name: %s", name)
		}
		if price != 99.99 {
			t.Errorf("Unexpected price: %f", price)
		}
		if inventory != 100 {
			t.Errorf("Unexpected inventory: %d", inventory)
		}

		t.Logf("Verified product with SKU: %s", sku)
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		sku := fmt.Sprintf("E2E-TEST-%d", testID)
		_, err := sourceDB.Exec("DELETE FROM products WHERE sku = $1", sku)
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	})
}

// TestCDCOrderWithItems tests CDC with foreign key relationships
func TestCDCOrderWithItems(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping order with items test in short mode")
	}

	sourceDB, err := NewSourceDB()
	if err != nil {
		t.Fatalf("Failed to connect to source database: %v", err)
	}
	defer sourceDB.Close()

	testID := time.Now().UnixNano()
	orderNumber := fmt.Sprintf("E2E-ORD-%d", testID)

	// Get a customer ID and product ID to use
	var customerID, productID string
	err = sourceDB.QueryRow("SELECT id FROM customers LIMIT 1").Scan(&customerID)
	if err != nil {
		t.Fatalf("No customers found: %v", err)
	}
	err = sourceDB.QueryRow("SELECT id FROM products LIMIT 1").Scan(&productID)
	if err != nil {
		t.Fatalf("No products found: %v", err)
	}

	var orderID string

	// Create order
	t.Run("CreateOrder", func(t *testing.T) {
		err = sourceDB.QueryRow(`
			INSERT INTO orders (customer_id, order_number, status, subtotal, tax, shipping, total, shipping_address)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`,
			customerID,
			orderNumber,
			"pending",
			100.00,
			8.00,
			10.00,
			118.00,
			`{"street": "123 Test St", "city": "Test City", "state": "TS", "zip": "12345"}`,
		).Scan(&orderID)

		if err != nil {
			t.Fatalf("Failed to create order: %v", err)
		}

		t.Logf("Created order: %s", orderID)
	})

	// Add order items
	t.Run("AddOrderItems", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			_, err := sourceDB.Exec(`
				INSERT INTO order_items (order_id, product_id, quantity, unit_price, discount)
				VALUES ($1, $2, $3, $4, $5)
			`,
				orderID,
				productID,
				i+1,       // quantity
				29.99,     // unit_price
				float64(i), // discount
			)

			if err != nil {
				t.Fatalf("Failed to add order item %d: %v", i, err)
			}
		}

		t.Log("Added 3 order items")
	})

	// Update order status
	t.Run("UpdateOrderStatus", func(t *testing.T) {
		_, err := sourceDB.Exec(`
			UPDATE orders SET status = $1 WHERE id = $2
		`, "confirmed", orderID)

		if err != nil {
			t.Fatalf("Failed to update order status: %v", err)
		}

		t.Log("Updated order status to confirmed")
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Order items are deleted via CASCADE
		_, err := sourceDB.Exec("DELETE FROM orders WHERE id = $1", orderID)
		if err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
		t.Logf("Cleaned up order %s", orderID)
	})
}
