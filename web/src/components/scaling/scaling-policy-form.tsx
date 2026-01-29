"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { RuleEditor } from "./rule-editor"
import { ScheduleEditor } from "./schedule-editor"
import type {
  ScalingPolicy,
  CreateScalingPolicyInput,
  ScalingTargetType,
} from "@/lib/api/types"

// Validation schema
const ruleSchema = z.object({
  metric: z.string().min(1),
  operator: z.enum(["gt", "lt", "gte", "lte", "eq"]),
  threshold: z.number(),
  duration_seconds: z.number().int().min(30),
  scale_by: z.number().int(),
})

const scheduleSchema = z.object({
  cron_expression: z.string().min(1),
  desired_replicas: z.number().int().min(0),
  timezone: z.string().min(1),
  enabled: z.boolean(),
})

const formSchema = z
  .object({
    name: z.string().min(1, "Name is required").max(100),
    target_type: z.enum(["cdc-worker", "trino", "risingwave", "nodes"]),
    target_id: z.string().optional(),
    min_replicas: z.number().int().min(0).max(100),
    max_replicas: z.number().int().min(1).max(100),
    cooldown_seconds: z.number().int().min(60).max(3600),
    max_hourly_cost: z.number().min(0).optional(),
    scale_to_zero: z.boolean(),
    enabled: z.boolean(),
    scale_up_rules: z.array(ruleSchema),
    scale_down_rules: z.array(ruleSchema),
    schedules: z.array(scheduleSchema),
  })
  .refine((data) => data.max_replicas >= data.min_replicas, {
    message: "Max replicas must be >= min replicas",
    path: ["max_replicas"],
  })

type FormValues = z.infer<typeof formSchema>

interface ScalingPolicyFormProps {
  policy?: ScalingPolicy
  onSubmit: (data: CreateScalingPolicyInput) => void
  isSubmitting?: boolean
}

const TARGET_TYPE_OPTIONS: { value: ScalingTargetType; label: string; description: string }[] = [
  {
    value: "cdc-worker",
    label: "CDC Worker",
    description: "Scale CDC pipeline workers",
  },
  {
    value: "trino",
    label: "Trino",
    description: "Scale Trino query workers",
  },
  {
    value: "risingwave",
    label: "RisingWave",
    description: "Scale RisingWave compute nodes",
  },
  {
    value: "nodes",
    label: "Infrastructure Nodes",
    description: "Scale underlying infrastructure",
  },
]

export function ScalingPolicyForm({
  policy,
  onSubmit,
  isSubmitting,
}: ScalingPolicyFormProps) {
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: policy?.name ?? "",
      target_type: policy?.target_type ?? "cdc-worker",
      target_id: policy?.target_id ?? "",
      min_replicas: policy?.min_replicas ?? 1,
      max_replicas: policy?.max_replicas ?? 5,
      cooldown_seconds: policy?.cooldown_seconds ?? 300,
      max_hourly_cost: policy?.max_hourly_cost ?? undefined,
      scale_to_zero: policy?.scale_to_zero ?? false,
      enabled: policy?.enabled ?? true,
      scale_up_rules: policy?.scale_up_rules ?? [],
      scale_down_rules: policy?.scale_down_rules ?? [],
      schedules: policy?.schedules ?? [],
    },
  })

  const handleSubmit = (data: FormValues) => {
    onSubmit({
      ...data,
      target_id: data.target_id || undefined,
      max_hourly_cost: data.max_hourly_cost || undefined,
    })
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
        {/* Basic Info */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Basic Information</CardTitle>
            <CardDescription>Configure the scaling policy name and target</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Policy Name</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g., CDC Worker Auto-scale" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="target_type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Target Type</FormLabel>
                  <Select onValueChange={field.onChange} defaultValue={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder="Select target type" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {TARGET_TYPE_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          <div>
                            <div className="font-medium">{option.label}</div>
                            <div className="text-xs text-muted-foreground">
                              {option.description}
                            </div>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="flex items-center gap-4">
              <FormField
                control={form.control}
                name="enabled"
                render={({ field }) => (
                  <FormItem className="flex items-center gap-2">
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                    <FormLabel className="!mt-0">Policy Enabled</FormLabel>
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
        </Card>

        {/* Scale Limits */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Scale Limits</CardTitle>
            <CardDescription>Set the minimum and maximum replica counts</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <FormField
                control={form.control}
                name="min_replicas"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Min Replicas</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...field}
                        onChange={(e) => field.onChange(parseInt(e.target.value) || 0)}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="max_replicas"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Max Replicas</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...field}
                        onChange={(e) => field.onChange(parseInt(e.target.value) || 1)}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="cooldown_seconds"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Cooldown (seconds)</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        {...field}
                        onChange={(e) => field.onChange(parseInt(e.target.value) || 300)}
                      />
                    </FormControl>
                    <FormDescription className="text-xs">
                      Wait time between scaling actions
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <Separator />

            <div className="grid gap-4 sm:grid-cols-2">
              <FormField
                control={form.control}
                name="scale_to_zero"
                render={({ field }) => (
                  <FormItem className="flex items-center gap-2">
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                    <div>
                      <FormLabel className="!mt-0">Scale to Zero</FormLabel>
                      <FormDescription className="text-xs">
                        Allow scaling down to 0 replicas when idle
                      </FormDescription>
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="max_hourly_cost"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Max Hourly Cost (optional)</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        step="0.01"
                        placeholder="No limit"
                        value={field.value ?? ""}
                        onChange={(e) =>
                          field.onChange(e.target.value ? parseFloat(e.target.value) : undefined)
                        }
                      />
                    </FormControl>
                    <FormDescription className="text-xs">
                      Cost limit in your currency per hour
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
        </Card>

        {/* Scale Up Rules */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Scale Up Rules</CardTitle>
            <CardDescription>
              Define conditions that trigger scaling up (adding replicas)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="scale_up_rules"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <RuleEditor
                      rules={field.value}
                      onChange={field.onChange}
                      type="up"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        {/* Scale Down Rules */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Scale Down Rules</CardTitle>
            <CardDescription>
              Define conditions that trigger scaling down (removing replicas)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="scale_down_rules"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <RuleEditor
                      rules={field.value}
                      onChange={field.onChange}
                      type="down"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        {/* Schedules */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Schedules</CardTitle>
            <CardDescription>
              Set time-based scaling schedules (e.g., scale up during business hours)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <FormField
              control={form.control}
              name="schedules"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <ScheduleEditor schedules={field.value} onChange={field.onChange} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        {/* Submit */}
        <div className="flex justify-end gap-4">
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Saving..." : policy ? "Update Policy" : "Create Policy"}
          </Button>
        </div>
      </form>
    </Form>
  )
}
