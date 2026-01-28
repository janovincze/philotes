"use client"

import {
  Activity,
  Database,
  GitBranch,
  CheckCircle,
  XCircle,
  AlertCircle,
} from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useHealth } from "@/lib/hooks/use-health"
import { useSources } from "@/lib/hooks/use-sources"
import { usePipelines } from "@/lib/hooks/use-pipelines"
import type { HealthStatus, PipelineStatus } from "@/lib/api/types"

function HealthStatusIcon({ status }: { status: HealthStatus }) {
  switch (status) {
    case "healthy":
      return <CheckCircle className="h-5 w-5 text-green-500" />
    case "degraded":
      return <AlertCircle className="h-5 w-5 text-yellow-500" />
    case "unhealthy":
      return <XCircle className="h-5 w-5 text-red-500" />
    default:
      return <AlertCircle className="h-5 w-5 text-muted-foreground" />
  }
}

function PipelineStatusBadge({ status }: { status: PipelineStatus }) {
  const variants: Record<PipelineStatus, "default" | "secondary" | "destructive" | "outline"> = {
    running: "default",
    starting: "secondary",
    stopping: "secondary",
    stopped: "outline",
    error: "destructive",
  }

  return (
    <Badge variant={variants[status]} className="capitalize">
      {status}
    </Badge>
  )
}

export default function DashboardPage() {
  const { data: health, isLoading: healthLoading } = useHealth()
  const { data: sources, isLoading: sourcesLoading } = useSources()
  const { data: pipelines, isLoading: pipelinesLoading } = usePipelines()

  const runningPipelines = pipelines?.filter((p) => p.status === "running").length ?? 0
  const activeSources = sources?.filter((s) => s.status === "active").length ?? 0

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">
          Overview of your CDC pipelines and data sources
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {/* Health status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">System Health</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {healthLoading ? (
              <Skeleton className="h-8 w-24" />
            ) : (
              <div className="flex items-center gap-2">
                <HealthStatusIcon status={health?.status ?? "unknown"} />
                <span className="text-2xl font-bold capitalize">
                  {health?.status ?? "Unknown"}
                </span>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Sources */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Data Sources</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {sourcesLoading ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <>
                <div className="text-2xl font-bold">{sources?.length ?? 0}</div>
                <p className="text-xs text-muted-foreground">
                  {activeSources} active
                </p>
              </>
            )}
          </CardContent>
        </Card>

        {/* Pipelines */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Pipelines</CardTitle>
            <GitBranch className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {pipelinesLoading ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <>
                <div className="text-2xl font-bold">{pipelines?.length ?? 0}</div>
                <p className="text-xs text-muted-foreground">
                  {runningPipelines} running
                </p>
              </>
            )}
          </CardContent>
        </Card>

        {/* Components health */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Components</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {healthLoading ? (
              <Skeleton className="h-8 w-16" />
            ) : (
              <>
                <div className="text-2xl font-bold">
                  {health?.components ? Object.keys(health.components).length : 0}
                </div>
                <p className="text-xs text-muted-foreground">
                  {health?.components
                    ? Object.values(health.components).filter(
                        (c) => c.status === "healthy"
                      ).length
                    : 0}{" "}
                  healthy
                </p>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Recent pipelines */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Pipelines</CardTitle>
          <CardDescription>
            Your most recently updated CDC pipelines
          </CardDescription>
        </CardHeader>
        <CardContent>
          {pipelinesLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : pipelines && pipelines.length > 0 ? (
            <div className="space-y-3">
              {pipelines.slice(0, 5).map((pipeline) => (
                <div
                  key={pipeline.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div className="flex items-center gap-3">
                    <GitBranch className="h-4 w-4 text-muted-foreground" />
                    <div>
                      <p className="font-medium">{pipeline.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {pipeline.tables?.length ?? 0} tables
                      </p>
                    </div>
                  </div>
                  <PipelineStatusBadge status={pipeline.status} />
                </div>
              ))}
            </div>
          ) : (
            <div className="py-8 text-center text-muted-foreground">
              No pipelines yet. Create your first pipeline to get started.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
