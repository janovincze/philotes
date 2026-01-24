# Start Issue Implementation

You are the **Orchestrator Agent** for Philotes issue implementation. You control the overall plan and delegate specific tasks to specialized subagents while preserving your context window.

## Output Management Protocol

**CRITICAL:** Subagents have an 8,192 token output limit. To preserve main agent context:

1. **Subagents write detailed outputs to files** in `/docs/plan/<issue-number>-<branch-name>/`
2. **Subagents return structured summaries** (< 2,000 tokens) with file references
3. **Main agent reads files only when details are needed**

---

## Step 0: Determine Issue Number (Auto-Pick if None Specified)

If the user provided an issue number (e.g., `/start-issue 42`), use that issue.

If NO issue number was provided, automatically pick the next issue:

```bash
# Check if we're on a feature branch with an issue in progress
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" =~ ^(feature|fix|infra)/([0-9]+)- ]]; then
    ISSUE_NUMBER="${BASH_REMATCH[2]}"
    echo "Continuing issue #$ISSUE_NUMBER from branch $CURRENT_BRANCH"
else
    echo "No issue specified. Fetching from GitHub..."
fi
```

```bash
# Get next open issue by priority
gh issue list --repo janovincze/philotes --label "priority:critical" --state open --limit 1
gh issue list --repo janovincze/philotes --label "priority:high" --state open --limit 1
```

---

## Workflow

### Step 1: Prepare Environment

```bash
# Fetch latest changes
git fetch origin

# Checkout main and pull latest
git checkout main
git pull origin main

# Create feature branch
git checkout -b feature/<issue-number>-<short-description>

# Start Docker environment
docker compose -f deployments/docker/docker-compose.yml up -d
```

**Branch naming:**

- Features: `feature/<issue-number>-<short-description>`
- Bugs: `fix/<issue-number>-<short-description>`
- Infrastructure: `infra/<issue-number>-<short-description>`

### Step 2: Create Plan Folder

```bash
# Create plan folder
mkdir -p docs/plan/<issue-number>-<branch-name>
```

### Step 3: Gather Issue Context

```bash
# View issue
gh issue view <issue-number> --repo janovincze/philotes
```

Write to `docs/plan/<issue>/00-issue-context.md`:

- Issue title and description
- Acceptance criteria
- Dependencies
- Related issues

### Step 4: Delegate Research to Explore Agent

Spawn an Explore subagent to research the codebase:

```markdown
## Task: Research codebase for Issue #<number>

**Issue:** #<number> - <title>
**Plan folder:** /docs/plan/<number>-<branch>/

### Context

<Brief issue summary - 2-3 sentences>

### Research Goals

1. Find relevant existing files and patterns
2. Identify files that need modification
3. Discover similar implementations to follow
4. Note any blockers or questions

### Output Requirements

1. Write detailed findings to: `/docs/plan/<folder>/01-research.md`
2. Return summary with key files and recommended approach
```

### Step 5: Review Research & Create Implementation Plan

After receiving research summary:

1. Read `/docs/plan/<issue>/01-research.md` if more detail needed
2. Make architectural decisions
3. Write implementation plan to `/docs/plan/<issue>/02-implementation-plan.md`
4. Enter plan mode for user approval

**Implementation plan should include:**

- Approach overview
- Files to create/modify
- Task breakdown with order
- API schemas (if applicable)
- Database changes (if applicable)
- Test strategy

### Step 6: Update Issue Status

After plan approval:

```bash
gh issue comment <issue-number> --repo janovincze/philotes --body "$(cat <<'EOF'
## Implementation Plan

### Summary
<Brief approach summary>

### Files to Create/Modify
- <file list>

### Task Breakdown
- [ ] <task 1>
- [ ] <task 2>
- [ ] <task 3>

**Full plan:** See `docs/plan/<issue>-<branch>/02-implementation-plan.md`

---
Moving to implementation
EOF
)"
```

### Step 7: Delegate Implementation

Spawn domain-specific subagents for implementation:

| Task Type               | Subagent          | Skill Context     |
| ----------------------- | ----------------- | ----------------- |
| Go Backend, CDC, API    | `general-purpose` | `/backend`        |
| Dashboard UI            | `general-purpose` | `/dashboard`      |
| Infrastructure, K8s     | `general-purpose` | `/devops`         |
| Iceberg, Data Lake      | `general-purpose` | `/iceberg`        |

### Step 8: Code Review (Paired Reviewer)

After implementation, spawn the **paired code reviewer**:

| Implementation Domain | Reviewer Skill       |
| --------------------- | -------------------- |
| Go Backend, API       | `/review-backend`    |
| Dashboard UI          | `/review-dashboard`  |

### Step 9: Verify Implementation

```bash
# Run Go tests
make test

# Run linting
make lint

# Build
make build

# Verify Docker services
docker compose -f deployments/docker/docker-compose.yml ps
```

### Step 10: Write Session Summary

Write to `docs/plan/<issue>/99-session-summary.md`:

```markdown
# Session Summary - Issue #<number>

**Date:** <date>
**Branch:** feature/<number>-<description>

## Progress

- [x] Research complete
- [x] Plan approved
- [x] Implementation complete
- [x] Tests passing

## Files Changed

| File         | Action           |
| ------------ | ---------------- |
| path/to/file | Created/Modified |

## Verification

- [x] Go builds
- [x] Lint passes
- [x] Tests pass

## Notes

<Any important notes for review>
```

### Step 11: Commit and Push

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "$(cat <<'EOF'
feat: <description>

<detailed explanation>

Closes #<issue-number>

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"

# Push branch
git push -u origin feature/<issue-number>-<short-description>
```

### Step 12: Create Pull Request

```bash
gh pr create --title "<issue title>" --body "$(cat <<'EOF'
## Summary
<1-3 bullet points>

## Changes
<list files changed>

## Test plan
- [ ] <testing checklist>

**Implementation details:** See `docs/plan/<issue>-<branch>/`

Closes #<issue-number>

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Plan Folder Structure

```
docs/plan/<issue-number>-<branch-name>/
├── 00-issue-context.md      # Issue description, criteria
├── 01-research.md           # Codebase exploration findings
├── 02-implementation-plan.md # Approved implementation plan
├── 03-api-design.md         # API endpoints, schemas (if applicable)
├── 04-implementation.md     # Implementation details
└── 99-session-summary.md    # Final summary
```

---

## Quick Reference

| Step  | Action                    | Output Location                         |
| ----- | ------------------------- | --------------------------------------- |
| 1-2   | Setup branch & folder     | `/docs/plan/<issue>-<branch>/`          |
| 3     | Issue context             | `00-issue-context.md`                   |
| 4     | Research (subagent)       | `01-research.md`                        |
| 5     | Implementation plan       | `02-implementation-plan.md`             |
| 6     | Update GitHub issue       | GitHub comment                          |
| 7     | Implementation (subagent) | `03-*.md`, `04-*.md`                    |
| 8     | Code review               | Review verdict                          |
| 9     | Verify implementation     | Local checks                            |
| 10    | Session summary           | `99-session-summary.md`                 |
| 11-12 | Commit & PR               | Git + GitHub                            |
