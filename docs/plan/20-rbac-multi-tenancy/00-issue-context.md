# Issue #20: AUTH-002 - RBAC and Multi-Tenancy Foundation

## Issue Description

Implement role-based access control (RBAC) and multi-tenancy so organizations can control who can view/edit which pipelines and isolate different teams or customers.

## Problem Statement

In teams with multiple users, not everyone should have admin access. Some should only view, others can edit specific pipelines. Multi-tenant deployments need complete isolation between customers.

## Who Benefits

- Large organizations with multiple teams sharing Philotes
- MSPs/SaaS companies running Philotes for multiple customers
- Security teams enforcing least-privilege access

## RBAC Model

- **Roles**: admin, editor, viewer, custom
- **Permissions**: create, read, update, delete, start, stop
- **Resources**: sources, pipelines, users, settings
- **Scopes**: global, tenant, resource-specific

## Acceptance Criteria

- [ ] Role and permission data model
- [ ] Role assignment to users/groups
- [ ] Permission checking middleware
- [ ] Tenant isolation for resources
- [ ] Tenant-scoped API responses
- [ ] Audit logging for permission changes
- [ ] Default roles with sensible permissions

## Dependencies

- AUTH-001 (API keys, JWT, SSO/OIDC) - Complete

## Estimate

~10,000 LOC
