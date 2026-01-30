"use client"

import { cn } from "@/lib/utils"
import { StepItem } from "./step-item"
import type { DeploymentStep, DeploymentLogMessage } from "@/lib/api/types"

interface DeploymentStepsProps {
  steps: DeploymentStep[]
  logs?: DeploymentLogMessage[]
  className?: string
}

export function DeploymentSteps({ steps, logs = [], className }: DeploymentStepsProps) {
  if (!steps || steps.length === 0) {
    return (
      <div className={cn("text-center text-muted-foreground py-8", className)}>
        No deployment steps available
      </div>
    )
  }

  return (
    <div className={cn("space-y-1", className)}>
      {steps.map((step, index) => (
        <StepItem
          key={step.id}
          step={step}
          logs={logs}
          isLast={index === steps.length - 1}
        />
      ))}
    </div>
  )
}
