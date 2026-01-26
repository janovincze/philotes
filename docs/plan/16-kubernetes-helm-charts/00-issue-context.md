# Issue Context - INFRA-001: Kubernetes Helm Charts

## Issue Details

- **Issue Number:** #16
- **Title:** INFRA-001: Kubernetes Helm Charts
- **Type:** Infrastructure
- **Priority:** High
- **Epic:** Infrastructure
- **Phase:** MVP
- **Milestone:** M2: Management Layer
- **Estimate:** ~8,000 LOC

## Goal

Package Philotes as production-ready Helm charts for easy deployment to any Kubernetes cluster, following best practices for cloud-native applications.

## Problem Statement

Docker Compose is great for development, but production needs Kubernetes. Writing Kubernetes manifests from scratch is error-prone and time-consuming. Helm charts provide a standard, configurable deployment method.

## Who Benefits

- Platform teams deploying Philotes to existing clusters
- Organizations with Kubernetes expertise
- Users who need production-grade deployment options

## How It's Used

`helm install philotes ./charts/philotes` deploys the full stack. Values files customize for different environments (dev/staging/prod). Helm upgrade handles rolling updates.

## Acceptance Criteria

- [ ] Helm chart per service (worker, api, dashboard)
- [ ] Umbrella chart for full deployment
- [ ] ConfigMaps and Secrets management
- [ ] Service and Ingress resources
- [ ] PodDisruptionBudgets
- [ ] Resource limits and requests
- [ ] Horizontal Pod Autoscaler definitions
- [ ] KEDA ScaledObject templates
- [ ] NetworkPolicies
- [ ] Values files for different environments

## Dependencies

- FOUND-003 (Foundation work - completed)

## Blocks

- INFRA-002 (HashiCorp Vault Integration)
- SCALE-001 (Scaling Engine)

## Philotes Components to Chart

1. **philotes-api** - Management REST API server
2. **philotes-worker** - CDC pipeline worker
3. **philotes-dashboard** - Next.js web UI (future)

## External Dependencies to Consider

- PostgreSQL (buffer database)
- MinIO (S3-compatible storage)
- Lakekeeper (Iceberg catalog)
- Prometheus (metrics)
- Grafana (dashboards)
