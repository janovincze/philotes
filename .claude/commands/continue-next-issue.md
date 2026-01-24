# Continue to Next Issue

Automatically continue to the next issue in the project queue after completing the current one.

## Usage

After completing an issue and creating a PR, use this command to:

1. Check the next available issue
2. Start working on it immediately

## Instructions

```bash
# Checkout main and pull latest
git checkout main
git pull origin main

# Find next issue by priority
gh issue list --repo janovincze/philotes \
  --label "priority:critical" \
  --state open \
  --limit 1

gh issue list --repo janovincze/philotes \
  --label "priority:high" \
  --state open \
  --limit 1
```

## Workflow

1. Display the next issue details
2. Ask user for confirmation
3. If confirmed, run `/start-issue <number>`

## Example

```
Completed: #26 - Scaling Engine (PR #42 created)

Next available issue:
#27 - Dashboard Scaling Configuration UI
Priority: high
Milestone: M3: Production Ready
Dependencies: #26 (just completed ✓)

Start this issue? [Y/n]
```

## Auto-Selection Logic

1. **Priority Order:**
   - `priority:critical` first
   - `priority:high` second
   - `priority:medium` third

2. **Dependency Check:**
   - Skip issues that have uncompleted blockers
   - Prefer issues that were just unblocked

3. **Milestone Grouping:**
   - Prefer issues in the current milestone

## Quick Commands

```bash
# List all open issues by priority
gh issue list --repo janovincze/philotes --state open --label "priority:critical"
gh issue list --repo janovincze/philotes --state open --label "priority:high"
gh issue list --repo janovincze/philotes --state open --label "priority:medium"

# View specific issue
gh issue view <number> --repo janovincze/philotes

# Check project board
gh project item-list 7 --owner janovincze --limit 10
```
