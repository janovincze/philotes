# Implementation Plan: KEDA Autoscaling Configuration

## Summary

Implement KEDA (Kubernetes Event-Driven Autoscaling) configuration for Philotes CDC workers with Prometheus-based scaling triggers. This enables intelligent auto-scaling based on CDC-specific metrics like replication lag and buffer depth.

## Approach Overview

1. Enhance the existing KEDA values configuration in philotes-worker chart
2. Update ScaledObject template to support multiple trigger types
3. Add Prometheus scaler as primary scaling trigger
4. Implement scale-to-zero capability with activation thresholds
5. Add documentation for configuration options

## Files to Modify

| File | Changes |
|------|---------|
| `charts/philotes-worker/values.yaml` | Expand KEDA config with Prometheus scalers |
| `charts/philotes-worker/templates/scaledobject.yaml` | Enhance template for multiple triggers |
| `charts/philotes-worker/templates/hpa.yaml` | Create fallback HPA for non-KEDA environments |
| `charts/philotes/values.yaml` | Add KEDA config to umbrella chart |

## Files to Create

| File | Purpose |
|------|---------|
| `charts/philotes-worker/templates/scaledjob.yaml` | Optional: ScaledJob for batch processing |
| `charts/philotes-worker/README.md` | Update with KEDA documentation |

## Task Breakdown

### Task 1: Update values.yaml KEDA Configuration

Expand the `keda` section in `charts/philotes-worker/values.yaml`:

```yaml
keda:
  enabled: false
  minReplicas: 1
  maxReplicas: 10
  pollingInterval: 30
  cooldownPeriod: 300
  # Scale to zero support
  idleReplicaCount: 0
  minReplicaCount: 0

  # Scaling behavior (KEDA v2)
  advanced:
    horizontalPodAutoscalerConfig:
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 300
          policies:
            - type: Percent
              value: 50
              periodSeconds: 60
        scaleUp:
          stabilizationWindowSeconds: 0
          policies:
            - type: Percent
              value: 100
              periodSeconds: 15
            - type: Pods
              value: 4
              periodSeconds: 15
          selectPolicy: Max

  # Prometheus scaler (primary)
  prometheus:
    enabled: true
    serverAddress: "http://prometheus:9090"
    # Lag-based scaling
    lagTrigger:
      enabled: true
      metricName: philotes_cdc_lag_seconds
      query: "max(philotes_cdc_lag_seconds)"
      threshold: "300"
      activationThreshold: "60"
    # Buffer depth scaling
    bufferTrigger:
      enabled: true
      metricName: philotes_buffer_depth
      query: "max(philotes_buffer_depth)"
      threshold: "8000"
      activationThreshold: "1000"
    # Events per second scaling
    throughputTrigger:
      enabled: false
      metricName: philotes_buffer_events_processed_total
      query: "sum(rate(philotes_buffer_events_processed_total[5m]))"
      threshold: "1000"

  # PostgreSQL scaler (alternative)
  postgresql:
    enabled: false
    host: ""
    port: "5432"
    database: ""
    user: ""
    password: ""
    existingSecret: ""
    passwordKey: "password"
    sslMode: "disable"
    query: "SELECT COALESCE(pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn), 0) FROM pg_replication_slots WHERE slot_name = 'philotes_cdc'"
    targetQueryValue: "1000000"
    activationTargetQueryValue: "100000"

  # CPU/Memory fallback scaler
  cpu:
    enabled: false
    metricType: "Utilization"
    value: "80"
  memory:
    enabled: false
    metricType: "Utilization"
    value: "80"
```

### Task 2: Enhance ScaledObject Template

Update `charts/philotes-worker/templates/scaledobject.yaml`:

```yaml
{{- if .Values.keda.enabled }}
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{ include "philotes-worker.fullname" . }}
  labels:
    {{- include "philotes-worker.labels" . | nindent 4 }}
spec:
  scaleTargetRef:
    name: {{ include "philotes-worker.fullname" . }}
  pollingInterval: {{ .Values.keda.pollingInterval }}
  cooldownPeriod: {{ .Values.keda.cooldownPeriod }}
  minReplicaCount: {{ .Values.keda.minReplicaCount | default .Values.keda.minReplicas }}
  maxReplicaCount: {{ .Values.keda.maxReplicas }}
  {{- if .Values.keda.idleReplicaCount }}
  idleReplicaCount: {{ .Values.keda.idleReplicaCount }}
  {{- end }}
  {{- if .Values.keda.advanced }}
  advanced:
    {{- toYaml .Values.keda.advanced | nindent 4 }}
  {{- end }}
  triggers:
    {{- /* Prometheus Lag Trigger */}}
    {{- if and .Values.keda.prometheus.enabled .Values.keda.prometheus.lagTrigger.enabled }}
    - type: prometheus
      metadata:
        serverAddress: {{ .Values.keda.prometheus.serverAddress | quote }}
        metricName: {{ .Values.keda.prometheus.lagTrigger.metricName | quote }}
        query: {{ .Values.keda.prometheus.lagTrigger.query | quote }}
        threshold: {{ .Values.keda.prometheus.lagTrigger.threshold | quote }}
        {{- if .Values.keda.prometheus.lagTrigger.activationThreshold }}
        activationThreshold: {{ .Values.keda.prometheus.lagTrigger.activationThreshold | quote }}
        {{- end }}
    {{- end }}
    {{- /* Prometheus Buffer Trigger */}}
    {{- if and .Values.keda.prometheus.enabled .Values.keda.prometheus.bufferTrigger.enabled }}
    - type: prometheus
      metadata:
        serverAddress: {{ .Values.keda.prometheus.serverAddress | quote }}
        metricName: {{ .Values.keda.prometheus.bufferTrigger.metricName | quote }}
        query: {{ .Values.keda.prometheus.bufferTrigger.query | quote }}
        threshold: {{ .Values.keda.prometheus.bufferTrigger.threshold | quote }}
        {{- if .Values.keda.prometheus.bufferTrigger.activationThreshold }}
        activationThreshold: {{ .Values.keda.prometheus.bufferTrigger.activationThreshold | quote }}
        {{- end }}
    {{- end }}
    {{- /* Prometheus Throughput Trigger */}}
    {{- if and .Values.keda.prometheus.enabled .Values.keda.prometheus.throughputTrigger.enabled }}
    - type: prometheus
      metadata:
        serverAddress: {{ .Values.keda.prometheus.serverAddress | quote }}
        metricName: {{ .Values.keda.prometheus.throughputTrigger.metricName | quote }}
        query: {{ .Values.keda.prometheus.throughputTrigger.query | quote }}
        threshold: {{ .Values.keda.prometheus.throughputTrigger.threshold | quote }}
    {{- end }}
    {{- /* PostgreSQL Trigger */}}
    {{- if .Values.keda.postgresql.enabled }}
    - type: postgresql
      metadata:
        host: {{ .Values.keda.postgresql.host | quote }}
        port: {{ .Values.keda.postgresql.port | quote }}
        dbName: {{ .Values.keda.postgresql.database | quote }}
        userName: {{ .Values.keda.postgresql.user | quote }}
        sslmode: {{ .Values.keda.postgresql.sslMode | quote }}
        query: {{ .Values.keda.postgresql.query | quote }}
        targetQueryValue: {{ .Values.keda.postgresql.targetQueryValue | quote }}
        {{- if .Values.keda.postgresql.activationTargetQueryValue }}
        activationTargetQueryValue: {{ .Values.keda.postgresql.activationTargetQueryValue | quote }}
        {{- end }}
      authenticationRef:
        name: {{ include "philotes-worker.fullname" . }}-keda-postgres
    {{- end }}
    {{- /* CPU Trigger */}}
    {{- if .Values.keda.cpu.enabled }}
    - type: cpu
      metricType: {{ .Values.keda.cpu.metricType }}
      metadata:
        value: {{ .Values.keda.cpu.value | quote }}
    {{- end }}
    {{- /* Memory Trigger */}}
    {{- if .Values.keda.memory.enabled }}
    - type: memory
      metricType: {{ .Values.keda.memory.metricType }}
      metadata:
        value: {{ .Values.keda.memory.value | quote }}
    {{- end }}
---
{{- /* TriggerAuthentication for PostgreSQL */}}
{{- if .Values.keda.postgresql.enabled }}
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: {{ include "philotes-worker.fullname" . }}-keda-postgres
  labels:
    {{- include "philotes-worker.labels" . | nindent 4 }}
spec:
  secretTargetRef:
    {{- if .Values.keda.postgresql.existingSecret }}
    - parameter: password
      name: {{ .Values.keda.postgresql.existingSecret }}
      key: {{ .Values.keda.postgresql.passwordKey }}
    {{- else }}
    - parameter: password
      name: {{ include "philotes-worker.fullname" . }}-keda-postgres
      key: password
    {{- end }}
---
{{- if not .Values.keda.postgresql.existingSecret }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "philotes-worker.fullname" . }}-keda-postgres
  labels:
    {{- include "philotes-worker.labels" . | nindent 4 }}
type: Opaque
data:
  password: {{ .Values.keda.postgresql.password | b64enc | quote }}
{{- end }}
{{- end }}
{{- end }}
```

### Task 3: Create HPA Template (Fallback)

Create `charts/philotes-worker/templates/hpa.yaml` for non-KEDA environments:

```yaml
{{- if and .Values.autoscaling.enabled (not .Values.keda.enabled) }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "philotes-worker.fullname" . }}
  labels:
    {{- include "philotes-worker.labels" . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "philotes-worker.fullname" . }}
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  metrics:
    {{- if .Values.autoscaling.targetCPUUtilizationPercentage }}
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
    {{- end }}
    {{- if .Values.autoscaling.targetMemoryUtilizationPercentage }}
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: {{ .Values.autoscaling.targetMemoryUtilizationPercentage }}
    {{- end }}
{{- end }}
```

### Task 4: Update Umbrella Chart

Add KEDA configuration to `charts/philotes/values.yaml` for the worker subchart.

### Task 5: Update Chart Documentation

Update `charts/philotes-worker/README.md` with KEDA configuration documentation.

## Configuration Examples

### Basic Prometheus Scaling

```yaml
keda:
  enabled: true
  minReplicas: 1
  maxReplicas: 5
  prometheus:
    enabled: true
    serverAddress: "http://prometheus:9090"
    lagTrigger:
      enabled: true
      threshold: "300"
```

### Scale to Zero

```yaml
keda:
  enabled: true
  minReplicaCount: 0
  idleReplicaCount: 0
  maxReplicas: 10
  prometheus:
    enabled: true
    lagTrigger:
      enabled: true
      activationThreshold: "60"
      threshold: "300"
```

### Multi-metric Scaling

```yaml
keda:
  enabled: true
  prometheus:
    enabled: true
    lagTrigger:
      enabled: true
      threshold: "300"
    bufferTrigger:
      enabled: true
      threshold: "8000"
  cpu:
    enabled: true
    value: "80"
```

## Verification

1. **Helm Template:** `helm template charts/philotes-worker --set keda.enabled=true`
2. **Lint:** `helm lint charts/philotes-worker`
3. **KEDA Installation:** Verify KEDA operator is installed in cluster
4. **Metrics Availability:** Verify Prometheus has the required metrics
5. **Scaling Test:** Deploy and verify scaling behavior under load

## Test Scenarios

1. **Lag Scaling Test:**
   - Simulate replication lag > 300s
   - Verify workers scale up
   - Reduce lag and verify scale down

2. **Buffer Depth Test:**
   - Fill buffer above threshold
   - Verify workers scale up
   - Drain buffer and verify scale down

3. **Scale to Zero Test:**
   - Stop all CDC activity
   - Verify workers scale to zero
   - Resume activity and verify wake-up

4. **Cooldown Test:**
   - Trigger rapid scale events
   - Verify cooldown prevents thrashing
