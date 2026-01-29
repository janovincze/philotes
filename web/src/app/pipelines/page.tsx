"use client"

import { useState } from "react"
import Link from "next/link"
import { GitBranch, Plus, Play, Square, XCircle, Activity } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { usePipelines, useStartPipeline, useStopPipeline } from "@/lib/hooks/use-pipelines"
import { usePipelineMetrics } from "@/lib/hooks/use-metrics"
import { AutoRefreshToggle } from "@/components/pipelines/auto-refresh-toggle"
import { PipelineStatusBadge } from "@/components/pipelines/pipeline-status-badge"
import { formatEventsPerSecond } from "@/lib/utils/format-metrics"
import type { PipelineStatus } from "@/lib/api/types"

function PipelineCard({
  pipeline,
  refreshInterval,
}: {
  pipeline: {
    id: string
    name: string
    status: PipelineStatus
    tables: { id: string }[]
    error_message?: string
  }
  refreshInterval: number | false
}) {
  const startPipeline = useStartPipeline()
  const stopPipeline = useStopPipeline()

  const isRunning = pipeline.status === "running"
  const isStopped = pipeline.status === "stopped"
  const isTransitioning = pipeline.status === "starting" || pipeline.status === "stopping"

  // Only fetch metrics for running pipelines
  const { data: metricsResponse } = usePipelineMetrics(pipeline.id, {
    enabled: isRunning,
    refetchInterval: isRunning ? refreshInterval : false,
  })

  const metrics = metricsResponse?.metrics

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between space-y-0">
        <div className="flex items-start gap-4">
          <div className="rounded-lg bg-primary/10 p-2">
            <GitBranch className="h-6 w-6 text-primary" />
          </div>
          <div>
            <CardTitle className="text-lg">{pipeline.name}</CardTitle>
            <CardDescription>
              {pipeline.tables?.length ?? 0} table mappings
            </CardDescription>
          </div>
        </div>
        <PipelineStatusBadge status={pipeline.status} />
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Mini metrics for running pipelines */}
        {isRunning && metrics && (
          <div className="flex items-center gap-4 rounded-md bg-muted/50 px-3 py-2 text-sm">
            <div className="flex items-center gap-1.5">
              <Activity className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="font-mono">
                {formatEventsPerSecond(metrics.events_per_second)}
              </span>
              <span className="text-muted-foreground">events/s</span>
            </div>
            {metrics.error_count > 0 && (
              <Badge variant="destructive" className="text-xs">
                {metrics.error_count} errors
              </Badge>
            )}
          </div>
        )}
        {pipeline.error_message && (
          <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
            {pipeline.error_message}
          </div>
        )}
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href={`/pipelines/${pipeline.id}`}>View Details</Link>
          </Button>
          {isStopped && (
            <Button
              size="sm"
              onClick={() => startPipeline.mutate(pipeline.id)}
              disabled={startPipeline.isPending}
            >
              <Play className="mr-2 h-4 w-4" />
              Start
            </Button>
          )}
          {isRunning && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => stopPipeline.mutate(pipeline.id)}
              disabled={stopPipeline.isPending}
            >
              <Square className="mr-2 h-4 w-4" />
              Stop
            </Button>
          )}
          {isTransitioning && (
            <Button size="sm" disabled>
              {pipeline.status === "starting" ? "Starting..." : "Stopping..."}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function PipelinesListSkeleton() {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {[1, 2, 3, 4].map((i) => (
        <Card key={i}>
          <CardHeader className="flex flex-row items-start gap-4">
            <Skeleton className="h-10 w-10 rounded-lg" />
            <div className="space-y-2">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-24" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              <Skeleton className="h-9 w-24" />
              <Skeleton className="h-9 w-16" />
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export default function PipelinesPage() {
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true)
  const [refreshInterval, setRefreshInterval] = useState(5000)

  const { data: pipelines, isLoading, error } = usePipelines({
    refetchInterval: autoRefreshEnabled ? refreshInterval : false,
  })

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Pipelines</h1>
          <p className="text-muted-foreground">
            Manage your CDC replication pipelines
          </p>
        </div>
        <div className="flex items-center gap-2">
          <AutoRefreshToggle
            enabled={autoRefreshEnabled}
            interval={refreshInterval}
            onToggle={setAutoRefreshEnabled}
            onIntervalChange={setRefreshInterval}
          />
          <Button asChild>
            <Link href="/pipelines/new">
              <Plus className="mr-2 h-4 w-4" />
              New Pipeline
            </Link>
          </Button>
        </div>
      </div>

      {/* Pipelines list */}
      {isLoading ? (
        <PipelinesListSkeleton />
      ) : error ? (
        <Card>
          <CardContent className="py-8 text-center">
            <XCircle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-muted-foreground">
              Failed to load pipelines. Please try again.
            </p>
          </CardContent>
        </Card>
      ) : pipelines && pipelines.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2">
          {pipelines.map((pipeline) => (
            <PipelineCard
              key={pipeline.id}
              pipeline={pipeline}
              refreshInterval={autoRefreshEnabled ? refreshInterval : false}
            />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <GitBranch className="mx-auto h-12 w-12 text-muted-foreground" />
            <h3 className="mt-4 text-lg font-medium">No pipelines</h3>
            <p className="mt-2 text-muted-foreground">
              Create your first pipeline to start replicating data to Iceberg.
            </p>
            <Button className="mt-4" asChild>
              <Link href="/pipelines/new">
                <Plus className="mr-2 h-4 w-4" />
                New Pipeline
              </Link>
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
