"use client"

import { cn } from "@/lib/utils"

interface ProgressBarProps {
  value: number
  size?: "sm" | "md" | "lg"
  variant?: "default" | "success" | "error"
  showLabel?: boolean
  className?: string
}

const sizeConfig = {
  sm: { dimension: 60, strokeWidth: 4, fontSize: "text-sm" },
  md: { dimension: 80, strokeWidth: 5, fontSize: "text-lg" },
  lg: { dimension: 120, strokeWidth: 6, fontSize: "text-2xl" },
}

const variantConfig = {
  default: "stroke-primary",
  success: "stroke-green-500",
  error: "stroke-red-500",
}

export function CircularProgress({
  value,
  size = "md",
  variant = "default",
  showLabel = true,
  className,
}: ProgressBarProps) {
  const { dimension, strokeWidth, fontSize } = sizeConfig[size]
  const radius = (dimension - strokeWidth) / 2
  const circumference = radius * 2 * Math.PI
  const offset = circumference - (Math.min(100, Math.max(0, value)) / 100) * circumference

  return (
    <div className={cn("relative inline-flex items-center justify-center", className)}>
      <svg
        width={dimension}
        height={dimension}
        className="transform -rotate-90"
      >
        {/* Background circle */}
        <circle
          cx={dimension / 2}
          cy={dimension / 2}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          className="text-muted"
        />
        {/* Progress circle */}
        <circle
          cx={dimension / 2}
          cy={dimension / 2}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className={cn(
            "transition-all duration-500 ease-out",
            variantConfig[variant]
          )}
        />
      </svg>
      {showLabel && (
        <span
          className={cn(
            "absolute font-semibold",
            fontSize
          )}
        >
          {Math.round(value)}%
        </span>
      )}
    </div>
  )
}

interface LinearProgressProps {
  value: number
  variant?: "default" | "success" | "error"
  showLabel?: boolean
  className?: string
}

const linearVariantConfig = {
  default: "bg-primary",
  success: "bg-green-500",
  error: "bg-red-500",
}

export function LinearProgress({
  value,
  variant = "default",
  showLabel = true,
  className,
}: LinearProgressProps) {
  const normalizedValue = Math.min(100, Math.max(0, value))

  return (
    <div className={cn("w-full", className)}>
      {showLabel && (
        <div className="flex justify-between mb-1 text-sm">
          <span className="text-muted-foreground">Progress</span>
          <span className="font-medium">{Math.round(normalizedValue)}%</span>
        </div>
      )}
      <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
        <div
          className={cn(
            "h-full rounded-full transition-all duration-500 ease-out",
            linearVariantConfig[variant]
          )}
          style={{ width: `${normalizedValue}%` }}
        />
      </div>
    </div>
  )
}
