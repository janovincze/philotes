"use client"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { RefreshCw, ChevronDown } from "lucide-react"
import { cn } from "@/lib/utils"

const INTERVAL_OPTIONS = [
  { label: "5s", value: 5000 },
  { label: "15s", value: 15000 },
  { label: "30s", value: 30000 },
  { label: "60s", value: 60000 },
]

interface AutoRefreshToggleProps {
  enabled: boolean
  interval: number
  onToggle: (enabled: boolean) => void
  onIntervalChange: (interval: number) => void
  className?: string
}

export function AutoRefreshToggle({
  enabled,
  interval,
  onToggle,
  onIntervalChange,
  className,
}: AutoRefreshToggleProps) {
  const currentOption = INTERVAL_OPTIONS.find((o) => o.value === interval)
  const label = currentOption?.label ?? `${interval / 1000}s`

  return (
    <div className={cn("flex items-center gap-1", className)}>
      <Button
        variant={enabled ? "default" : "outline"}
        size="sm"
        onClick={() => onToggle(!enabled)}
        className="gap-2"
      >
        <RefreshCw
          className={cn("h-4 w-4", enabled && "animate-spin")}
          style={{ animationDuration: "2s" }}
        />
        <span className="hidden sm:inline">Auto-refresh</span>
        {enabled && <span className="text-xs opacity-75">{label}</span>}
      </Button>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="px-2">
            <ChevronDown className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {INTERVAL_OPTIONS.map((option) => (
            <DropdownMenuItem
              key={option.value}
              onClick={() => {
                onIntervalChange(option.value)
                if (!enabled) onToggle(true)
              }}
              className={cn(
                interval === option.value && enabled && "bg-accent"
              )}
            >
              Every {option.label}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
