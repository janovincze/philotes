# Issue #100 - Data Preview in Verification Step

## Title
feat(dashboard): Data Preview in Verification Step

## Goal
Enhance the verification step to show actual sample data from replicated tables, providing the "see your data" moment within the wizard.

## Problem
Current verification step shows that data arrived but doesn't display the actual data. Users can't see their data without navigating to the Query page.

## Acceptance Criteria
- Display sample rows (first 5-10 rows) in a data table
- Show schema information (column names and types)
- Row count badge showing total rows replicated
- "Open in Query Editor" button linking to Query page
- Graceful handling when no data has arrived yet

## Technical Notes
- Modify `web/src/components/onboarding/step-data-verification.tsx`
- Use existing data verification API endpoint
- Reuse results table component from Query page

## Dependencies
- Requires #97 (Query Page) for results table component and "Open in Query Editor" link
- Note: #97 may not be implemented yet — need to check if results table component exists

## Labels
- epic:dashboard, phase:mvp, priority:medium
- Milestone: MVP: Dashboard Click-to-Query
