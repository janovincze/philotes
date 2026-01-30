"use client"

import { CheckCircle, Loader2, Circle, XCircle } from "lucide-react"
import { cn } from "@/lib/utils"
import type { SubStep, StepStatus } from "@/lib/api/types"

interface SubStepItemProps {
  subStep: SubStep
  className?: string
}

function StatusIcon({ status }: { status: StepStatus }) {
  switch (status) {
    case "completed":
      return <CheckCircle className="h-3.5 w-3.5 text-green-500" />
    case "in_progress":
      return <Loader2 className="h-3.5 w-3.5 text-primary animate-spin" />
    case "failed":
      return <XCircle className="h-3.5 w-3.5 text-red-500" />
    case "skipped":
      return <Circle className="h-3.5 w-3.5 text-muted-foreground" />
    default:
      return <Circle className="h-3.5 w-3.5 text-muted-foreground/50" />
  }
}

export function SubStepItem({ subStep, className }: SubStepItemProps) {
  const showCounter = subStep.total && subStep.total > 1

  return (
    <div
      className={cn(
        "flex items-center gap-2 py-1 pl-6 text-sm",
        subStep.status === "pending" && "text-muted-foreground",
        subStep.status === "in_progress" && "text-foreground",
        subStep.status === "completed" && "text-muted-foreground",
        subStep.status === "failed" && "text-red-500",
        className
      )}
    >
      <StatusIcon status={subStep.status} />
      <span>{subStep.name}</span>
      {showCounter && (
        <span className="text-xs text-muted-foreground">
          ({subStep.current} of {subStep.total})
        </span>
      )}
      {subStep.details && (
        <span className="text-xs text-muted-foreground truncate max-w-[200px]">
          - {subStep.details}
        </span>
      )}
    </div>
  )
}
