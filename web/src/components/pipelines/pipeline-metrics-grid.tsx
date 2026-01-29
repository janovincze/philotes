"use client"

import { Activity, Clock, Layers, AlertTriangle } from "lucide-react"
import { MetricCard } from "@/components/metrics/metric-card"
import {
  formatEventsPerSecond,
  formatLatency,
  formatNumber,
} from "@/lib/utils/format-metrics"
import type { PipelineMetrics } from "@/lib/api/types"

interface PipelineMetricsGridProps {
  metrics: PipelineMetrics | null | undefined
  isLoading?: boolean
  className?: string
}

export function PipelineMetricsGrid({
  metrics,
  isLoading,
  className,
}: PipelineMetricsGridProps) {
  const lagStatus = getLagStatus(metrics?.lag_seconds ?? 0)
  const errorStatus = getErrorStatus(metrics?.error_count ?? 0)

  return (
    <div className={className}>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Events/sec"
          value={metrics?.events_per_second ?? 0}
          icon={<Activity className="h-4 w-4" />}
          formatValue={formatEventsPerSecond}
          isLoading={isLoading}
        />
        <MetricCard
          title="Replication Lag"
          value={metrics?.lag_seconds ?? 0}
          icon={<Clock className="h-4 w-4" />}
          formatValue={(v) => formatLatency(v)}
          status={lagStatus}
          isLoading={isLoading}
        />
        <MetricCard
          title="Buffer Depth"
          value={metrics?.buffer_depth ?? 0}
          icon={<Layers className="h-4 w-4" />}
          formatValue={formatNumber}
          isLoading={isLoading}
        />
        <MetricCard
          title="Errors"
          value={metrics?.error_count ?? 0}
          icon={<AlertTriangle className="h-4 w-4" />}
          formatValue={formatNumber}
          status={errorStatus}
          isLoading={isLoading}
        />
      </div>
    </div>
  )
}

function getLagStatus(lagSeconds: number): "normal" | "warning" | "critical" {
  if (lagSeconds <= 1) return "normal"
  if (lagSeconds <= 10) return "warning"
  return "critical"
}

function getErrorStatus(errorCount: number): "normal" | "warning" | "critical" {
  if (errorCount === 0) return "normal"
  if (errorCount <= 10) return "warning"
  return "critical"
}
