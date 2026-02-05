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
  ArrowRight,
  Rocket,
} from "lucide-react"
import { useClusterHealth } from "@/lib/hooks/use-onboarding"
import { cn } from "@/lib/utils"

interface StepWelcomeProps {
  onNext: () => void
}

interface HealthCheck {
  name: string
  label: string
  icon: React.ElementType
  ready: boolean
  critical: boolean
}

export function StepWelcome({ onNext }: StepWelcomeProps) {
  const { data, isLoading, error } = useClusterHealth(true, 3000)

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
      label: "Object Storage",
      icon: HardDrive,
      ready: data?.minio_ready ?? false,
      critical: false,
    },
    {
      name: "lakekeeper",
      label: "Iceberg Catalog",
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
    <div className="space-y-8">
      {/* Welcome Header */}
      <div className="text-center space-y-4">
        <div className="flex justify-center">
          <div className="rounded-full bg-primary/10 p-4">
            <Rocket className="h-12 w-12 text-primary" />
          </div>
        </div>
        <div>
          <h2 className="text-2xl font-bold">Welcome to Philotes</h2>
          <p className="text-muted-foreground mt-2">
            Let&apos;s get your PostgreSQL data flowing to your data lake in just a few steps.
          </p>
        </div>
      </div>

      {/* What You'll Do */}
      <div className="bg-muted/50 rounded-lg p-6">
        <h3 className="font-semibold mb-4">In the next few minutes, you&apos;ll:</h3>
        <ol className="space-y-3">
          <li className="flex items-start gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">
              1
            </span>
            <span>Connect to your PostgreSQL database</span>
          </li>
          <li className="flex items-start gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">
              2
            </span>
            <span>Select which tables to replicate</span>
          </li>
          <li className="flex items-start gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">
              3
            </span>
            <span>Watch your data appear in the Iceberg data lake</span>
          </li>
          <li className="flex items-start gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">
              4
            </span>
            <span>Query your data with SQL</span>
          </li>
        </ol>
      </div>

      {/* Health Check Status */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold">System Status</h3>
          {isLoading && !hasChecked && (
            <Badge variant="outline" className="gap-1">
              <Loader2 className="h-3 w-3 animate-spin" />
              Checking...
            </Badge>
          )}
          {hasChecked && overallStatus === "healthy" && (
            <Badge variant="default" className="gap-1 bg-green-600">
              <CheckCircle2 className="h-3 w-3" />
              All Systems Ready
            </Badge>
          )}
          {hasChecked && overallStatus === "degraded" && (
            <Badge variant="secondary" className="gap-1">
              <CheckCircle2 className="h-3 w-3" />
              Ready (Degraded)
            </Badge>
          )}
          {hasChecked && overallStatus === "unhealthy" && (
            <Badge variant="destructive" className="gap-1">
              <XCircle className="h-3 w-3" />
              Not Ready
            </Badge>
          )}
        </div>

        <div className="grid grid-cols-2 gap-3">
          {healthChecks.map((check) => {
            const Icon = check.icon
            return (
              <div
                key={check.name}
                className={cn(
                  "flex items-center gap-3 rounded-lg border p-3",
                  check.ready && "border-green-200 bg-green-50 dark:border-green-900 dark:bg-green-950",
                  !check.ready && hasChecked && "border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950",
                  !hasChecked && "border-muted"
                )}
              >
                <Icon className={cn(
                  "h-5 w-5",
                  check.ready && "text-green-600 dark:text-green-400",
                  !check.ready && hasChecked && "text-red-600 dark:text-red-400",
                  !hasChecked && "text-muted-foreground"
                )} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{check.label}</p>
                  {check.critical && (
                    <p className="text-xs text-muted-foreground">Required</p>
                  )}
                </div>
                {!hasChecked && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
                {hasChecked && check.ready && <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />}
                {hasChecked && !check.ready && <XCircle className="h-4 w-4 text-red-600 dark:text-red-400" />}
              </div>
            )
          })}
        </div>

        {error && (
          <Alert variant="destructive">
            <XCircle className="h-4 w-4" />
            <AlertDescription>
              Unable to check system status. Please ensure all services are running.
            </AlertDescription>
          </Alert>
        )}

        {hasChecked && !allCriticalReady && (
          <Alert variant="destructive">
            <XCircle className="h-4 w-4" />
            <AlertDescription>
              Some required services are not ready. Please check your deployment and try again.
            </AlertDescription>
          </Alert>
        )}
      </div>

      {/* Action Button */}
      <div className="flex justify-end">
        <Button
          size="lg"
          onClick={onNext}
          disabled={!allCriticalReady}
          className="gap-2"
        >
          Get Started
          <ArrowRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
