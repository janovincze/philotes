/**
 * Format a large number with K, M, B suffixes
 */
export function formatNumber(n: number): string {
  if (n === 0) return "0"
  if (n < 0) return `-${formatNumber(-n)}`

  if (n >= 1_000_000_000) {
    return `${(n / 1_000_000_000).toFixed(1).replace(/\.0$/, "")}B`
  }
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1).replace(/\.0$/, "")}K`
  }
  if (n < 1 && n > 0) {
    return n.toFixed(2)
  }
  return Math.round(n).toString()
}

/**
 * Format bytes to human-readable string
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  if (bytes < 0) return `-${formatBytes(-bytes)}`

  const units = ["B", "KB", "MB", "GB", "TB", "PB"]
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / Math.pow(1024, exponent)

  return `${value.toFixed(value < 10 && exponent > 0 ? 1 : 0)} ${units[exponent]}`
}

/**
 * Format duration in seconds to human-readable string
 */
export function formatDuration(seconds: number): string {
  if (seconds < 0) return `-${formatDuration(-seconds)}`
  if (seconds === 0) return "0s"

  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  const parts: string[] = []
  if (days > 0) parts.push(`${days}d`)
  if (hours > 0) parts.push(`${hours}h`)
  if (minutes > 0) parts.push(`${minutes}m`)
  if (secs > 0 || parts.length === 0) parts.push(`${secs}s`)

  return parts.slice(0, 2).join(" ")
}

/**
 * Format latency/lag in seconds to human-readable string
 */
export function formatLatency(seconds: number): string {
  if (seconds < 0) return `-${formatLatency(-seconds)}`
  if (seconds === 0) return "0ms"

  if (seconds < 0.001) {
    return "< 1ms"
  }
  if (seconds < 1) {
    return `${Math.round(seconds * 1000)}ms`
  }
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`
  }
  return formatDuration(seconds)
}

/**
 * Format events per second with appropriate precision
 */
export function formatEventsPerSecond(eventsPerSec: number): string {
  if (eventsPerSec === 0) return "0"
  if (eventsPerSec < 1) return eventsPerSec.toFixed(2)
  if (eventsPerSec < 10) return eventsPerSec.toFixed(1)
  return formatNumber(eventsPerSec)
}

/**
 * Format a timestamp for chart display
 */
export function formatChartTime(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
}

/**
 * Format a timestamp for table display
 */
export function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSecs = Math.floor(diffMs / 1000)

  if (diffSecs < 60) return "just now"
  if (diffSecs < 3600) return `${Math.floor(diffSecs / 60)}m ago`
  if (diffSecs < 86400) return `${Math.floor(diffSecs / 3600)}h ago`
  return `${Math.floor(diffSecs / 86400)}d ago`
}
