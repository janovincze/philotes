# Issue #64 - Core SQL Editor with Monaco, Autocomplete, and Linting

## Title
feat(query): Core SQL editor with Monaco, autocomplete, and linting

## Labels
- epic:query, phase:v1, priority:high, type:feature

## Summary
Build a professional-grade SQL editor using Monaco with intelligent features: autocomplete, linting, formatting, multi-tab support, keyboard shortcuts, result grid enhancements, and theme support.

## Key Feature Areas
1. **Core Editor** — Monaco with SQL highlighting, line numbers, code folding, minimap, find/replace
2. **Autocomplete** — Schema-aware suggestions (tables, columns, functions, keywords)
3. **SQL Linting** — Real-time syntax error detection, common mistake warnings
4. **Query Formatting** — One-click SQL beautification (sql-formatter)
5. **Multi-Tab Support** — Multiple query tabs with persistence
6. **Keyboard Shortcuts** — Cmd+Enter, Cmd+Shift+Enter (execute selected), Cmd+Shift+F (format), etc.
7. **Result Grid** — Sortable columns, resizing, filter, pagination, copy, null highlighting
8. **Theme Support** — Dark/light toggle with system preference detection

## Already Implemented (from previous PRs)
- Monaco editor rendering with SQL input
- Basic SQL syntax highlighting
- Schema-aware autocomplete (tables, columns from metadata hooks)
- Query execution with results table
- Ctrl+Enter to execute
- Schema browser sidebar (#66)
- Query history (client-side, in-memory)

## Acceptance Criteria
See issue for full list — key ones:
- [ ] Monaco editor renders and accepts input ✅ (done)
- [ ] SQL syntax highlighting for Trino/PostgreSQL/DuckDB
- [ ] Schema-aware autocomplete (tables, columns) ✅ (partially done)
- [ ] Real-time linting with error messages
- [ ] Query formatting works
- [ ] Multi-tab support with state persistence
- [ ] Execute query with results grid ✅ (basic done)
- [ ] Keyboard shortcuts functional
- [ ] Dark/light theme toggle
- [ ] Performance: <100ms keystroke latency

## Dependencies
- None (foundation issue)
