"use client"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const DEFAULT_OPTIONS = [
  { label: "15m", value: "15m" },
  { label: "1h", value: "1h" },
  { label: "6h", value: "6h" },
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
]

interface TimeRangeSelectorProps {
  value: string
  onChange: (range: string) => void
  options?: Array<{ label: string; value: string }>
  className?: string
}

export function TimeRangeSelector({
  value,
  onChange,
  options = DEFAULT_OPTIONS,
  className,
}: TimeRangeSelectorProps) {
  return (
    <div className={cn("inline-flex rounded-lg border p-1", className)}>
      {options.map((option) => (
        <Button
          key={option.value}
          variant={value === option.value ? "secondary" : "ghost"}
          size="sm"
          className={cn(
            "h-7 px-3 text-xs",
            value === option.value && "bg-secondary"
          )}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </Button>
      ))}
    </div>
  )
}
