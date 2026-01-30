"use client"

import { useEffect, useState, useCallback, useRef } from "react"
import { Clock } from "lucide-react"

interface TimeEstimateProps {
  startedAt?: string
  estimatedRemainingMs: number
  showElapsed?: boolean
  showRemaining?: boolean
  className?: string
}

function formatDuration(ms: number): string {
  if (ms < 0) ms = 0

  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  }
  return `${seconds}s`
}

function calculateElapsed(startedAt: string | undefined): number {
  if (!startedAt) return 0
  return Math.max(0, Date.now() - new Date(startedAt).getTime())
}

export function TimeEstimate({
  startedAt,
  estimatedRemainingMs,
  showElapsed = true,
  showRemaining = true,
  className = "",
}: TimeEstimateProps) {
  const [elapsedMs, setElapsedMs] = useState(() => calculateElapsed(startedAt))
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const updateElapsed = useCallback(() => {
    setElapsedMs(calculateElapsed(startedAt))
  }, [startedAt])

  // Set up interval for elapsed time updates
  useEffect(() => {
    // Clear any existing interval
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }

    if (!startedAt) {
      // Schedule state reset outside effect body via timeout
      const timeoutId = setTimeout(() => setElapsedMs(0), 0)
      return () => clearTimeout(timeoutId)
    }

    // Schedule initial update outside effect body
    const initialTimeoutId = setTimeout(updateElapsed, 0)

    // Set up recurring interval
    intervalRef.current = setInterval(updateElapsed, 1000)

    return () => {
      clearTimeout(initialTimeoutId)
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [startedAt, updateElapsed])

  return (
    <div className={`flex items-center gap-4 text-sm text-muted-foreground ${className}`}>
      <Clock className="h-4 w-4" />
      {showElapsed && startedAt && (
        <span>
          Elapsed: <span className="font-medium text-foreground">{formatDuration(elapsedMs)}</span>
        </span>
      )}
      {showRemaining && estimatedRemainingMs > 0 && (
        <span>
          Remaining: <span className="font-medium text-foreground">~{formatDuration(estimatedRemainingMs)}</span>
        </span>
      )}
      {!startedAt && !estimatedRemainingMs && (
        <span>Waiting to start...</span>
      )}
    </div>
  )
}
