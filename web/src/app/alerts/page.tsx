"use client"

import { useState } from "react"
import {
  Bell,
  AlertTriangle,
  AlertCircle,
  Info,
  CheckCircle,
  Clock,
  Plus,
  ToggleLeft,
  ToggleRight,
  Trash2,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import {
  useAlertSummary,
  useAlertRules,
  useAlerts,
  useUpdateAlertRule,
  useDeleteAlertRule,
  useAcknowledgeAlert,
} from "@/lib/hooks/use-alerts"
import type {
  AlertRule,
  AlertInstance,
  AlertSeverity,
  AlertStatus,
} from "@/lib/api/alerts"

function SeverityBadge({ severity }: { severity: AlertSeverity }) {
  const config = {
    critical: {
      icon: AlertTriangle,
      variant: "destructive" as const,
      className: "",
    },
    warning: {
      icon: AlertCircle,
      variant: "default" as const,
      className: "bg-yellow-500 hover:bg-yellow-600",
    },
    info: {
      icon: Info,
      variant: "secondary" as const,
      className: "",
    },
  }

  const { icon: Icon, variant, className } = config[severity]

  return (
    <Badge variant={variant} className={`gap-1 ${className}`}>
      <Icon className="h-3 w-3" />
      <span className="capitalize">{severity}</span>
    </Badge>
  )
}

function StatusBadge({ status }: { status: AlertStatus }) {
  const config = {
    firing: {
      icon: AlertTriangle,
      variant: "destructive" as const,
      className: "",
    },
    pending: {
      icon: Clock,
      variant: "default" as const,
      className: "bg-yellow-500 hover:bg-yellow-600",
    },
    resolved: {
      icon: CheckCircle,
      variant: "secondary" as const,
      className: "",
    },
  }

  const { icon: Icon, variant, className } = config[status]

  return (
    <Badge variant={variant} className={`gap-1 ${className}`}>
      <Icon className="h-3 w-3" />
      <span className="capitalize">{status}</span>
    </Badge>
  )
}

function SummaryCards() {
  const { data: summary, isLoading } = useAlertSummary()

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <Card key={i}>
            <CardHeader className="pb-2">
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-12" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  const cards = [
    {
      label: "Firing Alerts",
      value: summary?.firing_alerts ?? 0,
      icon: AlertTriangle,
      color: "text-red-500",
    },
    {
      label: "Active Rules",
      value: summary?.enabled_rules ?? 0,
      icon: Bell,
      color: "text-primary",
    },
    {
      label: "Total Rules",
      value: summary?.total_rules ?? 0,
      icon: ToggleRight,
      color: "text-muted-foreground",
    },
    {
      label: "Active Silences",
      value: summary?.active_silences ?? 0,
      icon: Clock,
      color: "text-yellow-500",
    },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-4">
      {cards.map(({ label, value, icon: Icon, color }) => (
        <Card key={label}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">{label}</CardTitle>
            <Icon className={`h-4 w-4 ${color}`} />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function AlertRulesTable() {
  const { data, isLoading, error } = useAlertRules()
  const updateRule = useUpdateAlertRule()
  const deleteRule = useDeleteAlertRule()

  const handleToggleEnabled = (rule: AlertRule) => {
    updateRule.mutate({
      id: rule.id,
      input: { enabled: !rule.enabled },
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-16 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="py-8 text-center text-muted-foreground">
        Failed to load alert rules
      </div>
    )
  }

  const rules = data?.rules ?? []

  if (rules.length === 0) {
    return (
      <div className="py-12 text-center">
        <Bell className="mx-auto h-12 w-12 text-muted-foreground" />
        <h3 className="mt-4 text-lg font-medium">No alert rules</h3>
        <p className="mt-2 text-muted-foreground">
          Create your first alert rule to start monitoring
        </p>
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Metric</TableHead>
          <TableHead>Condition</TableHead>
          <TableHead>Severity</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rules.map((rule) => (
          <TableRow key={rule.id}>
            <TableCell>
              <div>
                <div className="font-medium">{rule.name}</div>
                {rule.description && (
                  <div className="text-sm text-muted-foreground">
                    {rule.description}
                  </div>
                )}
              </div>
            </TableCell>
            <TableCell className="font-mono text-sm">
              {rule.metric_name}
            </TableCell>
            <TableCell className="font-mono text-sm">
              {rule.operator} {rule.threshold}
            </TableCell>
            <TableCell>
              <SeverityBadge severity={rule.severity} />
            </TableCell>
            <TableCell>
              <Badge variant={rule.enabled ? "default" : "secondary"}>
                {rule.enabled ? "Enabled" : "Disabled"}
              </Badge>
            </TableCell>
            <TableCell className="text-right">
              <div className="flex justify-end gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleToggleEnabled(rule)}
                  disabled={updateRule.isPending}
                >
                  {rule.enabled ? (
                    <ToggleRight className="h-4 w-4" />
                  ) : (
                    <ToggleLeft className="h-4 w-4" />
                  )}
                </Button>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant="ghost" size="sm">
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Delete alert rule?</AlertDialogTitle>
                      <AlertDialogDescription>
                        This will permanently delete the alert rule &quot;{rule.name}&quot;.
                        Any active alerts from this rule will be resolved.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction
                        onClick={() => deleteRule.mutate(rule.id)}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                      >
                        Delete
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function ActiveAlertsTable() {
  const [statusFilter, setStatusFilter] = useState<string | undefined>("firing")
  const { data, isLoading, error } = useAlerts(statusFilter)
  const acknowledgeAlert = useAcknowledgeAlert()

  const handleAcknowledge = (alert: AlertInstance) => {
    acknowledgeAlert.mutate({
      id: alert.id,
      acknowledgedBy: "admin", // TODO: Get from auth context
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-16 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="py-8 text-center text-muted-foreground">
        Failed to load alerts
      </div>
    )
  }

  const alerts = data?.alerts ?? []

  if (alerts.length === 0) {
    return (
      <div className="py-12 text-center">
        <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
        <h3 className="mt-4 text-lg font-medium">No active alerts</h3>
        <p className="mt-2 text-muted-foreground">
          All systems are operating normally
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Button
          variant={statusFilter === "firing" ? "default" : "outline"}
          size="sm"
          onClick={() => setStatusFilter("firing")}
        >
          Firing
        </Button>
        <Button
          variant={statusFilter === "pending" ? "default" : "outline"}
          size="sm"
          onClick={() => setStatusFilter("pending")}
        >
          Pending
        </Button>
        <Button
          variant={statusFilter === "resolved" ? "default" : "outline"}
          size="sm"
          onClick={() => setStatusFilter("resolved")}
        >
          Resolved
        </Button>
        <Button
          variant={!statusFilter ? "default" : "outline"}
          size="sm"
          onClick={() => setStatusFilter(undefined)}
        >
          All
        </Button>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Alert</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Value</TableHead>
            <TableHead>Fired At</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {alerts.map((alert) => (
            <TableRow key={alert.id}>
              <TableCell>
                <div>
                  <div className="font-medium">
                    {alert.rule?.name ?? "Unknown Rule"}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {alert.fingerprint.substring(0, 16)}...
                  </div>
                </div>
              </TableCell>
              <TableCell>
                <StatusBadge status={alert.status} />
              </TableCell>
              <TableCell className="font-mono">
                {alert.current_value?.toFixed(2) ?? "-"}
              </TableCell>
              <TableCell>
                {new Date(alert.fired_at).toLocaleString()}
              </TableCell>
              <TableCell className="text-right">
                {alert.status === "firing" && !alert.acknowledged_at && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleAcknowledge(alert)}
                    disabled={acknowledgeAlert.isPending}
                  >
                    Acknowledge
                  </Button>
                )}
                {alert.acknowledged_at && (
                  <span className="text-sm text-muted-foreground">
                    Acked by {alert.acknowledged_by}
                  </span>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
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
            Monitor and manage alert rules and active alerts
          </p>
        </div>
        <Button disabled>
          <Plus className="mr-2 h-4 w-4" />
          Create Rule
        </Button>
      </div>

      {/* Summary cards */}
      <SummaryCards />

      {/* Tabs for alerts and rules */}
      <Tabs defaultValue="alerts" className="space-y-4">
        <TabsList>
          <TabsTrigger value="alerts">Active Alerts</TabsTrigger>
          <TabsTrigger value="rules">Alert Rules</TabsTrigger>
        </TabsList>

        <TabsContent value="alerts">
          <Card>
            <CardHeader>
              <CardTitle>Active Alerts</CardTitle>
              <CardDescription>
                Current alerts and their status
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ActiveAlertsTable />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="rules">
          <Card>
            <CardHeader>
              <CardTitle>Alert Rules</CardTitle>
              <CardDescription>
                Configured alert rules for monitoring
              </CardDescription>
            </CardHeader>
            <CardContent>
              <AlertRulesTable />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
