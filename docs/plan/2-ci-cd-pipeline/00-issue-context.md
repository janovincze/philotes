# Issue Context: FOUND-002 - CI/CD Pipeline Setup

## Issue Details
- **Number:** #2
- **Title:** FOUND-002: CI/CD Pipeline Setup
- **Type:** Infrastructure
- **Priority:** High
- **Phase:** MVP
- **Milestone:** M0: Foundation
- **Epic:** Foundation

## Goal
Establish automated testing and deployment pipelines to ensure code quality and enable continuous delivery.

## Problem Statement
Without CI/CD, every code change requires manual testing and deployment, slowing development and increasing the risk of bugs reaching production. This is especially critical for a data pipeline product where reliability is paramount.

## Who Benefits
- Developers get immediate feedback on code changes
- Users get faster bug fixes and feature releases
- Maintainers can confidently accept contributions

## How It's Used
- Every pull request triggers automated tests, linting, and builds
- Merges to main automatically deploy to staging
- Tagged releases deploy to production

## Acceptance Criteria
- [ ] GitHub Actions workflow for PR checks (lint, test, build)
- [ ] Automated release workflow with semantic versioning
- [ ] Docker image builds and push to GitHub Container Registry
- [ ] Integration test suite with docker-compose
- [ ] Code coverage reporting
- [ ] Security scanning (Snyk or similar)

## Dependencies
- FOUND-001 (completed) - Project foundation

## Blocks
- All other issues (indirectly) - CI/CD is foundational infrastructure

## Technical Notes
- Using GitHub Actions as the CI/CD platform
- GitHub Container Registry (ghcr.io) for Docker images
- Semantic versioning for releases
- Integration tests run with docker-compose
