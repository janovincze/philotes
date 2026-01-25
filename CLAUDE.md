# Philotes - Project Context for AI Assistants

This document provides comprehensive context about the Philotes project to help AI assistants understand the vision, architecture, and development approach.

## What is Philotes?

**Philotes** is an open-source Change Data Capture (CDC) platform that replicates data from PostgreSQL databases to Apache Iceberg data lakes in near-real-time. It's designed to be the "simple, affordable alternative" to enterprise CDC solutions like Fivetran, Airbyte, or Debezium.

### The Name
In Greek mythology, Philotes was the personification of affection and friendship - representing the connection between systems. The project connects operational databases to analytical data lakes.

## Problem Statement

### The Data Lake Challenge
Organizations want to analyze their operational data alongside historical data, but:

1. **Enterprise CDC solutions are expensive** - Fivetran/Airbyte charge per row or connector, costing $1,000+/month for modest workloads
2. **Self-hosted alternatives are complex** - Debezium + Kafka + Spark requires significant expertise and infrastructure
3. **Cloud lock-in is real** - AWS/GCP/Azure solutions tie you to expensive cloud ecosystems
4. **European data residency** - GDPR and data sovereignty requirements favor EU-based infrastructure

### Our Solution
Philotes provides:
- **Simple deployment** - One-click installer for European cloud providers
- **Low cost** - Run on €30-150/month infrastructure (Hetzner, OVHcloud, Scaleway)
- **Modern stack** - PostgreSQL → Iceberg with Parquet files, queryable by any engine
- **Self-hosted** - Your data stays in your infrastructure

## Target Users

### Primary: Small/Medium Business Data Teams
- 1-5 data engineers
- PostgreSQL as primary database
- Need analytics without enterprise budgets
- Value simplicity over infinite configurability

### Secondary: Cost-Conscious Enterprises
- Teams looking to reduce CDC costs
- European companies with data residency requirements
- Organizations wanting to avoid cloud vendor lock-in

### Persona: "Dana the Data Engineer"
Dana works at a 50-person SaaS company. They have PostgreSQL for their app and want to:
- Build a data warehouse for analytics
- Run historical reports without impacting production
- Eventually add ML/AI workloads

Dana doesn't have time to learn Kafka, manage Spark clusters, or justify $2,000/month for Fivetran. Philotes lets Dana set up CDC in an afternoon.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           User Interfaces                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │  Dashboard   │  │  REST API    │  │     CLI      │                   │
│  │  (Next.js)   │  │  (Gin/Go)    │  │   (Cobra)    │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Management Layer                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │   Sources    │  │  Pipelines   │  │   Scaling    │                   │
│  │  Management  │  │  Management  │  │   Policies   │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          CDC Pipeline                                    │
│                                                                          │
│  PostgreSQL ──► pgstream ──► Buffer DB ──► CDC Worker ──► Iceberg       │
│  (Source)       (WAL)        (Events)      (Batches)      (Parquet)     │
│                                                                          │
│  Features:                                                               │
│  • Logical replication via pgstream                                      │
│  • Checkpointing for exactly-once delivery                              │
│  • Dead-letter queue for failed events                                   │
│  • Backpressure control                                                  │
│  • Health monitoring                                                     │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Data Lake Layer                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │  Lakekeeper  │  │    MinIO     │  │   Iceberg    │                   │
│  │  (Catalog)   │  │  (Storage)   │  │   Tables     │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Query Layer                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │
│  │    Trino     │  │  RisingWave  │  │   DuckDB     │                   │
│  │  (SQL/BI)    │  │  (Streaming) │  │  (Local)     │                   │
│  └──────────────┘  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────────────────────┘
```

## Current Implementation Status

### Completed (Production-Ready)
- **CDC Pipeline** - Full PostgreSQL → Iceberg flow with pgstream
- **Buffer Database** - Event persistence with checkpointing
- **Iceberg Writer** - Parquet files with schema management
- **Management API** - CRUD for sources, pipelines, table mappings
- **Health System** - Liveness/readiness probes, metrics foundation

### In Progress (Open Issues)
- **Authentication** - API keys, JWT, SSO/OIDC
- **Dashboard** - Next.js web UI
- **Observability** - Prometheus metrics, alerting
- **Auto-scaling** - KEDA-based scaling with cost controls
- **Query Layer** - Trino/RisingWave/DuckDB integration
- **One-Click Installer** - Web-based cloud deployment

## Development Guidelines

### Code Organization
```
philotes/
├── cmd/                    # Application entry points
│   ├── philotes-api/       # Management API server
│   ├── philotes-worker/    # CDC Worker service
│   └── philotes-cli/       # CLI tool
├── internal/               # Private application code
│   ├── api/                # API handlers, services, repositories
│   ├── cdc/                # CDC pipeline components
│   ├── config/             # Configuration management
│   └── iceberg/            # Iceberg integration
├── deployments/            # Docker, Kubernetes, Pulumi
└── web/                    # Dashboard (Next.js)
```

### Patterns to Follow
1. **Layered Architecture** - Handler → Service → Repository for API
2. **Structured Logging** - Use `slog` with component context
3. **Configuration** - Environment variables via `internal/config`
4. **Error Handling** - Wrap errors with context, use typed errors
5. **Testing** - Table-driven tests, mock interfaces

### Technology Stack
- **Language**: Go 1.22+
- **API Framework**: Gin
- **CDC Library**: pgstream
- **Table Format**: Apache Iceberg v2
- **Catalog**: Lakekeeper (REST)
- **Storage**: MinIO (S3-compatible)
- **Dashboard**: Next.js 14+, TypeScript, Tailwind, shadcn/ui

## Issue Organization

### Epics
Issues are organized into epics representing major feature areas:

| Epic | Description | Priority |
|------|-------------|----------|
| `foundation` | Project setup, CI/CD, IaC | High |
| `api` | REST API endpoints | High |
| `dashboard` | Web UI | Medium |
| `observability` | Metrics, alerting | High |
| `infrastructure` | Helm, Vault, KEDA | High |
| `authentication` | API keys, SSO, RBAC | Medium |
| `query` | Trino, RisingWave, DuckDB | Medium |
| `scaling` | Auto-scaling engine | High |
| `installation` | One-click installer | High |
| `connectors` | Future source/dest types | Low |

### Phases
- **MVP** - Minimum viable product for early adopters
- **v1** - First stable release with core features
- **Future** - Post-v1 enhancements

### Milestones
- **M1: Core Pipeline** - CDC working end-to-end
- **M2: Management Layer** - API, dashboard, authentication
- **M3: Production Ready** - Scaling, observability, installer

## Value Proposition Summary

| Competitor | Monthly Cost | Complexity | Data Location |
|------------|--------------|------------|---------------|
| Fivetran | $1,000+ | Low | Their cloud |
| Airbyte Cloud | $500+ | Low | Their cloud |
| Self-hosted Debezium | $200+ | Very High | Your infra |
| **Philotes** | **$30-150** | **Low** | **Your infra** |

## Key Decisions

1. **PostgreSQL-first** - Start with PostgreSQL, add MySQL/others later via connector SDK
2. **Iceberg-native** - Modern table format with time travel, schema evolution
3. **European clouds** - Hetzner, OVHcloud, Scaleway for cost and GDPR
4. **No Kafka** - Direct CDC to buffer DB, simpler than Kafka Connect
5. **REST catalog** - Lakekeeper instead of Hive Metastore

## How to Contribute

1. Check open issues in [GitHub Projects](https://github.com/users/janovincze/projects/7)
2. Issues are labeled by epic, priority, and phase
3. Each issue should have Context (goal, problem, users) and Acceptance Criteria
4. Follow the patterns in existing code
5. Write tests for new functionality

## Useful Commands

```bash
# Development
make docker-up      # Start local environment
make build          # Build all binaries
make test           # Run tests
make lint           # Run linter

# Running
make run-api        # Start API server
make run-worker     # Start CDC worker

# Docker services
# PostgreSQL (buffer): localhost:5432
# PostgreSQL (source): localhost:5433
# MinIO: localhost:9000 (console: 9001)
# Lakekeeper: localhost:8181
# Prometheus: localhost:9090
# Grafana: localhost:3000
```
