"use client"

import { use } from "react"
import Link from "next/link"
import { ArrowLeft, Pencil, Play, Pause, Trash2, XCircle } from "lucide-react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"
import {
  useScalingPolicy,
  useScalingState,
  useScalingHistory,
  useEnableScalingPolicy,
  useDisableScalingPolicy,
  useDeleteScalingPolicy,
} from "@/lib/hooks/use-scaling"
import { ScaleStateCard } from "@/components/scaling/scale-state-card"
import { ScalingHistoryTable } from "@/components/scaling/scaling-history-table"
import { TARGET_TYPE_LABELS } from "@/components/scaling/scaling-policy-card"
import type { RuleOperator } from "@/lib/api/types"

const OPERATOR_LABELS: Record<RuleOperator, string> = {
  gt: ">",
  gte: ">=",
  lt: "<",
  lte: "<=",
  eq: "=",
}

function PolicyDetailSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Skeleton className="h-10 w-10 rounded-lg" />
        <div className="space-y-2">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-32" />
        </div>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Skeleton className="h-40" />
        <Skeleton className="h-40" />
      </div>
      <Skeleton className="h-64" />
    </div>
  )
}

export default function ScalingPolicyDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const router = useRouter()

  const { data: policy, isLoading: policyLoading, error: policyError } = useScalingPolicy(id)
  const { data: state, isLoading: stateLoading } = useScalingState(id)
  const { data: history, isLoading: historyLoading } = useScalingHistory(id)

  const enablePolicy = useEnableScalingPolicy()
  const disablePolicy = useDisableScalingPolicy()
  const deletePolicy = useDeleteScalingPolicy()

  const handleEnable = () => {
    enablePolicy.mutate(id)
  }

  const handleDisable = () => {
    disablePolicy.mutate(id)
  }

  const handleDelete = () => {
    if (confirm("Are you sure you want to delete this policy?")) {
      deletePolicy.mutate(id, {
        onSuccess: () => {
          router.push("/scaling")
        },
      })
    }
  }

  if (policyLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/scaling">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
        </div>
        <PolicyDetailSkeleton />
      </div>
    )
  }

  if (policyError || !policy) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/scaling">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="py-8 text-center">
            <XCircle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-muted-foreground">
              Failed to load scaling policy. Please try again.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const isPending = enablePolicy.isPending || disablePolicy.isPending || deletePolicy.isPending

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/scaling">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-3xl font-bold">{policy.name}</h1>
              <Badge variant={policy.enabled ? "default" : "secondary"}>
                {policy.enabled ? "Active" : "Disabled"}
              </Badge>
            </div>
            <p className="text-muted-foreground">
              {TARGET_TYPE_LABELS[policy.target_type]}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href={`/scaling/${id}/edit`}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit
            </Link>
          </Button>
          {policy.enabled ? (
            <Button
              variant="outline"
              size="sm"
              onClick={handleDisable}
              disabled={isPending}
            >
              <Pause className="mr-2 h-4 w-4" />
              Disable
            </Button>
          ) : (
            <Button
              variant="outline"
              size="sm"
              onClick={handleEnable}
              disabled={isPending}
            >
              <Play className="mr-2 h-4 w-4" />
              Enable
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleDelete}
            disabled={isPending}
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Current state and config */}
      <div className="grid gap-4 md:grid-cols-2">
        <ScaleStateCard
          state={state}
          policy={policy}
          isLoading={stateLoading}
        />

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base font-medium">Configuration</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-2 text-sm">
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Min Replicas</dt>
                <dd className="font-medium">{policy.min_replicas}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Max Replicas</dt>
                <dd className="font-medium">{policy.max_replicas}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Cooldown</dt>
                <dd className="font-medium">{policy.cooldown_seconds}s</dd>
              </div>
              {policy.scale_to_zero && (
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Scale to Zero</dt>
                  <dd><Badge variant="outline" className="text-xs">Enabled</Badge></dd>
                </div>
              )}
              {policy.max_hourly_cost && (
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">Max Hourly Cost</dt>
                  <dd className="font-medium">${policy.max_hourly_cost.toFixed(2)}</dd>
                </div>
              )}
            </dl>
          </CardContent>
        </Card>
      </div>

      {/* Rules */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base font-medium">Scale Up Rules</CardTitle>
            <CardDescription>
              {policy.scale_up_rules.length} rule{policy.scale_up_rules.length !== 1 ? "s" : ""} configured
            </CardDescription>
          </CardHeader>
          <CardContent>
            {policy.scale_up_rules.length === 0 ? (
              <p className="text-sm text-muted-foreground">No scale up rules configured</p>
            ) : (
              <ul className="space-y-2">
                {policy.scale_up_rules.map((rule, index) => (
                  <li key={index} className="text-sm rounded-md bg-muted/50 p-2">
                    <span className="font-medium">{rule.metric}</span>{" "}
                    {OPERATOR_LABELS[rule.operator]} {rule.threshold} for {rule.duration_seconds}s{" "}
                    <span className="text-green-600">+{Math.abs(rule.scale_by)}</span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base font-medium">Scale Down Rules</CardTitle>
            <CardDescription>
              {policy.scale_down_rules.length} rule{policy.scale_down_rules.length !== 1 ? "s" : ""} configured
            </CardDescription>
          </CardHeader>
          <CardContent>
            {policy.scale_down_rules.length === 0 ? (
              <p className="text-sm text-muted-foreground">No scale down rules configured</p>
            ) : (
              <ul className="space-y-2">
                {policy.scale_down_rules.map((rule, index) => (
                  <li key={index} className="text-sm rounded-md bg-muted/50 p-2">
                    <span className="font-medium">{rule.metric}</span>{" "}
                    {OPERATOR_LABELS[rule.operator]} {rule.threshold} for {rule.duration_seconds}s{" "}
                    <span className="text-orange-600">{rule.scale_by}</span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Schedules */}
      {policy.schedules.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base font-medium">Schedules</CardTitle>
            <CardDescription>
              {policy.schedules.length} schedule{policy.schedules.length !== 1 ? "s" : ""} configured
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2">
              {policy.schedules.map((schedule, index) => (
                <li key={index} className="text-sm rounded-md bg-muted/50 p-2 flex items-center justify-between">
                  <div>
                    <span className="font-mono">{schedule.cron_expression}</span>{" "}
                    <span className="text-muted-foreground">({schedule.timezone})</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{schedule.desired_replicas} replicas</span>
                    {!schedule.enabled && (
                      <Badge variant="secondary" className="text-xs">Disabled</Badge>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      <Separator />

      {/* History */}
      <ScalingHistoryTable
        history={history ?? []}
        isLoading={historyLoading}
        showPolicyName={false}
      />
    </div>
  )
}
