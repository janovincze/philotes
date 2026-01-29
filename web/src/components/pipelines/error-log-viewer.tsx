"use client"

import { useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { AlertTriangle, RefreshCw } from "lucide-react"
import { cn } from "@/lib/utils"
import { formatRelativeTime } from "@/lib/utils/format-metrics"

export interface PipelineError {
  id: string
  timestamp: string
  type: "replication" | "schema" | "connection" | "write" | "unknown"
  message: string
  table?: string
}

interface ErrorLogViewerProps {
  errors: PipelineError[]
  errorCount?: number
  isLoading?: boolean
  onRefresh?: () => void
  className?: string
}

const ERROR_TYPE_LABELS: Record<PipelineError["type"], string> = {
  replication: "Replication",
  schema: "Schema",
  connection: "Connection",
  write: "Write",
  unknown: "Unknown",
}

const ERROR_TYPE_COLORS: Record<PipelineError["type"], string> = {
  replication: "border-orange-500/50 text-orange-600 dark:text-orange-400",
  schema: "border-purple-500/50 text-purple-600 dark:text-purple-400",
  connection: "border-red-500/50 text-red-600 dark:text-red-400",
  write: "border-yellow-500/50 text-yellow-600 dark:text-yellow-400",
  unknown: "border-gray-500/50 text-gray-600 dark:text-gray-400",
}

export function ErrorLogViewer({
  errors,
  errorCount = 0,
  isLoading,
  onRefresh,
  className,
}: ErrorLogViewerProps) {
  const [typeFilter, setTypeFilter] = useState<string>("all")

  const filteredErrors =
    typeFilter === "all"
      ? errors
      : errors.filter((e) => e.type === typeFilter)

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-base font-medium">Error Log</CardTitle>
          <Skeleton className="h-8 w-24" />
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div className="flex items-center gap-2">
          <CardTitle className="text-base font-medium">Error Log</CardTitle>
          {errorCount > 0 && (
            <Badge variant="destructive" className="text-xs">
              {errorCount}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Select value={typeFilter} onValueChange={setTypeFilter}>
            <SelectTrigger className="h-8 w-[130px]">
              <SelectValue placeholder="Filter type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All types</SelectItem>
              <SelectItem value="replication">Replication</SelectItem>
              <SelectItem value="schema">Schema</SelectItem>
              <SelectItem value="connection">Connection</SelectItem>
              <SelectItem value="write">Write</SelectItem>
              <SelectItem value="unknown">Unknown</SelectItem>
            </SelectContent>
          </Select>
          {onRefresh && (
            <Button variant="outline" size="icon" className="h-8 w-8" onClick={onRefresh}>
              <RefreshCw className="h-4 w-4" />
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {filteredErrors.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <AlertTriangle className="h-8 w-8 text-muted-foreground/50 mb-2" />
            <p className="text-sm text-muted-foreground">
              {errors.length === 0
                ? "No errors recorded"
                : "No errors match the current filter"}
            </p>
          </div>
        ) : (
          <div className="space-y-3 max-h-80 overflow-y-auto">
            {filteredErrors.map((error) => (
              <ErrorLogEntry key={error.id} error={error} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ErrorLogEntry({ error }: { error: PipelineError }) {
  return (
    <div className="flex flex-col gap-1 rounded-md border p-3 bg-muted/30">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Badge
            variant="outline"
            className={cn("text-xs", ERROR_TYPE_COLORS[error.type])}
          >
            {ERROR_TYPE_LABELS[error.type]}
          </Badge>
          {error.table && (
            <span className="text-xs text-muted-foreground font-mono">
              {error.table}
            </span>
          )}
        </div>
        <span className="text-xs text-muted-foreground">
          {formatRelativeTime(error.timestamp)}
        </span>
      </div>
      <p className="text-sm text-foreground/90">{error.message}</p>
    </div>
  )
}
