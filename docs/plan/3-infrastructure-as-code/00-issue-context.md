# Issue Context - #3 Infrastructure as Code - Multi-Provider

## Issue Details

**Title:** FOUND-003: Infrastructure as Code - Multi-Provider
**State:** OPEN
**Labels:** epic:foundation, phase:mvp, priority:high, type:infra
**Milestone:** M0: Foundation

## Goal

Enable one-command deployment of Philotes to multiple European cloud providers, making self-hosted CDC accessible to teams without DevOps expertise.

## Problem Statement

Setting up Kubernetes clusters, configuring networking, and deploying applications requires significant infrastructure knowledge. Most data teams just want to replicate their data, not become cloud architects.

## Target Users

- Data engineers who want CDC without managing infrastructure
- Small teams without dedicated DevOps
- European companies needing EU data residency

## Usage

User runs `pulumi up` with their cloud provider credentials and gets a fully functional Philotes deployment in 15 minutes.

## Supported Providers

1. **Hetzner Cloud** (primary - best price/performance)
2. OVHcloud
3. Scaleway
4. Exoscale
5. Contabo

## Acceptance Criteria

- [ ] Pulumi project structure with provider abstraction
- [ ] K3s cluster deployment per provider
- [ ] Networking (VPC, firewall, load balancer)
- [ ] Storage provisioning (block storage for MinIO)
- [ ] DNS configuration (optional)
- [ ] SSL/TLS via cert-manager
- [ ] Cost estimation output
- [ ] Destroy/cleanup support

## Estimate

~15,000 LOC

## Dependencies

- FOUND-001 (completed)

## Blocks

- INSTALL-001
- INFRA-001

## Value Proposition

This is a key differentiator. While competitors require cloud expertise or lock you into their managed service, Philotes lets you own your infrastructure with minimal effort.
