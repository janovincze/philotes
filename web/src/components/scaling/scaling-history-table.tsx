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
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { TrendingUp, TrendingDown, Clock, User, ArrowRight } from "lucide-react"
import { formatRelativeTime } from "@/lib/utils/format-metrics"
import type { ScalingHistory, ScalingAction } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface ScalingHistoryTableProps {
  history: ScalingHistory[]
  isLoading?: boolean
  maxItems?: number
  showPolicyName?: boolean
  className?: string
}

const ACTION_CONFIG: Record<
  ScalingAction,
  { icon: typeof TrendingUp; color: string; label: string }
> = {
  scale_up: {
    icon: TrendingUp,
    color: "text-green-500",
    label: "Scaled Up",
  },
  scale_down: {
    icon: TrendingDown,
    color: "text-orange-500",
    label: "Scaled Down",
  },
  scheduled: {
    icon: Clock,
    color: "text-blue-500",
    label: "Scheduled",
  },
  manual: {
    icon: User,
    color: "text-purple-500",
    label: "Manual",
  },
}

export function ScalingHistoryTable({
  history,
  isLoading,
  maxItems,
  showPolicyName = true,
  className,
}: ScalingHistoryTableProps) {
  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="text-base font-medium">Scaling History</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center gap-4">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-4 w-32 ml-auto" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  const displayHistory = maxItems ? history.slice(0, maxItems) : history

  if (displayHistory.length === 0) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="text-base font-medium">Scaling History</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground text-center py-8">
            No scaling events recorded yet
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle className="text-base font-medium">Scaling History</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              {showPolicyName && <TableHead>Policy</TableHead>}
              <TableHead>Action</TableHead>
              <TableHead className="text-center">Replicas</TableHead>
              <TableHead>Reason</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {displayHistory.map((event) => {
              const config = ACTION_CONFIG[event.action]
              const Icon = config.icon

              return (
                <TableRow key={event.id}>
                  <TableCell className="text-muted-foreground">
                    {formatRelativeTime(event.executed_at)}
                  </TableCell>
                  {showPolicyName && (
                    <TableCell className="font-medium">{event.policy_name}</TableCell>
                  )}
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Icon className={cn("h-4 w-4", config.color)} />
                      <span>{config.label}</span>
                      {event.dry_run && (
                        <Badge variant="outline" className="text-xs">
                          Dry Run
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-center gap-2 font-mono">
                      <span>{event.previous_replicas}</span>
                      <ArrowRight className="h-3 w-3 text-muted-foreground" />
                      <span
                        className={cn(
                          event.new_replicas > event.previous_replicas && "text-green-500",
                          event.new_replicas < event.previous_replicas && "text-orange-500"
                        )}
                      >
                        {event.new_replicas}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground max-w-[200px] truncate">
                    {event.reason}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>

        {maxItems && history.length > maxItems && (
          <p className="text-xs text-muted-foreground text-center mt-4">
            Showing {maxItems} of {history.length} events
          </p>
        )}
      </CardContent>
    </Card>
  )
}
