# Issue #30: SCALE-005 - Query Engine Auto-scaling

## Issue Description

Enable auto-scaling for query engines (Trino, RisingWave) based on query load, ensuring fast query performance during peak hours without wasting resources during quiet periods.

## Problem Statement

Query engines are expensive (memory-intensive). Fixed-size clusters either can't handle peak load or waste money during off-hours. Dynamic scaling matches capacity to actual query demand.

## Who Benefits

- BI teams with predictable usage patterns (business hours)
- Organizations running expensive analytical queries
- Multi-tenant deployments with variable per-tenant load

## Acceptance Criteria

- [ ] Trino worker auto-scaling via KEDA
- [ ] RisingWave compute scaling
- [ ] Query metrics collection (queue, latency, concurrency)
- [ ] Cost-aware scaling decisions
- [ ] Scale-to-zero with fast warmup
- [ ] Query admission control during warmup
- [ ] Per-query resource estimation (optional)

## Scaling Triggers

- Query queue depth > threshold
- Query latency p95 > threshold
- Concurrent queries > threshold
- Scheduled scale-up (before business hours)

## Dependencies

- SCALE-001 (Scaling Engine) - Complete
- QUERY-001 (Trino Integration) - Complete

## Estimate

~6,000 LOC (likely reduced due to existing infrastructure)
