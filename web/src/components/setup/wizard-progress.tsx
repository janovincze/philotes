"use client"

import { Check } from "lucide-react"
import { cn } from "@/lib/utils"

export interface WizardStep {
  id: number
  title: string
}

interface WizardProgressProps {
  steps: WizardStep[]
  currentStep: number
  className?: string
}

export function WizardProgress({ steps, currentStep, className }: WizardProgressProps) {
  return (
    <div className={cn("w-full", className)}>
      <div className="flex items-center justify-between">
        {steps.map((step, index) => {
          const isCompleted = step.id < currentStep
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
                    isCurrent && "border-primary bg-background text-primary",
                    !isCompleted && !isCurrent && "border-muted bg-background text-muted-foreground"
                  )}
                >
                  {isCompleted ? (
                    <Check className="h-5 w-5" />
                  ) : (
                    step.id
                  )}
                </div>
                <span
                  className={cn(
                    "mt-2 text-xs font-medium",
                    isCurrent && "text-primary",
                    !isCurrent && "text-muted-foreground"
                  )}
                >
                  {step.title}
                </span>
              </div>

              {/* Connector line */}
              {!isLast && (
                <div
                  className={cn(
                    "h-0.5 flex-1 mx-2 transition-colors",
                    isCompleted ? "bg-primary" : "bg-muted"
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
