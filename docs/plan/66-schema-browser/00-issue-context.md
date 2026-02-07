# Issue #66 - Schema Browser with Table Explorer and Data Profiling

## Goal
Provide a tree-view navigator for discovering catalogs, schemas, tables, and columns with data profiling capabilities.

## Key Features
1. Tree navigation: catalogs → schemas → tables → columns
2. Search & filter across all objects
3. Table details panel with tabs: Overview, Columns, Preview, Profile, DDL
4. Quick actions: click-to-insert, generate SELECT, copy qualified name
5. Favorites/starred tables
6. Schema cache with refresh

## API Requirements
- GET /api/v1/query/schemas - list schemas
- GET /api/v1/query/schemas/:schema/tables - list tables in schema
- GET /api/v1/query/tables/:table/columns - get table columns
- GET /api/v1/query/tables/:table/preview - preview data (limit 100)
- GET /api/v1/query/tables/:table/profile - column profiling stats
- GET /api/v1/query/tables/:table/ddl - get DDL

## Acceptance Criteria
- Tree view with catalogs, schemas, tables, columns
- Search across all objects
- Table details panel with all tabs
- Data preview (first 100 rows)
- Column profiling with statistics
- Click-to-insert column names
- Generate SELECT statement
- Favorite/starred tables
- Schema cache with refresh

## Dependencies
- Trino query execution already exists (#97 closed)
- QueryService in backend already handles SQL execution
