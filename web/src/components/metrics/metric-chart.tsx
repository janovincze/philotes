"use client"

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { formatChartTime } from "@/lib/utils/format-metrics"

// Using a flexible type for chart data - recharts requires dynamic property access
// and TypeScript's index signature requirements prevent using stricter types here
interface MetricChartProps {
  title?: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: Array<{ timestamp: string } & Record<string, any>>
  dataKey: string
  color?: string
  unit?: string
  height?: number
  isLoading?: boolean
  formatValue?: (value: number) => string
  className?: string
}

export function MetricChart({
  title,
  data,
  dataKey,
  color = "hsl(var(--primary))",
  unit = "",
  height = 256,
  isLoading,
  formatValue,
  className,
}: MetricChartProps) {
  if (isLoading) {
    return (
      <Card className={className}>
        {title && (
          <CardHeader className="pb-2">
            <CardTitle className="text-base font-medium">{title}</CardTitle>
          </CardHeader>
        )}
        <CardContent className="pb-4">
          <Skeleton className="w-full" style={{ height }} />
        </CardContent>
      </Card>
    )
  }

  const hasData = data && data.length > 0

  return (
    <Card className={className}>
      {title && (
        <CardHeader className="pb-2">
          <CardTitle className="text-base font-medium">{title}</CardTitle>
        </CardHeader>
      )}
      <CardContent className="pb-4">
        {hasData ? (
          <ResponsiveContainer width="100%" height={height}>
            <LineChart
              data={data}
              margin={{ top: 5, right: 10, left: 10, bottom: 5 }}
            >
              <CartesianGrid
                strokeDasharray="3 3"
                className="stroke-muted"
                vertical={false}
              />
              <XAxis
                dataKey="timestamp"
                tickFormatter={formatChartTime}
                tick={{ fontSize: 12 }}
                tickLine={false}
                axisLine={false}
                className="text-muted-foreground"
              />
              <YAxis
                tick={{ fontSize: 12 }}
                tickLine={false}
                axisLine={false}
                tickFormatter={formatValue}
                className="text-muted-foreground"
                width={50}
              />
              <Tooltip
                content={({ active, payload, label }) => {
                  if (!active || !payload?.length || !label) return null
                  const value = payload[0].value as number
                  return (
                    <div className="rounded-lg border bg-background p-2 shadow-sm">
                      <p className="text-xs text-muted-foreground">
                        {formatChartTime(String(label))}
                      </p>
                      <p className="text-sm font-medium">
                        {formatValue ? formatValue(value) : value}
                        {unit && ` ${unit}`}
                      </p>
                    </div>
                  )
                }}
              />
              <Line
                type="monotone"
                dataKey={dataKey}
                stroke={color}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 0 }}
              />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div
            className="flex items-center justify-center text-muted-foreground"
            style={{ height }}
          >
            No data available
          </div>
        )}
      </CardContent>
    </Card>
  )
}

interface MultiSeriesChartProps {
  title?: string
  data: Array<{ timestamp: string; [key: string]: number | string }>
  series: Array<{
    dataKey: string
    color: string
    name: string
  }>
  height?: number
  isLoading?: boolean
  formatValue?: (value: number) => string
  className?: string
}

export function MultiSeriesChart({
  title,
  data,
  series,
  height = 256,
  isLoading,
  formatValue,
  className,
}: MultiSeriesChartProps) {
  if (isLoading) {
    return (
      <Card className={className}>
        {title && (
          <CardHeader className="pb-2">
            <CardTitle className="text-base font-medium">{title}</CardTitle>
          </CardHeader>
        )}
        <CardContent className="pb-4">
          <Skeleton className="w-full" style={{ height }} />
        </CardContent>
      </Card>
    )
  }

  const hasData = data && data.length > 0

  return (
    <Card className={className}>
      {title && (
        <CardHeader className="pb-2">
          <CardTitle className="text-base font-medium">{title}</CardTitle>
        </CardHeader>
      )}
      <CardContent className="pb-4">
        {hasData ? (
          <ResponsiveContainer width="100%" height={height}>
            <LineChart
              data={data}
              margin={{ top: 5, right: 10, left: 10, bottom: 5 }}
            >
              <CartesianGrid
                strokeDasharray="3 3"
                className="stroke-muted"
                vertical={false}
              />
              <XAxis
                dataKey="timestamp"
                tickFormatter={formatChartTime}
                tick={{ fontSize: 12 }}
                tickLine={false}
                axisLine={false}
                className="text-muted-foreground"
              />
              <YAxis
                tick={{ fontSize: 12 }}
                tickLine={false}
                axisLine={false}
                tickFormatter={formatValue}
                className="text-muted-foreground"
                width={50}
              />
              <Tooltip
                content={({ active, payload, label }) => {
                  if (!active || !payload?.length || !label) return null
                  return (
                    <div className="rounded-lg border bg-background p-2 shadow-sm">
                      <p className="text-xs text-muted-foreground mb-1">
                        {formatChartTime(String(label))}
                      </p>
                      {payload.map((entry) => (
                        <p
                          key={entry.dataKey}
                          className="text-sm"
                          style={{ color: entry.color }}
                        >
                          {entry.name}:{" "}
                          <span className="font-medium">
                            {formatValue
                              ? formatValue(entry.value as number)
                              : entry.value}
                          </span>
                        </p>
                      ))}
                    </div>
                  )
                }}
              />
              {series.map((s) => (
                <Line
                  key={s.dataKey}
                  type="monotone"
                  dataKey={s.dataKey}
                  stroke={s.color}
                  strokeWidth={2}
                  dot={false}
                  name={s.name}
                  activeDot={{ r: 4, strokeWidth: 0 }}
                />
              ))}
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div
            className="flex items-center justify-center text-muted-foreground"
            style={{ height }}
          >
            No data available
          </div>
        )}
      </CardContent>
    </Card>
  )
}
