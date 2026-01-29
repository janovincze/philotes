"use client"

import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { TrendingUp, TrendingDown, Minus } from "lucide-react"
import { cn } from "@/lib/utils"

interface MetricCardProps {
  title: string
  value: number | string
  unit?: string
  icon?: React.ReactNode
  trend?: "up" | "down" | "stable"
  status?: "normal" | "warning" | "critical"
  formatValue?: (value: number) => string
  isLoading?: boolean
  className?: string
}

export function MetricCard({
  title,
  value,
  unit,
  icon,
  trend,
  status = "normal",
  formatValue,
  isLoading,
  className,
}: MetricCardProps) {
  const displayValue =
    typeof value === "number" && formatValue ? formatValue(value) : value

  const statusColors = {
    normal: "text-foreground",
    warning: "text-yellow-600 dark:text-yellow-400",
    critical: "text-red-600 dark:text-red-400",
  }

  const trendIcons = {
    up: <TrendingUp className="h-4 w-4 text-green-500" />,
    down: <TrendingDown className="h-4 w-4 text-red-500" />,
    stable: <Minus className="h-4 w-4 text-muted-foreground" />,
  }

  if (isLoading) {
    return (
      <Card className={className}>
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <Skeleton className="h-4 w-24" />
            {icon && <Skeleton className="h-5 w-5 rounded" />}
          </div>
          <Skeleton className="mt-3 h-8 w-20" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardContent className="p-6">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-muted-foreground">
            {title}
          </span>
          {icon && (
            <div className="text-muted-foreground">{icon}</div>
          )}
        </div>
        <div className="mt-2 flex items-baseline gap-2">
          <span
            className={cn(
              "text-3xl font-bold tracking-tight",
              statusColors[status]
            )}
          >
            {displayValue}
          </span>
          {unit && (
            <span className="text-sm text-muted-foreground">{unit}</span>
          )}
          {trend && <span className="ml-auto">{trendIcons[trend]}</span>}
        </div>
      </CardContent>
    </Card>
  )
}

export function MetricCardSkeleton({ className }: { className?: string }) {
  return (
    <Card className={className}>
      <CardContent className="p-6">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="mt-3 h-8 w-20" />
      </CardContent>
    </Card>
  )
}
