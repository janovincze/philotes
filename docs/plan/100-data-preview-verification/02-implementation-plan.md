# Implementation Plan: Issue #100 - Data Preview in Verification Step

## Overview

The backend already returns `sample_rows` in `DataVerificationResponse`, and the `ResultsTable` component from the Query page already exists. This is a focused UI enhancement to the existing `step-data-verification.tsx` component.

## Single Task: Enhance Verification Step with Data Preview

**File to modify:** `web/src/components/onboarding/step-data-verification.tsx`

### Changes

1. **Import `ResultsTable`** from `@/components/query/results-table`
2. **Import `QueryColumn`** type from `@/lib/api/types`
3. **Derive columns from sample rows** — when `verifyMutation.data.sample_rows` exists, extract column names from the first row and create `QueryColumn[]` with inferred types
4. **Replace the basic "Verification Results" section** (lines 217-227) with:
   - Row count badge showing total rows replicated
   - Schema information via column headers with types (provided by ResultsTable)
   - Sample data rows displayed in the ResultsTable component
   - "Open in Query Editor" link button to `/query`
5. **Handle no-data gracefully** — when `sample_rows` is empty/undefined, show current simple result

### No Backend Changes Needed

The `VerifyDataFlow` service already:
- Returns `sample_rows []map[string]any` with up to 10 rows
- Returns `row_count int64` with total count
- The frontend `DataVerificationResponse` type already has `sample_rows?: Record<string, unknown>[]`

### No New Files Needed

Reusing existing `ResultsTable` component as-is.
