"use client"

import { Badge } from "@/components/ui/badge"
import type { PipelineStatus } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface PipelineStatusBadgeProps {
  status: PipelineStatus
  className?: string
}

export function PipelineStatusBadge({ status, className }: PipelineStatusBadgeProps) {
  const variants: Record<PipelineStatus, "default" | "secondary" | "destructive" | "outline"> = {
    running: "default",
    starting: "secondary",
    stopping: "secondary",
    stopped: "outline",
    error: "destructive",
  }

  const shouldPulse = status === "running" || status === "starting" || status === "stopping"

  return (
    <Badge variant={variants[status]} className={cn("gap-1.5", className)}>
      <span
        className={cn(
          "h-2 w-2 rounded-full",
          status === "running" && "bg-green-500",
          status === "starting" && "bg-yellow-500",
          status === "stopping" && "bg-yellow-500",
          status === "stopped" && "bg-gray-400",
          status === "error" && "bg-red-500",
          shouldPulse && "animate-pulse"
        )}
      />
      <span className="capitalize">{status}</span>
    </Badge>
  )
}
