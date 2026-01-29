"use client"

import Link from "next/link"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Scale,
  MoreVertical,
  Play,
  Pause,
  Pencil,
  Trash2,
  Clock,
  TrendingUp,
  TrendingDown,
} from "lucide-react"
import type { ScalingPolicy, ScalingTargetType } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface ScalingPolicyCardProps {
  policy: ScalingPolicy
  onEnable?: (id: string) => void
  onDisable?: (id: string) => void
  onDelete?: (id: string) => void
  mutatingId?: string | null
}

export const TARGET_TYPE_LABELS: Record<ScalingTargetType, string> = {
  "cdc-worker": "CDC Worker",
  trino: "Trino",
  risingwave: "RisingWave",
  nodes: "Infrastructure Nodes",
}

export function ScalingPolicyCard({
  policy,
  onEnable,
  onDisable,
  onDelete,
  mutatingId,
}: ScalingPolicyCardProps) {
  const isPending = mutatingId === policy.id

  return (
    <Card className={cn(!policy.enabled && "opacity-60")}>
      <CardHeader className="flex flex-row items-start justify-between space-y-0">
        <div className="flex items-start gap-4">
          <div className="rounded-lg bg-primary/10 p-2">
            <Scale className="h-6 w-6 text-primary" />
          </div>
          <div>
            <CardTitle className="text-lg">{policy.name}</CardTitle>
            <CardDescription>
              {TARGET_TYPE_LABELS[policy.target_type]}
            </CardDescription>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={policy.enabled ? "default" : "secondary"}>
            {policy.enabled ? "Active" : "Disabled"}
          </Badge>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem asChild>
                <Link href={`/scaling/${policy.id}/edit`}>
                  <Pencil className="mr-2 h-4 w-4" />
                  Edit
                </Link>
              </DropdownMenuItem>
              {policy.enabled ? (
                <DropdownMenuItem
                  onClick={() => onDisable?.(policy.id)}
                  disabled={isPending}
                >
                  <Pause className="mr-2 h-4 w-4" />
                  Disable
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem
                  onClick={() => onEnable?.(policy.id)}
                  disabled={isPending}
                >
                  <Play className="mr-2 h-4 w-4" />
                  Enable
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => onDelete?.(policy.id)}
                disabled={isPending}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Replica limits */}
        <div className="flex items-center gap-6 text-sm">
          <div>
            <span className="text-muted-foreground">Min: </span>
            <span className="font-medium">{policy.min_replicas}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Max: </span>
            <span className="font-medium">{policy.max_replicas}</span>
          </div>
          {policy.scale_to_zero && (
            <Badge variant="outline" className="text-xs">
              Scale to Zero
            </Badge>
          )}
        </div>

        {/* Rules and schedules summary */}
        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          {policy.scale_up_rules.length > 0 && (
            <div className="flex items-center gap-1">
              <TrendingUp className="h-3.5 w-3.5 text-green-500" />
              <span>{policy.scale_up_rules.length} up rules</span>
            </div>
          )}
          {policy.scale_down_rules.length > 0 && (
            <div className="flex items-center gap-1">
              <TrendingDown className="h-3.5 w-3.5 text-orange-500" />
              <span>{policy.scale_down_rules.length} down rules</span>
            </div>
          )}
          {policy.schedules.length > 0 && (
            <div className="flex items-center gap-1">
              <Clock className="h-3.5 w-3.5" />
              <span>{policy.schedules.length} schedules</span>
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href={`/scaling/${policy.id}`}>View Details</Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export function ScalingPolicyCardSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-start gap-4">
        <Skeleton className="h-10 w-10 rounded-lg" />
        <div className="space-y-2">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-4 w-24" />
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <Skeleton className="h-4 w-40" />
          <div className="flex gap-2">
            <Skeleton className="h-9 w-24" />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
