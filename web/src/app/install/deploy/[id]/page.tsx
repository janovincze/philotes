"use client"

import { useEffect, useMemo } from "react"
import { useRouter, useParams } from "next/navigation"
import {
  CheckCircle,
  XCircle,
  Loader2,
  AlertCircle,
  ArrowRight,
  Copy,
  ExternalLink,
} from "lucide-react"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useDeployment, useDeploymentLogsStream, useCancelDeployment } from "@/lib/hooks/use-installer"
import type { DeploymentStatus } from "@/lib/api/types"

const DEPLOYMENT_STEPS = [
  { id: "pending", label: "Queued" },
  { id: "provisioning", label: "Provisioning Infrastructure" },
  { id: "configuring", label: "Configuring Cluster" },
  { id: "deploying", label: "Deploying Philotes" },
  { id: "verifying", label: "Health Verification" },
  { id: "completed", label: "Completed" },
]

function getStepIndex(status: DeploymentStatus): number {
  const index = DEPLOYMENT_STEPS.findIndex((s) => s.id === status)
  return index >= 0 ? index : 0
}

function StatusIcon({
  status,
  size = "default",
}: {
  status: DeploymentStatus
  size?: "default" | "large"
}) {
  const className = size === "large" ? "h-12 w-12" : "h-5 w-5"

  switch (status) {
    case "completed":
      return <CheckCircle className={`${className} text-green-500`} />
    case "failed":
      return <XCircle className={`${className} text-red-500`} />
    case "cancelled":
      return <AlertCircle className={`${className} text-yellow-500`} />
    default:
      return <Loader2 className={`${className} text-primary animate-spin`} />
  }
}

function DeploymentProgress({ status }: { status: DeploymentStatus }) {
  const currentIndex = getStepIndex(status)
  const isTerminal = ["completed", "failed", "cancelled"].includes(status)

  return (
    <div className="space-y-4">
      {DEPLOYMENT_STEPS.slice(0, -1).map((step, index) => {
        const isActive = index === currentIndex && !isTerminal
        const isCompleted = index < currentIndex || status === "completed"
        const isFailed = status === "failed" && index === currentIndex

        return (
          <div key={step.id} className="flex items-center gap-4">
            <div
              className={`flex h-8 w-8 items-center justify-center rounded-full ${
                isCompleted
                  ? "bg-green-500 text-white"
                  : isActive
                    ? "bg-primary text-primary-foreground"
                    : isFailed
                      ? "bg-red-500 text-white"
                      : "bg-muted text-muted-foreground"
              }`}
            >
              {isCompleted ? (
                <CheckCircle className="h-4 w-4" />
              ) : isActive ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : isFailed ? (
                <XCircle className="h-4 w-4" />
              ) : (
                index + 1
              )}
            </div>
            <span
              className={`font-medium ${
                isActive || isCompleted || isFailed
                  ? "text-foreground"
                  : "text-muted-foreground"
              }`}
            >
              {step.label}
            </span>
          </div>
        )
      })}
    </div>
  )
}

export default function DeploymentPage() {
  const router = useRouter()
  const params = useParams()
  const deploymentId = params.id as string

  const { data: deployment, isLoading: deploymentLoading } =
    useDeployment(deploymentId)
  const logsStream = useDeploymentLogsStream(deploymentId)
  const cancelDeployment = useCancelDeployment()

  const isActive = useMemo(() => {
    if (!deployment) return false
    return ["pending", "provisioning", "configuring", "deploying", "verifying"].includes(
      deployment.status
    )
  }, [deployment])

  // Connect to WebSocket when deployment is active
  useEffect(() => {
    if (isActive && !logsStream.connected) {
      logsStream.connect()
    }
    return () => {
      logsStream.disconnect()
    }
  }, [isActive, logsStream])

  const handleCancel = async () => {
    if (!deployment) return
    try {
      await cancelDeployment.mutateAsync(deployment.id)
    } catch (error) {
      console.error("Failed to cancel deployment:", error)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  if (deploymentLoading) {
    return (
      <div className="container max-w-4xl py-8 space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (!deployment) {
    return (
      <div className="container max-w-4xl py-8">
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive text-center">
              Deployment not found.
            </p>
            <Button
              variant="outline"
              className="mt-4 mx-auto block"
              onClick={() => router.push("/install")}
            >
              Back to Install
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  const isComplete = deployment.status === "completed"
  const isFailed = deployment.status === "failed"
  const isCancelled = deployment.status === "cancelled"
  const isTerminal = isComplete || isFailed || isCancelled

  return (
    <div className="container max-w-4xl py-8 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">{deployment.name}</h1>
          <p className="text-muted-foreground">
            Deploying to {deployment.provider} in {deployment.region}
          </p>
        </div>
        <Badge
          variant={
            isComplete ? "default" : isFailed ? "destructive" : "secondary"
          }
          className="text-lg px-4 py-2"
        >
          {deployment.status}
        </Badge>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Progress and Logs */}
        <div className="lg:col-span-2 space-y-6">
          {/* Progress Steps */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <StatusIcon status={deployment.status} />
                Deployment Progress
              </CardTitle>
            </CardHeader>
            <CardContent>
              <DeploymentProgress status={deployment.status} />
            </CardContent>
          </Card>

          {/* Deployment Logs */}
          <Card>
            <CardHeader>
              <CardTitle>Deployment Logs</CardTitle>
              <CardDescription>
                {logsStream.connected
                  ? "Connected to real-time log stream"
                  : "Logs from deployment process"}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ScrollArea className="h-[300px] rounded-md border p-4 bg-muted/50 font-mono text-sm">
                {logsStream.logs.length > 0 ? (
                  logsStream.logs.map((log, index) => (
                    <div
                      key={index}
                      className={`py-1 ${
                        log.level === "error"
                          ? "text-red-500"
                          : log.level === "warn"
                            ? "text-yellow-500"
                            : "text-foreground"
                      }`}
                    >
                      <span className="text-muted-foreground">
                        [{new Date(log.timestamp).toLocaleTimeString()}]
                      </span>{" "}
                      {log.step && <span className="text-primary">[{log.step}]</span>}{" "}
                      {log.message}
                    </div>
                  ))
                ) : (
                  <div className="text-muted-foreground">
                    Waiting for logs...
                  </div>
                )}
              </ScrollArea>
            </CardContent>
          </Card>

          {/* Error Message */}
          {isFailed && deployment.error_message && (
            <Card className="border-destructive">
              <CardHeader>
                <CardTitle className="text-destructive">
                  Deployment Failed
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-destructive">{deployment.error_message}</p>
              </CardContent>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Actions */}
          <Card>
            <CardHeader>
              <CardTitle>Actions</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {isActive && (
                <Button
                  variant="destructive"
                  className="w-full"
                  onClick={handleCancel}
                  disabled={cancelDeployment.isPending}
                >
                  {cancelDeployment.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Cancelling...
                    </>
                  ) : (
                    "Cancel Deployment"
                  )}
                </Button>
              )}

              {isComplete && (
                <>
                  <Button
                    className="w-full"
                    onClick={() => router.push("/")}
                  >
                    <ArrowRight className="mr-2 h-4 w-4" />
                    Go to Dashboard
                  </Button>
                  {deployment.outputs?.dashboard_url && (
                    <Button
                      variant="outline"
                      className="w-full"
                      asChild
                    >
                      <a
                        href={deployment.outputs.dashboard_url}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <ExternalLink className="mr-2 h-4 w-4" />
                        Open Philotes
                      </a>
                    </Button>
                  )}
                </>
              )}

              {isTerminal && (
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => router.push("/install")}
                >
                  New Deployment
                </Button>
              )}
            </CardContent>
          </Card>

          {/* Deployment Details */}
          <Card>
            <CardHeader>
              <CardTitle>Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Provider</span>
                <span className="capitalize">{deployment.provider}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Region</span>
                <span>{deployment.region}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Size</span>
                <span className="capitalize">{deployment.size}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Environment</span>
                <span className="capitalize">{deployment.environment}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <span>
                  {new Date(deployment.created_at).toLocaleDateString()}
                </span>
              </div>
            </CardContent>
          </Card>

          {/* Outputs (when completed) */}
          {isComplete && deployment.outputs && (
            <Card>
              <CardHeader>
                <CardTitle>Connection Info</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {deployment.outputs.control_plane_ip && (
                  <div className="space-y-1">
                    <Label className="text-muted-foreground">
                      Control Plane IP
                    </Label>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 text-sm bg-muted p-2 rounded">
                        {deployment.outputs.control_plane_ip}
                      </code>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() =>
                          copyToClipboard(deployment.outputs!.control_plane_ip!)
                        }
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}

                {deployment.outputs.load_balancer_ip && (
                  <div className="space-y-1">
                    <Label className="text-muted-foreground">
                      Load Balancer IP
                    </Label>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 text-sm bg-muted p-2 rounded">
                        {deployment.outputs.load_balancer_ip}
                      </code>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() =>
                          copyToClipboard(deployment.outputs!.load_balancer_ip!)
                        }
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}

                {deployment.outputs.dashboard_url && (
                  <div className="space-y-1">
                    <Label className="text-muted-foreground">Dashboard URL</Label>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 text-sm bg-muted p-2 rounded truncate">
                        {deployment.outputs.dashboard_url}
                      </code>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() =>
                          copyToClipboard(deployment.outputs!.dashboard_url!)
                        }
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}

// Placeholder for Label component (should be imported from UI)
function Label({
  className,
  children,
}: {
  className?: string
  children: React.ReactNode
}) {
  return (
    <label className={`text-sm font-medium ${className || ""}`}>{children}</label>
  )
}
