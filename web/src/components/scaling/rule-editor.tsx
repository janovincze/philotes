"use client"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Card, CardContent } from "@/components/ui/card"
import { Plus, Trash2 } from "lucide-react"
import type { ScalingRule, RuleOperator } from "@/lib/api/types"

interface RuleEditorProps {
  rules: ScalingRule[]
  onChange: (rules: ScalingRule[]) => void
  type: "up" | "down"
}

const AVAILABLE_METRICS = [
  { value: "cdc_lag_seconds", label: "Replication Lag (seconds)" },
  { value: "buffer_depth", label: "Buffer Queue Depth" },
  { value: "cdc_events_rate", label: "Events per Second" },
  { value: "cdc_errors_total", label: "Total Errors" },
  { value: "cpu_percent", label: "CPU Usage (%)" },
  { value: "memory_percent", label: "Memory Usage (%)" },
]

const OPERATORS: { value: RuleOperator; label: string }[] = [
  { value: "gt", label: ">" },
  { value: "gte", label: ">=" },
  { value: "lt", label: "<" },
  { value: "lte", label: "<=" },
  { value: "eq", label: "=" },
]

const DEFAULT_RULE: ScalingRule = {
  metric: "cdc_lag_seconds",
  operator: "gt",
  threshold: 10,
  duration_seconds: 60,
  scale_by: 1,
}

export function RuleEditor({ rules, onChange, type }: RuleEditorProps) {
  const addRule = () => {
    const newRule = {
      ...DEFAULT_RULE,
      scale_by: type === "up" ? 1 : -1,
    }
    onChange([...rules, newRule])
  }

  const removeRule = (index: number) => {
    onChange(rules.filter((_, i) => i !== index))
  }

  const updateRule = (index: number, field: keyof ScalingRule, value: string | number) => {
    const updated = [...rules]
    updated[index] = { ...updated[index], [field]: value }
    onChange(updated)
  }

  return (
    <div className="space-y-3">
      {rules.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-4">
          No {type === "up" ? "scale up" : "scale down"} rules configured
        </p>
      )}

      {rules.map((rule, index) => (
        <Card key={index}>
          <CardContent className="pt-4">
            <div className="grid gap-4 sm:grid-cols-6">
              {/* Metric */}
              <div className="sm:col-span-2">
                <Label className="text-xs">Metric</Label>
                <Select
                  value={rule.metric}
                  onValueChange={(v) => updateRule(index, "metric", v)}
                >
                  <SelectTrigger className="mt-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {AVAILABLE_METRICS.map((m) => (
                      <SelectItem key={m.value} value={m.value}>
                        {m.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Operator */}
              <div>
                <Label className="text-xs">Operator</Label>
                <Select
                  value={rule.operator}
                  onValueChange={(v) => updateRule(index, "operator", v as RuleOperator)}
                >
                  <SelectTrigger className="mt-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {OPERATORS.map((op) => (
                      <SelectItem key={op.value} value={op.value}>
                        {op.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Threshold */}
              <div>
                <Label className="text-xs">Threshold</Label>
                <Input
                  type="number"
                  value={rule.threshold}
                  onChange={(e) => updateRule(index, "threshold", parseFloat(e.target.value) || 0)}
                  className="mt-1"
                />
              </div>

              {/* Duration */}
              <div>
                <Label className="text-xs">Duration (s)</Label>
                <Input
                  type="number"
                  value={rule.duration_seconds}
                  onChange={(e) => updateRule(index, "duration_seconds", parseInt(e.target.value) || 60)}
                  className="mt-1"
                  min={30}
                />
              </div>

              {/* Scale By + Delete */}
              <div className="flex items-end gap-2">
                <div className="flex-1">
                  <Label className="text-xs">Scale By</Label>
                  <Input
                    type="number"
                    value={Math.abs(rule.scale_by)}
                    onChange={(e) => {
                      const val = parseInt(e.target.value) || 1
                      updateRule(index, "scale_by", type === "up" ? Math.abs(val) : -Math.abs(val))
                    }}
                    className="mt-1"
                    min={1}
                  />
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => removeRule(index)}
                  className="shrink-0"
                >
                  <Trash2 className="h-4 w-4 text-destructive" />
                </Button>
              </div>
            </div>

            {/* Rule summary */}
            <p className="mt-3 text-xs text-muted-foreground">
              {type === "up" ? "Scale up" : "Scale down"} by {Math.abs(rule.scale_by)} replica
              {Math.abs(rule.scale_by) > 1 ? "s" : ""} when{" "}
              <span className="font-medium">
                {AVAILABLE_METRICS.find((m) => m.value === rule.metric)?.label}
              </span>{" "}
              is {OPERATORS.find((op) => op.value === rule.operator)?.label}{" "}
              <span className="font-medium">{rule.threshold}</span> for{" "}
              <span className="font-medium">{rule.duration_seconds}s</span>
            </p>
          </CardContent>
        </Card>
      ))}

      <Button type="button" variant="outline" size="sm" onClick={addRule} className="w-full">
        <Plus className="mr-2 h-4 w-4" />
        Add {type === "up" ? "Scale Up" : "Scale Down"} Rule
      </Button>
    </div>
  )
}
