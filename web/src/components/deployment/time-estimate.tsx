"use client"

import { useEffect, useState } from "react"
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

export function TimeEstimate({
  startedAt,
  estimatedRemainingMs,
  showElapsed = true,
  showRemaining = true,
  className = "",
}: TimeEstimateProps) {
  const [elapsedMs, setElapsedMs] = useState(0)

  useEffect(() => {
    if (!startedAt) {
      // Schedule reset to avoid direct setState in effect body
      queueMicrotask(() => setElapsedMs(0))
      return
    }

    const startTime = new Date(startedAt).getTime()

    const updateElapsed = () => {
      setElapsedMs(Date.now() - startTime)
    }

    // Initial update via microtask
    queueMicrotask(updateElapsed)
    const interval = setInterval(updateElapsed, 1000)

    return () => clearInterval(interval)
  }, [startedAt])

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
