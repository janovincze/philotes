-- Sample Trino Queries for Philotes Iceberg Data Lake
-- =====================================================

-- Connect to Trino:
-- docker exec -it philotes-trino trino --catalog iceberg --schema philotes

-- =====================================================
-- CATALOG AND SCHEMA EXPLORATION
-- =====================================================

-- List all catalogs
SHOW CATALOGS;

-- List schemas in Iceberg catalog
SHOW SCHEMAS FROM iceberg;

-- List tables in the philotes schema
SHOW TABLES FROM iceberg.philotes;

-- Describe a table
DESCRIBE iceberg.philotes.my_table;

-- Show table properties
SHOW CREATE TABLE iceberg.philotes.my_table;


-- =====================================================
-- BASIC QUERIES
-- =====================================================

-- Select all data
SELECT * FROM iceberg.philotes.my_table LIMIT 100;

-- Count records
SELECT COUNT(*) FROM iceberg.philotes.my_table;

-- Filter by column
SELECT * FROM iceberg.philotes.my_table
WHERE created_at > CURRENT_DATE - INTERVAL '7' DAY;


-- =====================================================
-- ICEBERG TIME TRAVEL
-- =====================================================

-- Query a specific snapshot
SELECT * FROM iceberg.philotes.my_table
FOR VERSION AS OF 12345678901234567890;

-- Query data at a specific timestamp
SELECT * FROM iceberg.philotes.my_table
FOR TIMESTAMP AS OF TIMESTAMP '2024-01-15 10:00:00 UTC';

-- List table snapshots
SELECT * FROM iceberg.philotes."my_table$snapshots";

-- List table history
SELECT * FROM iceberg.philotes."my_table$history";

-- List table partitions
SELECT * FROM iceberg.philotes."my_table$partitions";


-- =====================================================
-- AGGREGATIONS AND ANALYTICS
-- =====================================================

-- Daily aggregation
SELECT
    DATE_TRUNC('day', created_at) AS date,
    COUNT(*) AS record_count
FROM iceberg.philotes.my_table
GROUP BY 1
ORDER BY 1 DESC;

-- Top N by column
SELECT *
FROM iceberg.philotes.my_table
ORDER BY value DESC
LIMIT 10;

-- Running totals
SELECT
    created_at,
    value,
    SUM(value) OVER (ORDER BY created_at) AS running_total
FROM iceberg.philotes.my_table;


-- =====================================================
-- CDC-SPECIFIC QUERIES
-- =====================================================

-- Find latest version of each record (deduplication)
SELECT *
FROM (
    SELECT
        *,
        ROW_NUMBER() OVER (PARTITION BY id ORDER BY _cdc_updated_at DESC) AS rn
    FROM iceberg.philotes.my_table
)
WHERE rn = 1;

-- Track changes over time
SELECT
    id,
    _cdc_operation,
    _cdc_updated_at,
    value
FROM iceberg.philotes.my_table
WHERE id = 'some-id'
ORDER BY _cdc_updated_at;

-- Count operations by type
SELECT
    _cdc_operation,
    COUNT(*) AS count
FROM iceberg.philotes.my_table
GROUP BY 1;


-- =====================================================
-- JOINING TABLES
-- =====================================================

-- Join two replicated tables
SELECT
    o.id AS order_id,
    o.total,
    c.name AS customer_name
FROM iceberg.philotes.orders o
JOIN iceberg.philotes.customers c ON o.customer_id = c.id;


-- =====================================================
-- SCHEMA EVOLUTION QUERIES
-- =====================================================

-- View schema evolution history
SELECT * FROM iceberg.philotes."my_table$metadata";

-- Current table schema
SELECT * FROM iceberg.philotes."my_table$properties";


-- =====================================================
-- PERFORMANCE OPTIMIZATION
-- =====================================================

-- Explain query plan
EXPLAIN SELECT * FROM iceberg.philotes.my_table WHERE id = 'test';

-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM iceberg.philotes.my_table LIMIT 1000;

-- Partition pruning example (if table is partitioned)
SELECT * FROM iceberg.philotes.my_table
WHERE partition_column = '2024-01';


-- =====================================================
-- DATA QUALITY CHECKS
-- =====================================================

-- Find duplicate records
SELECT id, COUNT(*) AS count
FROM iceberg.philotes.my_table
GROUP BY id
HAVING COUNT(*) > 1;

-- Find null values in important columns
SELECT
    COUNT(*) AS total_rows,
    COUNT(id) AS non_null_id,
    COUNT(value) AS non_null_value
FROM iceberg.philotes.my_table;

-- Data freshness check
SELECT
    MAX(_cdc_updated_at) AS last_update,
    CURRENT_TIMESTAMP - MAX(_cdc_updated_at) AS lag
FROM iceberg.philotes.my_table;


-- =====================================================
-- MONITORING QUERIES (for Philotes pipeline health)
-- =====================================================

-- Records per pipeline (if tracked)
SELECT
    _philotes_pipeline_id,
    COUNT(*) AS record_count,
    MIN(_cdc_updated_at) AS earliest,
    MAX(_cdc_updated_at) AS latest
FROM iceberg.philotes.my_table
GROUP BY 1;
