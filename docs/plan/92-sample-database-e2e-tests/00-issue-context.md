# Issue #92: Create sample database, usage guide, and E2E test for CDC pipeline

## Summary

Create a complete end-to-end demonstration of Philotes CDC pipeline with:
1. Sample source database with realistic data
2. Step-by-step usage guide
3. Automated E2E tests validating the entire pipeline

## Context

New users need a working example to understand how Philotes works. This issue ensures the platform is usable end-to-end with proper documentation and testing.

## Tasks

### 1. Sample Source Database
- Choose appropriate sample dataset (e-commerce orders recommended for CRUD demonstration)
- Create schema with multiple related tables
- Include various data types (timestamps, numerics, text, JSON)
- Add seed data script for docker-compose
- Include data generator script for continuous CDC testing

### 2. Usage Guide (`docs/guides/getting-started.md`)
- Prerequisites and installation
- Starting development environment
- Creating source connection via API/Dashboard
- Creating pipeline and table mappings
- Starting CDC pipeline
- Verifying data in Iceberg
- Querying with Trino
- Monitoring with Grafana
- Troubleshooting

### 3. E2E Tests
- Set up test infrastructure
- Test source connection creation
- Test pipeline creation with table mappings
- Test pipeline start/stop
- Test data insertion and CDC capture
- Test data verification in Iceberg
- Test Trino queries
- Test update/delete CDC operations

### 4. Bug Fixes
- Document and fix any non-working components discovered during implementation

## Acceptance Criteria

- [ ] Sample database initializes automatically with `docker compose up`
- [ ] Usage guide is complete and followable by new users
- [ ] E2E tests pass in CI/CD
- [ ] Bugs fixed or documented as separate issues
- [ ] README updated to reference new guide

## Technical Notes

- Sample data: ~10k-100k rows for quick testing
- E2E tests: < 5 minutes completion time
- Consider testcontainers for isolated testing
- Data generator: configurable insert rates

## Dependencies

- Docker compose environment
- Philotes API
- CDC Worker
- Lakekeeper catalog
- Trino query engine
