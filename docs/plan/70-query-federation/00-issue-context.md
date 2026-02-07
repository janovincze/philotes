# Issue #70: Multi-source Query Federation and Cross-source JOINs

## Issue Summary
Enable querying across multiple data sources and joining data from different databases/systems in a single query. Users have data spread across Iceberg data lakes, PostgreSQL/MySQL databases, local files, etc.

## Key Features (from issue)
1. Data Source Management — CRUD for external data sources
2. Source Switcher — quick source switching in editor
3. Cross-Source Queries — federated JOINs via Trino
4. Local File Import — CSV/Parquet as queryable tables
5. Connection Pooling, Credentials Management, SQL Dialect Handling

## Acceptance Criteria
- [ ] Add/edit/delete data sources
- [ ] Test connection functionality
- [ ] Source switcher in editor
- [ ] Cross-source JOIN queries work
- [ ] CSV/Parquet file import
- [ ] Credentials stored in Vault
- [ ] Connection pooling
- [ ] SQL dialect handling
- [ ] Predicate pushdown
- [ ] Performance warnings for large joins

## Dependencies
- #64 (SQL Editor enhancements) — MERGED
- #66 (Schema Browser) — MERGED
- Trino deployment for federation engine

## Assessment
This is a VERY large issue covering both backend and frontend. Needs scoping to a practical first PR that delivers core value (data source management + source switching).
