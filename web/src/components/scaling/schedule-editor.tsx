"use client"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Card, CardContent } from "@/components/ui/card"
import { Plus, Trash2, Clock } from "lucide-react"
import type { ScalingSchedule } from "@/lib/api/types"

interface ScheduleEditorProps {
  schedules: ScalingSchedule[]
  onChange: (schedules: ScalingSchedule[]) => void
}

const COMMON_TIMEZONES = [
  { value: "UTC", label: "UTC" },
  { value: "Europe/London", label: "Europe/London" },
  { value: "Europe/Paris", label: "Europe/Paris (CET)" },
  { value: "Europe/Budapest", label: "Europe/Budapest (CET)" },
  { value: "America/New_York", label: "America/New York (EST)" },
  { value: "America/Los_Angeles", label: "America/Los Angeles (PST)" },
  { value: "Asia/Tokyo", label: "Asia/Tokyo (JST)" },
]

const CRON_PRESETS = [
  { value: "0 8 * * 1-5", label: "Weekdays at 8am" },
  { value: "0 18 * * 1-5", label: "Weekdays at 6pm" },
  { value: "0 0 * * *", label: "Daily at midnight" },
  { value: "0 * * * *", label: "Every hour" },
]

const DEFAULT_SCHEDULE: ScalingSchedule = {
  cron_expression: "0 8 * * 1-5",
  desired_replicas: 2,
  timezone: "UTC",
  enabled: true,
}

export function ScheduleEditor({ schedules, onChange }: ScheduleEditorProps) {
  const addSchedule = () => {
    onChange([...schedules, { ...DEFAULT_SCHEDULE }])
  }

  const removeSchedule = (index: number) => {
    onChange(schedules.filter((_, i) => i !== index))
  }

  const updateSchedule = (
    index: number,
    field: keyof ScalingSchedule,
    value: string | number | boolean
  ) => {
    const updated = [...schedules]
    updated[index] = { ...updated[index], [field]: value }
    onChange(updated)
  }

  return (
    <div className="space-y-3">
      {schedules.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-4">
          No schedules configured
        </p>
      )}

      {schedules.map((schedule, index) => (
        <Card key={index}>
          <CardContent className="pt-4">
            <div className="grid gap-4 sm:grid-cols-5">
              {/* Cron expression */}
              <div className="sm:col-span-2">
                <Label className="text-xs">Cron Expression</Label>
                <div className="flex gap-2 mt-1">
                  <Input
                    value={schedule.cron_expression}
                    onChange={(e) => updateSchedule(index, "cron_expression", e.target.value)}
                    placeholder="0 8 * * 1-5"
                    className="font-mono"
                  />
                  <Select
                    value=""
                    onValueChange={(v) => updateSchedule(index, "cron_expression", v)}
                  >
                    <SelectTrigger className="w-[140px]">
                      <SelectValue placeholder="Presets" />
                    </SelectTrigger>
                    <SelectContent>
                      {CRON_PRESETS.map((preset) => (
                        <SelectItem key={preset.value} value={preset.value}>
                          {preset.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Replicas */}
              <div>
                <Label className="text-xs">Replicas</Label>
                <Input
                  type="number"
                  value={schedule.desired_replicas}
                  onChange={(e) =>
                    updateSchedule(index, "desired_replicas", parseInt(e.target.value) || 1)
                  }
                  className="mt-1"
                  min={0}
                />
              </div>

              {/* Timezone */}
              <div>
                <Label className="text-xs">Timezone</Label>
                <Select
                  value={schedule.timezone}
                  onValueChange={(v) => updateSchedule(index, "timezone", v)}
                >
                  <SelectTrigger className="mt-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {COMMON_TIMEZONES.map((tz) => (
                      <SelectItem key={tz.value} value={tz.value}>
                        {tz.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Enabled + Delete */}
              <div className="flex items-end gap-2">
                <div className="flex items-center gap-2">
                  <Switch
                    checked={schedule.enabled}
                    onCheckedChange={(v) => updateSchedule(index, "enabled", v)}
                  />
                  <Label className="text-xs">Enabled</Label>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => removeSchedule(index)}
                  className="shrink-0 ml-auto"
                >
                  <Trash2 className="h-4 w-4 text-destructive" />
                </Button>
              </div>
            </div>

            {/* Schedule summary */}
            <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              <span>
                Scale to <span className="font-medium">{schedule.desired_replicas}</span> replica
                {schedule.desired_replicas !== 1 ? "s" : ""} at{" "}
                <span className="font-mono">{schedule.cron_expression}</span> ({schedule.timezone})
              </span>
            </div>
          </CardContent>
        </Card>
      ))}

      <Button type="button" variant="outline" size="sm" onClick={addSchedule} className="w-full">
        <Plus className="mr-2 h-4 w-4" />
        Add Schedule
      </Button>

      {schedules.length > 0 && (
        <p className="text-xs text-muted-foreground">
          Cron format: minute hour day-of-month month day-of-week (e.g., &quot;0 8 * * 1-5&quot; = weekdays at 8am)
        </p>
      )}
    </div>
  )
}
