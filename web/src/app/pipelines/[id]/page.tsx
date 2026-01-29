"use client"

import { useState } from "react"
import { use } from "react"
import Link from "next/link"
import { ArrowLeft, GitBranch, Play, Square } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { usePipeline, useStartPipeline, useStopPipeline } from "@/lib/hooks/use-pipelines"
import { usePipelineMetrics, usePipelineMetricsHistory } from "@/lib/hooks/use-metrics"
import { PipelineMetricsGrid } from "@/components/pipelines/pipeline-metrics-grid"
import { TableMetricsTable } from "@/components/pipelines/table-metrics"
import { AutoRefreshToggle } from "@/components/pipelines/auto-refresh-toggle"
import { PipelineStatusBadge } from "@/components/pipelines/pipeline-status-badge"
import { ErrorLogViewer, type PipelineError } from "@/components/pipelines/error-log-viewer"
import { MetricChart, TimeRangeSelector } from "@/components/metrics"
import { formatBytes, formatEventsPerSecond, formatNumber } from "@/lib/utils/format-metrics"
import type { PipelineStatus } from "@/lib/api/types"

interface PageProps {
  params: Promise<{ id: string }>
}

function PipelineHeader({
  pipeline,
  isLoading,
}: {
  pipeline?: {
    id: string
    name: string
    status: PipelineStatus
    error_message?: string
    started_at?: string
  }
  isLoading?: boolean
}) {
  const startPipeline = useStartPipeline()
  const stopPipeline = useStopPipeline()

  if (isLoading) {
    return (
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <Skeleton className="h-12 w-12 rounded-lg" />
          <div className="space-y-2">
            <Skeleton className="h-7 w-48" />
            <Skeleton className="h-5 w-32" />
          </div>
        </div>
        <Skeleton className="h-9 w-24" />
      </div>
    )
  }

  if (!pipeline) return null

  const isRunning = pipeline.status === "running"
  const isStopped = pipeline.status === "stopped"
  const isTransitioning = pipeline.status === "starting" || pipeline.status === "stopping"

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <div className="rounded-lg bg-primary/10 p-3">
            <GitBranch className="h-8 w-8 text-primary" />
          </div>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{pipeline.name}</h1>
              <PipelineStatusBadge status={pipeline.status} />
            </div>
            {pipeline.started_at && isRunning && (
              <p className="text-sm text-muted-foreground">
                Running since {new Date(pipeline.started_at).toLocaleString()}
              </p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isStopped && (
            <Button
              onClick={() => startPipeline.mutate(pipeline.id)}
              disabled={startPipeline.isPending}
            >
              <Play className="mr-2 h-4 w-4" />
              Start Pipeline
            </Button>
          )}
          {isRunning && (
            <Button
              variant="secondary"
              onClick={() => stopPipeline.mutate(pipeline.id)}
              disabled={stopPipeline.isPending}
            >
              <Square className="mr-2 h-4 w-4" />
              Stop
            </Button>
          )}
          {isTransitioning && (
            <Button disabled>
              {pipeline.status === "starting" ? "Starting..." : "Stopping..."}
            </Button>
          )}
        </div>
      </div>
      {pipeline.error_message && (
        <div className="rounded-md bg-destructive/10 p-4 text-sm text-destructive">
          <strong>Error:</strong> {pipeline.error_message}
        </div>
      )}
    </div>
  )
}

function SummaryCard({
  label,
  value,
  isLoading,
}: {
  label: string
  value: string
  isLoading?: boolean
}) {
  if (isLoading) {
    return (
      <div>
        <p className="text-sm text-muted-foreground">{label}</p>
        <Skeleton className="mt-1 h-6 w-20" />
      </div>
    )
  }
  return (
    <div>
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="font-mono font-medium">{value}</p>
    </div>
  )
}

export default function PipelineDetailPage({ params }: PageProps) {
  const { id } = use(params)
  const [timeRange, setTimeRange] = useState("1h")
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true)
  const [refreshInterval, setRefreshInterval] = useState(5000)

  const {
    data: pipeline,
    isLoading: isPipelineLoading,
    error: pipelineError,
  } = usePipeline(id)

  const {
    data: metricsResponse,
    isLoading: isMetricsLoading,
  } = usePipelineMetrics(id, {
    refetchInterval: autoRefreshEnabled ? refreshInterval : false,
  })

  const { data: historyResponse, isLoading: isHistoryLoading } =
    usePipelineMetricsHistory(id, timeRange)

  const metrics = metricsResponse?.metrics
  const history = historyResponse?.history

  // Placeholder for errors - will be populated when error log API is implemented
  const errors: PipelineError[] = []

  if (pipelineError) {
    return (
      <div className="space-y-6">
        <Link
          href="/pipelines"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Pipelines
        </Link>
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-destructive">
              Failed to load pipeline. It may have been deleted or you may not have access.
            </p>
            <Button asChild className="mt-4">
              <Link href="/pipelines">Return to Pipelines</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Link
        href="/pipelines"
        className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Pipelines
      </Link>

      {/* Pipeline header */}
      <PipelineHeader pipeline={pipeline} isLoading={isPipelineLoading} />

      {/* Controls bar */}
      <div className="flex items-center justify-between">
        <TimeRangeSelector value={timeRange} onChange={setTimeRange} />
        <AutoRefreshToggle
          enabled={autoRefreshEnabled}
          interval={refreshInterval}
          onToggle={setAutoRefreshEnabled}
          onIntervalChange={setRefreshInterval}
        />
      </div>

      {/* Metrics grid */}
      <PipelineMetricsGrid metrics={metrics} isLoading={isMetricsLoading} />

      {/* Summary stats */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base font-medium">Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-6 md:grid-cols-4">
            <SummaryCard
              label="Total Events"
              value={formatNumber(metrics?.events_processed ?? 0)}
              isLoading={isMetricsLoading}
            />
            <SummaryCard
              label="Iceberg Commits"
              value={String(metrics?.iceberg_commits ?? 0)}
              isLoading={isMetricsLoading}
            />
            <SummaryCard
              label="Data Written"
              value={formatBytes(metrics?.iceberg_bytes_written ?? 0)}
              isLoading={isMetricsLoading}
            />
            <SummaryCard
              label="Uptime"
              value={metrics?.uptime ?? "-"}
              isLoading={isMetricsLoading}
            />
          </div>
        </CardContent>
      </Card>

      {/* Charts section */}
      <div className="grid gap-4 lg:grid-cols-2">
        <MetricChart
          title="Events per Second"
          data={history?.data_points ?? []}
          dataKey="events_per_second"
          color="hsl(var(--primary))"
          unit="events/s"
          formatValue={formatEventsPerSecond}
          isLoading={isHistoryLoading}
        />
        <MetricChart
          title="Replication Lag"
          data={history?.data_points ?? []}
          dataKey="lag_seconds"
          color="hsl(var(--chart-2))"
          unit="s"
          formatValue={(v) => v.toFixed(2)}
          isLoading={isHistoryLoading}
        />
      </div>

      {/* Table metrics and Error log */}
      <div className="grid gap-4 lg:grid-cols-2">
        <TableMetricsTable
          tables={metrics?.tables ?? []}
          isLoading={isMetricsLoading}
        />
        <ErrorLogViewer
          errors={errors}
          errorCount={metrics?.error_count ?? 0}
          isLoading={isMetricsLoading}
        />
      </div>
    </div>
  )
}
