"use client"

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { formatNumber, formatLatency, formatRelativeTime } from "@/lib/utils/format-metrics"
import type { TableMetrics as TableMetricsType } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface TableMetricsProps {
  tables: TableMetricsType[]
  isLoading?: boolean
  className?: string
}

export function TableMetricsTable({
  tables,
  isLoading,
  className,
}: TableMetricsProps) {
  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="text-base font-medium">Table Metrics</CardTitle>
        </CardHeader>
        <CardContent>
          <TableMetricsSkeleton />
        </CardContent>
      </Card>
    )
  }

  if (!tables || tables.length === 0) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="text-base font-medium">Table Metrics</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground text-center py-8">
            No tables configured for this pipeline
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle className="text-base font-medium">Table Metrics</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Table</TableHead>
              <TableHead className="text-right">Events</TableHead>
              <TableHead className="text-right">Lag</TableHead>
              <TableHead className="text-right">Last Event</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tables.map((table) => (
              <TableRow key={`${table.schema}.${table.table}`}>
                <TableCell className="font-medium">
                  <span className="text-muted-foreground">{table.schema}.</span>
                  {table.table}
                </TableCell>
                <TableCell className="text-right font-mono">
                  {formatNumber(table.events_processed)}
                </TableCell>
                <TableCell className="text-right">
                  <LagBadge lagSeconds={table.lag_seconds} />
                </TableCell>
                <TableCell className="text-right text-muted-foreground">
                  {table.last_event_at
                    ? formatRelativeTime(table.last_event_at)
                    : "—"}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}

function LagBadge({ lagSeconds }: { lagSeconds: number }) {
  const status =
    lagSeconds <= 1 ? "normal" : lagSeconds <= 10 ? "warning" : "critical"

  return (
    <Badge
      variant="outline"
      className={cn(
        "font-mono",
        status === "normal" && "border-green-500/50 text-green-600 dark:text-green-400",
        status === "warning" && "border-yellow-500/50 text-yellow-600 dark:text-yellow-400",
        status === "critical" && "border-red-500/50 text-red-600 dark:text-red-400"
      )}
    >
      {formatLatency(lagSeconds)}
    </Badge>
  )
}

function TableMetricsSkeleton() {
  return (
    <div className="space-y-3">
      {[1, 2, 3].map((i) => (
        <div key={i} className="flex items-center justify-between">
          <Skeleton className="h-4 w-32" />
          <div className="flex gap-4">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-20" />
          </div>
        </div>
      ))}
    </div>
  )
}
