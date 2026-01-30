"use client"

import { useState } from "react"
import {
  CheckCircle,
  Loader2,
  Circle,
  XCircle,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
} from "lucide-react"
import { cn } from "@/lib/utils"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { SubStepItem } from "./sub-step-item"
import type { DeploymentStep, StepStatus, DeploymentLogMessage } from "@/lib/api/types"

interface StepItemProps {
  step: DeploymentStep
  logs?: DeploymentLogMessage[]
  isLast?: boolean
  className?: string
}

function StatusIcon({ status, size = "default" }: { status: StepStatus; size?: "default" | "large" }) {
  const className = size === "large" ? "h-6 w-6" : "h-5 w-5"

  switch (status) {
    case "completed":
      return <CheckCircle className={cn(className, "text-green-500")} />
    case "in_progress":
      return <Loader2 className={cn(className, "text-primary animate-spin")} />
    case "failed":
      return <XCircle className={cn(className, "text-red-500")} />
    case "skipped":
      return <AlertTriangle className={cn(className, "text-yellow-500")} />
    default:
      return <Circle className={cn(className, "text-muted-foreground/50")} />
  }
}

function formatElapsed(ms: number): string {
  if (!ms || ms <= 0) return ""
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  return `${minutes}m ${seconds % 60}s`
}

export function StepItem({ step, logs = [], isLast = false, className }: StepItemProps) {
  const [isOpen, setIsOpen] = useState(step.status === "in_progress" || step.status === "failed")
  const hasSubSteps = step.sub_steps && step.sub_steps.length > 0
  const hasLogs = logs.length > 0
  const isExpandable = hasSubSteps || hasLogs

  const stepLogs = logs.filter((log) => log.step === step.id)

  return (
    <div className={cn("relative", className)}>
      {/* Connector line */}
      {!isLast && (
        <div
          className={cn(
            "absolute left-[11px] top-8 w-0.5 h-[calc(100%-16px)]",
            step.status === "completed" ? "bg-green-500" : "bg-muted"
          )}
        />
      )}

      <Collapsible open={isOpen} onOpenChange={setIsOpen}>
        <CollapsibleTrigger
          className={cn(
            "flex items-start gap-3 w-full text-left py-2 hover:bg-muted/50 rounded-lg px-2 -mx-2 transition-colors",
            !isExpandable && "cursor-default hover:bg-transparent"
          )}
          disabled={!isExpandable}
        >
          {/* Status indicator */}
          <div
            className={cn(
              "flex-shrink-0 flex items-center justify-center rounded-full",
              step.status === "completed" && "bg-green-500/10",
              step.status === "in_progress" && "bg-primary/10",
              step.status === "failed" && "bg-red-500/10"
            )}
          >
            <StatusIcon status={step.status} />
          </div>

          {/* Step info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span
                className={cn(
                  "font-medium",
                  step.status === "pending" && "text-muted-foreground",
                  step.status === "in_progress" && "text-foreground",
                  step.status === "completed" && "text-foreground",
                  step.status === "failed" && "text-red-500"
                )}
              >
                {step.name}
              </span>
              {step.elapsed_time_ms && step.elapsed_time_ms > 0 && (
                <span className="text-xs text-muted-foreground">
                  {formatElapsed(step.elapsed_time_ms)}
                </span>
              )}
            </div>
            <p className="text-sm text-muted-foreground">{step.description}</p>
          </div>

          {/* Expand indicator */}
          {isExpandable && (
            <div className="flex-shrink-0 text-muted-foreground">
              {isOpen ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </div>
          )}
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className="ml-8 mt-1 space-y-1">
            {/* Sub-steps */}
            {hasSubSteps && (
              <div className="space-y-0.5">
                {step.sub_steps!.map((subStep, index) => (
                  <SubStepItem key={subStep.id || index} subStep={subStep} />
                ))}
              </div>
            )}

            {/* Logs for this step */}
            {stepLogs.length > 0 && (
              <div className="mt-2 rounded-md border bg-muted/30 p-2 font-mono text-xs max-h-[200px] overflow-y-auto">
                {stepLogs.map((log, index) => (
                  <div
                    key={index}
                    className={cn(
                      "py-0.5",
                      log.level === "error" && "text-red-500",
                      log.level === "warn" && "text-yellow-500"
                    )}
                  >
                    <span className="text-muted-foreground">
                      [{new Date(log.timestamp).toLocaleTimeString()}]
                    </span>{" "}
                    {log.message}
                  </div>
                ))}
              </div>
            )}

            {/* Error info */}
            {step.error && (
              <div className="mt-2 rounded-md border border-red-200 bg-red-50 dark:bg-red-950/20 dark:border-red-900 p-3">
                <p className="text-sm font-medium text-red-600 dark:text-red-400">
                  {step.error.message}
                </p>
                {step.error.details && (
                  <p className="mt-1 text-xs text-red-500 dark:text-red-400/80">
                    {step.error.details}
                  </p>
                )}
              </div>
            )}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  )
}
