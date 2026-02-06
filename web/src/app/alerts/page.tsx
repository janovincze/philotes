"use client"

import { useState } from "react"
import {
  Bell,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Shield,
  Plus,
  Trash2,
  Send,
  Loader2,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  useAlerts,
  useAlertRules,
  useAlertSummary,
  useAcknowledgeAlert,
  useDeleteAlertRule,
  useNotificationChannels,
  useDeleteChannel,
  useTestChannel,
} from "@/lib/hooks/use-alerts"
import type { AlertInstance, AlertRule, AlertSeverity, AlertStatus, NotificationChannel } from "@/lib/api/types"

function SeverityBadge({ severity }: { severity: AlertSeverity }) {
  const config = {
    critical: { variant: "destructive" as const, icon: XCircle },
    warning: { variant: "default" as const, icon: AlertTriangle },
    info: { variant: "secondary" as const, icon: Bell },
  }
  const { variant, icon: Icon } = config[severity] || config.info
  return (
    <Badge variant={variant} className="gap-1">
      <Icon className="h-3 w-3" />
      <span className="capitalize">{severity}</span>
    </Badge>
  )
}

function StatusBadge({ status }: { status: AlertStatus }) {
  const config = {
    firing: { variant: "destructive" as const, color: "text-red-500" },
    acknowledged: { variant: "default" as const, color: "text-yellow-500" },
    resolved: { variant: "secondary" as const, color: "text-green-500" },
  }
  const { variant } = config[status] || config.firing
  return (
    <Badge variant={variant}>
      <span className="capitalize">{status}</span>
    </Badge>
  )
}

function SummaryCards() {
  const { data: summaryResp, isLoading } = useAlertSummary()

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <Card key={i}>
            <CardHeader className="pb-2">
              <Skeleton className="h-4 w-20" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-12" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  const summary = summaryResp?.summary
  return (
    <div className="grid gap-4 md:grid-cols-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Firing</CardTitle>
          <AlertTriangle className="h-4 w-4 text-destructive" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{summary?.firing_alerts ?? 0}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Acknowledged</CardTitle>
          <CheckCircle className="h-4 w-4 text-yellow-500" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{summary?.acknowledged_alerts ?? 0}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Total Rules</CardTitle>
          <Shield className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{summary?.total_rules ?? 0}</div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Active Rules</CardTitle>
          <Bell className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{summary?.enabled_rules ?? 0}</div>
        </CardContent>
      </Card>
    </div>
  )
}

function ActiveAlertsList() {
  const { data, isLoading, error } = useAlerts()
  const acknowledgeAlert = useAcknowledgeAlert()

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <Card key={i}>
            <CardContent className="py-4">
              <Skeleton className="h-16 w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <XCircle className="mx-auto h-8 w-8 text-destructive" />
          <p className="mt-2 text-muted-foreground">Failed to load alerts.</p>
        </CardContent>
      </Card>
    )
  }

  const alerts = data?.alerts || []

  if (alerts.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <CheckCircle className="mx-auto h-8 w-8 text-green-500" />
          <h3 className="mt-4 text-lg font-medium">All clear</h3>
          <p className="mt-2 text-muted-foreground">No active alerts at this time.</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-3">
      {alerts.map((alert: AlertInstance) => (
        <Card key={alert.id}>
          <CardContent className="flex items-center justify-between py-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <StatusBadge status={alert.status} />
                {alert.rule && <SeverityBadge severity={alert.rule.severity} />}
                <span className="font-medium">{alert.rule?.name || alert.fingerprint}</span>
              </div>
              <p className="text-sm text-muted-foreground">
                Fired {new Date(alert.fired_at).toLocaleString()}
                {alert.current_value != null && ` — value: ${alert.current_value}`}
              </p>
            </div>
            {alert.status === "firing" && (
              <Button
                variant="outline"
                size="sm"
                disabled={acknowledgeAlert.isPending}
                onClick={() =>
                  acknowledgeAlert.mutate({
                    id: alert.id,
                    acknowledgedBy: "admin",
                  })
                }
              >
                Acknowledge
              </Button>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function AlertRulesList() {
  const { data, isLoading, error } = useAlertRules()
  const deleteRule = useDeleteAlertRule()

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <Card key={i}>
            <CardContent className="py-4">
              <Skeleton className="h-12 w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <XCircle className="mx-auto h-8 w-8 text-destructive" />
          <p className="mt-2 text-muted-foreground">Failed to load alert rules.</p>
        </CardContent>
      </Card>
    )
  }

  const rules = data?.rules || []

  if (rules.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <Shield className="mx-auto h-8 w-8 text-muted-foreground" />
          <h3 className="mt-4 text-lg font-medium">No alert rules</h3>
          <p className="mt-2 text-muted-foreground">
            Create alert rules to monitor your CDC pipelines.
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-3">
      {rules.map((rule: AlertRule) => (
        <Card key={rule.id}>
          <CardContent className="flex items-center justify-between py-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <SeverityBadge severity={rule.severity} />
                <span className="font-medium">{rule.name}</span>
                {!rule.enabled && (
                  <Badge variant="outline" className="text-muted-foreground">
                    Disabled
                  </Badge>
                )}
              </div>
              <p className="text-sm text-muted-foreground">
                {rule.metric_name} {rule.operator} {rule.threshold}
                {rule.duration_seconds > 0 && ` for ${rule.duration_seconds}s`}
              </p>
              {rule.description && (
                <p className="text-sm text-muted-foreground">{rule.description}</p>
              )}
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="text-muted-foreground hover:text-destructive"
              disabled={deleteRule.isPending}
              onClick={() => {
                if (confirm(`Delete rule "${rule.name}"?`)) {
                  deleteRule.mutate(rule.id)
                }
              }}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function ChannelsList() {
  const { data, isLoading, error } = useNotificationChannels()
  const deleteChannel = useDeleteChannel()
  const testChannel = useTestChannel()
  const [testingId, setTestingId] = useState<string | null>(null)

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2].map((i) => (
          <Card key={i}>
            <CardContent className="py-4">
              <Skeleton className="h-12 w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <XCircle className="mx-auto h-8 w-8 text-destructive" />
          <p className="mt-2 text-muted-foreground">Failed to load notification channels.</p>
        </CardContent>
      </Card>
    )
  }

  const channels = data?.channels || []

  if (channels.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center">
          <Send className="mx-auto h-8 w-8 text-muted-foreground" />
          <h3 className="mt-4 text-lg font-medium">No notification channels</h3>
          <p className="mt-2 text-muted-foreground">
            Add notification channels (Slack, Email, Webhook) to receive alerts.
          </p>
        </CardContent>
      </Card>
    )
  }

  const channelTypeIcons: Record<string, string> = {
    slack: "#",
    email: "@",
    webhook: "{}",
    pagerduty: "PD",
  }

  return (
    <div className="space-y-3">
      {channels.map((channel: NotificationChannel) => (
        <Card key={channel.id}>
          <CardContent className="flex items-center justify-between py-4">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <Badge variant="outline">{channelTypeIcons[channel.type] || channel.type}</Badge>
                <span className="font-medium">{channel.name}</span>
                <Badge variant={channel.enabled ? "default" : "secondary"}>
                  {channel.enabled ? "Enabled" : "Disabled"}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground capitalize">{channel.type}</p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={testChannel.isPending && testingId === channel.id}
                onClick={() => {
                  setTestingId(channel.id)
                  testChannel.mutate(channel.id, {
                    onSettled: () => setTestingId(null),
                  })
                }}
              >
                {testChannel.isPending && testingId === channel.id ? (
                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                ) : (
                  <Send className="mr-1 h-3 w-3" />
                )}
                Test
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="text-muted-foreground hover:text-destructive"
                disabled={deleteChannel.isPending}
                onClick={() => {
                  if (confirm(`Delete channel "${channel.name}"?`)) {
                    deleteChannel.mutate(channel.id)
                  }
                }}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export default function AlertsPage() {
  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Alerts</h1>
          <p className="text-muted-foreground">
            Monitor active alerts, manage rules, and configure notification channels
          </p>
        </div>
      </div>

      {/* Summary cards */}
      <SummaryCards />

      {/* Tabs */}
      <Tabs defaultValue="active">
        <TabsList>
          <TabsTrigger value="active">Active Alerts</TabsTrigger>
          <TabsTrigger value="rules">Rules</TabsTrigger>
          <TabsTrigger value="channels">Channels</TabsTrigger>
        </TabsList>

        <TabsContent value="active" className="mt-4">
          <ActiveAlertsList />
        </TabsContent>

        <TabsContent value="rules" className="mt-4">
          <AlertRulesList />
        </TabsContent>

        <TabsContent value="channels" className="mt-4">
          <ChannelsList />
        </TabsContent>
      </Tabs>
    </div>
  )
}
