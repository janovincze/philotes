# Issue #12: DASH-002 - Setup Wizard for Non-Technical Users

## Overview

**Goal:** Guide new users through setting up their first CDC pipeline with a step-by-step wizard, achieving "time to value" under 10 minutes.

**Problem:** First-time users face a learning curve - concepts like sources, replication slots, and table mappings need explanation. The wizard abstracts these into simple questions.

**Who Benefits:**
- First-time users getting started
- Non-technical users needing guidance
- Teams evaluating Philotes

## Wizard Steps

1. **Welcome & overview** - Introduction to CDC and what they'll accomplish
2. **Connect source database** - Database credentials with connection test
3. **Select tables to replicate** - Table browser with search and select
4. **Configure destination settings** - Smart defaults with optional customization
5. **Review and create pipeline** - Summary before creation
6. **Watch first sync complete** - Success celebration with next steps

## Acceptance Criteria

- [ ] Multi-step form with progress indicator
- [ ] Database connection wizard with test button
- [ ] Table browser with search and select
- [ ] Configuration recommendations (smart defaults)
- [ ] Real-time validation at each step
- [ ] Success celebration with next steps

## Dependencies

- DASH-001 (Dashboard Framework) - Completed

## Blocks

- INSTALL-004 (Post-Installation Setup Wizard) - Shares wizard components

## Labels

- `epic:dashboard`
- `phase:mvp`
- `priority:medium`
- `type:feature`

## Milestone

M2: Management Layer
