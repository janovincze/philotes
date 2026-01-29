"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Activity, TrendingUp, TrendingDown, Minus } from "lucide-react"
import { formatRelativeTime } from "@/lib/utils/format-metrics"
import type { ScalingState, ScalingPolicy } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface ScaleStateCardProps {
  state: ScalingState | undefined
  policy: ScalingPolicy | undefined
  isLoading?: boolean
  className?: string
}

export function ScaleStateCard({
  state,
  policy,
  isLoading,
  className,
}: ScaleStateCardProps) {
  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader className="pb-2">
          <CardTitle className="text-base font-medium">Current Scale</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4">
            <Skeleton className="h-16 w-16 rounded-lg" />
            <div className="space-y-2">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-4 w-24" />
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  const currentReplicas = state?.current_replicas ?? 0
  const minReplicas = policy?.min_replicas ?? 0
  const maxReplicas = policy?.max_replicas ?? 1
  const lastAction = state?.last_scale_action
  const lastScaleTime = state?.last_scale_time

  const ActionIcon = lastAction === "scale_up"
    ? TrendingUp
    : lastAction === "scale_down"
      ? TrendingDown
      : Minus

  const actionColor = lastAction === "scale_up"
    ? "text-green-500"
    : lastAction === "scale_down"
      ? "text-orange-500"
      : "text-muted-foreground"

  // Calculate progress percentage
  const range = maxReplicas - minReplicas
  const progress = range > 0
    ? ((currentReplicas - minReplicas) / range) * 100
    : 0

  return (
    <Card className={className}>
      <CardHeader className="pb-2">
        <CardTitle className="text-base font-medium flex items-center gap-2">
          <Activity className="h-4 w-4" />
          Current Scale
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-6">
          {/* Large replica count */}
          <div className="text-center">
            <div className="text-4xl font-bold">{currentReplicas}</div>
            <div className="text-sm text-muted-foreground">replicas</div>
          </div>

          {/* Scale bar and info */}
          <div className="flex-1 space-y-3">
            {/* Progress bar */}
            <div className="space-y-1">
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>Min: {minReplicas}</span>
                <span>Max: {maxReplicas}</span>
              </div>
              <div className="h-2 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary transition-all duration-300"
                  style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
                />
              </div>
            </div>

            {/* Last action */}
            {lastAction && (
              <div className="flex items-center gap-2 text-sm">
                <ActionIcon className={cn("h-4 w-4", actionColor)} />
                <span className="capitalize">{lastAction.replace("_", " ")}</span>
                {lastScaleTime && (
                  <span className="text-muted-foreground">
                    {formatRelativeTime(lastScaleTime)}
                  </span>
                )}
              </div>
            )}

            {!lastAction && (
              <div className="text-sm text-muted-foreground">
                No scaling activity yet
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
