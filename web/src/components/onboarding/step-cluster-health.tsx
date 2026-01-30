"use client"

import { useMemo } from "react"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import {
  CheckCircle2,
  XCircle,
  Loader2,
  Server,
  Database,
  HardDrive,
  Network,
} from "lucide-react"
import { useClusterHealth } from "@/lib/hooks/use-onboarding"
import { cn } from "@/lib/utils"

interface StepClusterHealthProps {
  onNext: () => void
}

interface HealthCheck {
  name: string
  label: string
  icon: React.ElementType
  ready: boolean
  critical: boolean
}

export function StepClusterHealth({ onNext }: StepClusterHealthProps) {
  const { data, isLoading, error } = useClusterHealth(true, 3000)

  // Derive hasChecked from data instead of using state + effect
  const hasChecked = useMemo(() => !!data, [data])

  const healthChecks: HealthCheck[] = [
    {
      name: "api",
      label: "API Server",
      icon: Server,
      ready: data?.api_ready ?? false,
      critical: true,
    },
    {
      name: "buffer_db",
      label: "Buffer Database",
      icon: Database,
      ready: data?.buffer_db_ready ?? false,
      critical: true,
    },
    {
      name: "minio",
      label: "MinIO (Object Storage)",
      icon: HardDrive,
      ready: data?.minio_ready ?? false,
      critical: false,
    },
    {
      name: "lakekeeper",
      label: "Lakekeeper (Iceberg Catalog)",
      icon: Network,
      ready: data?.lakekeeper_ready ?? false,
      critical: false,
    },
  ]

  const allCriticalReady = data?.all_critical_ready ?? false
  const allReady = healthChecks.every((check) => check.ready)

  const getOverallStatus = () => {
    if (!hasChecked) return "checking"
    if (allReady) return "healthy"
    if (allCriticalReady) return "degraded"
    return "unhealthy"
  }

  const overallStatus = getOverallStatus()

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">Cluster Health Check</h2>
        <p className="text-muted-foreground mt-2">
          Let&apos;s verify that all cluster components are healthy before proceeding with setup.
        </p>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>
            Failed to connect to the API server. Please ensure Philotes is running and try again.
          </AlertDescription>
        </Alert>
      )}

      <div className="space-y-4">
        {/* Overall Status */}
        <div
          className={cn(
            "p-4 rounded-lg border",
            overallStatus === "healthy" && "bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-800",
            overallStatus === "degraded" && "bg-yellow-50 border-yellow-200 dark:bg-yellow-950/20 dark:border-yellow-800",
            overallStatus === "unhealthy" && "bg-red-50 border-red-200 dark:bg-red-950/20 dark:border-red-800",
            overallStatus === "checking" && "bg-muted border-muted-foreground/20"
          )}
        >
          <div className="flex items-center gap-3">
            {overallStatus === "checking" && (
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            )}
            {overallStatus === "healthy" && (
              <CheckCircle2 className="h-6 w-6 text-green-600 dark:text-green-400" />
            )}
            {overallStatus === "degraded" && (
              <CheckCircle2 className="h-6 w-6 text-yellow-600 dark:text-yellow-400" />
            )}
            {overallStatus === "unhealthy" && (
              <XCircle className="h-6 w-6 text-red-600 dark:text-red-400" />
            )}
            <div>
              <p className="font-medium">
                {overallStatus === "checking" && "Checking cluster health..."}
                {overallStatus === "healthy" && "All components are healthy"}
                {overallStatus === "degraded" && "Some non-critical components are unavailable"}
                {overallStatus === "unhealthy" && "Critical components are unavailable"}
              </p>
              {overallStatus === "degraded" && (
                <p className="text-sm text-muted-foreground">
                  You can proceed, but some features may be limited.
                </p>
              )}
            </div>
          </div>
        </div>

        {/* Individual Components */}
        <div className="grid gap-3">
          {healthChecks.map((check) => {
            const Icon = check.icon
            return (
              <div
                key={check.name}
                className={cn(
                  "flex items-center justify-between p-3 rounded-lg border",
                  check.ready
                    ? "bg-green-50/50 border-green-200/50 dark:bg-green-950/10 dark:border-green-800/50"
                    : "bg-muted/50 border-muted-foreground/20"
                )}
              >
                <div className="flex items-center gap-3">
                  <Icon className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <p className="font-medium text-sm">{check.label}</p>
                    {check.critical && (
                      <p className="text-xs text-muted-foreground">Required</p>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {!hasChecked ? (
                    <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                  ) : check.ready ? (
                    <Badge variant="outline" className="bg-green-100 text-green-700 border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800">
                      Ready
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="bg-red-100 text-red-700 border-red-200 dark:bg-red-900/30 dark:text-red-400 dark:border-red-800">
                      Not Ready
                    </Badge>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      <div className="flex justify-end pt-4">
        <Button
          onClick={onNext}
          disabled={!allCriticalReady || isLoading}
          size="lg"
        >
          {!hasChecked ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Checking...
            </>
          ) : (
            "Continue"
          )}
        </Button>
      </div>
    </div>
  )
}
