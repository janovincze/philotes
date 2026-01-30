"use client"

import { Check, SkipForward } from "lucide-react"
import { cn } from "@/lib/utils"

export interface OnboardingStepInfo {
  id: number
  title: string
  optional?: boolean
}

interface OnboardingProgressProps {
  steps: OnboardingStepInfo[]
  currentStep: number
  completedSteps: number[]
  skippedSteps: number[]
  className?: string
}

export function OnboardingProgress({
  steps,
  currentStep,
  completedSteps,
  skippedSteps,
  className,
}: OnboardingProgressProps) {
  return (
    <div className={cn("w-full", className)}>
      <div className="flex items-center justify-between">
        {steps.map((step, index) => {
          const isCompleted = completedSteps.includes(step.id)
          const isSkipped = skippedSteps.includes(step.id)
          const isCurrent = step.id === currentStep
          const isLast = index === steps.length - 1

          return (
            <div key={step.id} className="flex items-center flex-1 last:flex-none">
              {/* Step circle */}
              <div className="flex flex-col items-center">
                <div
                  className={cn(
                    "flex h-10 w-10 items-center justify-center rounded-full border-2 text-sm font-medium transition-colors",
                    isCompleted && "border-primary bg-primary text-primary-foreground",
                    isSkipped && "border-muted bg-muted text-muted-foreground",
                    isCurrent && !isCompleted && !isSkipped && "border-primary bg-background text-primary",
                    !isCompleted && !isSkipped && !isCurrent && "border-muted bg-background text-muted-foreground"
                  )}
                >
                  {isCompleted ? (
                    <Check className="h-5 w-5" />
                  ) : isSkipped ? (
                    <SkipForward className="h-4 w-4" />
                  ) : (
                    step.id
                  )}
                </div>
                <div className="flex flex-col items-center mt-2">
                  <span
                    className={cn(
                      "text-xs font-medium",
                      isCurrent && "text-primary",
                      !isCurrent && "text-muted-foreground"
                    )}
                  >
                    {step.title}
                  </span>
                  {step.optional && (
                    <span className="text-[10px] text-muted-foreground">(Optional)</span>
                  )}
                </div>
              </div>

              {/* Connector line */}
              {!isLast && (
                <div
                  className={cn(
                    "h-0.5 flex-1 mx-2 transition-colors",
                    isCompleted || isSkipped ? "bg-primary" : "bg-muted"
                  )}
                />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
