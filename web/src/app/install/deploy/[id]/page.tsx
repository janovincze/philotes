"use client"

import { useEffect, useMemo, useState } from "react"
import { useRouter, useParams } from "next/navigation"
import {
  CheckCircle,
  XCircle,
  Loader2,
  AlertCircle,
  ArrowRight,
  Copy,
  ExternalLink,
  RefreshCw,
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
import { Label } from "@/components/ui/label"
import {
  useDeployment,
  useDeploymentLogsStream,
  useCancelDeployment,
  useDeploymentProgress,
  useRetryDeployment,
  useRetryInfo,
} from "@/lib/hooks/use-installer"
import type { DeploymentStatus, DeploymentStep } from "@/lib/api/types"
import {
  CircularProgress,
  LinearProgress,
  TimeEstimate,
  DeploymentSteps,
  LogViewer,
  ErrorCard,
  CancelDialog,
  ShareButton,
  SuccessCelebration,
} from "@/components/deployment"

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
    case "canceled":
      return <AlertCircle className={`${className} text-yellow-500`} />
    default:
      return <Loader2 className={`${className} text-primary animate-spin`} />
  }
}

function getStatusBadgeVariant(status: DeploymentStatus): "default" | "destructive" | "secondary" | "outline" {
  switch (status) {
    case "completed":
      return "default"
    case "failed":
      return "destructive"
    case "canceled":
      return "outline"
    default:
      return "secondary"
  }
}

export default function DeploymentPage() {
  const router = useRouter()
  const params = useParams()
  const deploymentId = params.id as string

  const [showCelebration, setShowCelebration] = useState(false)
  const [previousStatus, setPreviousStatus] = useState<DeploymentStatus | null>(null)

  const { data: deployment, isLoading: deploymentLoading, refetch: refetchDeployment } =
    useDeployment(deploymentId)
  const { data: progressData } = useDeploymentProgress(deploymentId)
  const { data: retryInfo } = useRetryInfo(deploymentId)
  const logsStream = useDeploymentLogsStream(deploymentId)
  const cancelDeployment = useCancelDeployment()
  const retryDeployment = useRetryDeployment()

  const isActive = useMemo(() => {
    if (!deployment) return false
    return ["pending", "provisioning", "configuring", "deploying", "verifying"].includes(
      deployment.status
    )
  }, [deployment])

  // Default steps when no progress data available
  const defaultSteps: DeploymentStep[] = useMemo(() => [
    { id: "auth", name: "Authenticating", description: "Authenticating with cloud provider", status: "pending", estimated_time_ms: 5000, current_sub_step: 0 },
    { id: "network", name: "Network Setup", description: "Provisioning network resources", status: "pending", estimated_time_ms: 30000, current_sub_step: 0 },
    { id: "compute", name: "Creating Servers", description: "Creating compute instances", status: "pending", estimated_time_ms: 120000, current_sub_step: 0 },
    { id: "k3s", name: "Installing K3s", description: "Installing Kubernetes cluster", status: "pending", estimated_time_ms: 180000, current_sub_step: 0 },
    { id: "storage", name: "Deploying Storage", description: "Setting up MinIO storage", status: "pending", estimated_time_ms: 60000, current_sub_step: 0 },
    { id: "catalog", name: "Deploying Catalog", description: "Setting up Lakekeeper catalog", status: "pending", estimated_time_ms: 45000, current_sub_step: 0 },
    { id: "philotes", name: "Deploying Philotes", description: "Installing Philotes services", status: "pending", estimated_time_ms: 90000, current_sub_step: 0 },
    { id: "health", name: "Health Checks", description: "Running health verification", status: "pending", estimated_time_ms: 30000, current_sub_step: 0 },
    { id: "ssl", name: "SSL Configuration", description: "Configuring TLS certificates", status: "pending", estimated_time_ms: 60000, current_sub_step: 0 },
    { id: "ready", name: "Ready", description: "Deployment complete", status: "pending", estimated_time_ms: 5000, current_sub_step: 0 },
  ], [])

  // Merge WebSocket step updates with progress data
  const steps: DeploymentStep[] = useMemo(() => {
    // Start with REST API progress or default steps
    const baseSteps = progressData?.steps || defaultSteps

    // Apply WebSocket step updates if available
    if (logsStream.stepUpdates.size > 0) {
      return baseSteps.map(step => {
        const update = logsStream.stepUpdates.get(step.id)
        if (update) {
          return {
            ...step,
            status: update.status,
            elapsed_time_ms: update.elapsed_time_ms,
          }
        }
        return step
      })
    }

    return baseSteps
  }, [progressData, defaultSteps, logsStream.stepUpdates])

  // Calculate overall progress percentage
  const overallProgress = useMemo(() => {
    // Prefer WebSocket progress
    if (logsStream.progress?.overall_percent !== undefined) {
      return logsStream.progress.overall_percent
    }
    // Fall back to REST API progress
    if (progressData?.overall_progress !== undefined) {
      return progressData.overall_progress
    }
    // Calculate from steps
    const completedSteps = steps.filter(s => s.status === "completed").length
    return Math.round((completedSteps / steps.length) * 100)
  }, [logsStream.progress, progressData, steps])

  // Get current step index
  const currentStepIndex = useMemo(() => {
    if (logsStream.progress?.current_step_index !== undefined) {
      return logsStream.progress.current_step_index
    }
    if (progressData?.current_step_index !== undefined) {
      return progressData.current_step_index
    }
    // Find current step from status
    const idx = steps.findIndex(s => s.status === "in_progress")
    return idx >= 0 ? idx : 0
  }, [logsStream.progress, progressData, steps])

  // Calculate time estimates (in milliseconds)
  const estimatedRemainingMs = useMemo(() => {
    // Get remaining time from WebSocket progress
    if (logsStream.progress?.estimated_remaining_ms !== undefined) {
      return logsStream.progress.estimated_remaining_ms
    }
    // Fall back to REST API progress
    if (progressData?.estimated_remaining_ms !== undefined) {
      return progressData.estimated_remaining_ms
    }
    // Estimate from steps
    const remainingSteps = steps.filter(s => s.status === "pending" || s.status === "in_progress")
    return remainingSteps.reduce((acc, s) => acc + (s.estimated_time_ms || 0), 0)
  }, [logsStream.progress, progressData, steps])

  // Connect to WebSocket when deployment is active
  useEffect(() => {
    if (isActive && !logsStream.connected) {
      logsStream.connect()
    }
    return () => {
      logsStream.disconnect()
    }
  }, [isActive, logsStream])

  // Trigger celebration when deployment completes
  useEffect(() => {
    if (!deployment) return

    // Use a microtask to batch state updates and avoid cascading renders
    const scheduleUpdates = () => {
      if (deployment.status === "completed" && previousStatus !== "completed") {
        setShowCelebration(true)
      }
      setPreviousStatus(deployment.status)
    }

    // Schedule to avoid direct setState in effect body
    queueMicrotask(scheduleUpdates)
  }, [deployment, previousStatus])

  const handleCancel = async () => {
    if (!deployment) return
    await cancelDeployment.mutateAsync(deployment.id)
  }

  const handleRetry = async () => {
    if (!deployment) return
    try {
      await retryDeployment.mutateAsync(deployment.id)
      refetchDeployment()
    } catch (error) {
      console.error("Failed to retry deployment:", error)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  if (deploymentLoading) {
    return (
      <div className="container max-w-5xl py-8 space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <Skeleton className="h-[200px] w-full" />
            <Skeleton className="h-[400px] w-full" />
          </div>
          <div className="space-y-6">
            <Skeleton className="h-[150px] w-full" />
            <Skeleton className="h-[200px] w-full" />
          </div>
        </div>
      </div>
    )
  }

  if (!deployment) {
    return (
      <div className="container max-w-5xl py-8">
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
  const isCancelled = deployment.status === "canceled"
  const isTerminal = isComplete || isFailed || isCancelled
  const canRetry = isFailed && retryInfo?.can_retry

  // Get error info from progress or deployment
  const failedStep = steps.find(s => s.status === "failed")
  const errorInfo = failedStep?.error || (isFailed && deployment.error_message ? {
    code: "DEPLOYMENT_FAILED",
    message: deployment.error_message,
    suggestions: [],
    retryable: false,
  } : null)


  return (
    <div className="container max-w-5xl py-8 space-y-6">
      {/* Success Celebration */}
      <SuccessCelebration
        show={showCelebration}
        onComplete={() => setShowCelebration(false)}
      />

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div>
            <h1 className="text-3xl font-bold">{deployment.name}</h1>
            <p className="text-muted-foreground">
              Deploying to {deployment.provider} in {deployment.region}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <ShareButton deploymentId={deployment.id} />
          <Badge
            variant={getStatusBadgeVariant(deployment.status)}
            className="text-lg px-4 py-2"
          >
            {deployment.status}
          </Badge>
        </div>
      </div>

      {/* Progress Overview */}
      {!isTerminal && (
        <Card>
          <CardContent className="py-6">
            <div className="flex items-center gap-8">
              <CircularProgress
                value={overallProgress}
                size="lg"
              />
              <div className="flex-1 space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-lg font-semibold">Deployment in Progress</h3>
                    <p className="text-muted-foreground text-sm">
                      {steps.find(s => s.status === "in_progress")?.name || "Initializing..."}
                    </p>
                  </div>
                  <TimeEstimate
                    startedAt={deployment.started_at || deployment.created_at}
                    estimatedRemainingMs={estimatedRemainingMs}
                    showElapsed={true}
                    showRemaining={true}
                  />
                </div>
                <LinearProgress value={overallProgress} showLabel={false} />
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Progress and Logs */}
        <div className="lg:col-span-2 space-y-6">
          {/* Progress Steps */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <StatusIcon status={deployment.status} />
                Deployment Steps
              </CardTitle>
              <CardDescription>
                {isComplete
                  ? "All steps completed successfully"
                  : isFailed
                    ? "Deployment failed - see error details below"
                    : `Step ${currentStepIndex + 1} of ${steps.length}`}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DeploymentSteps
                steps={steps}
                logs={logsStream.logs}
              />
            </CardContent>
          </Card>

          {/* Error Card */}
          {isFailed && errorInfo && (
            <ErrorCard
              error={errorInfo}
              onRetry={canRetry ? handleRetry : undefined}
              isRetrying={retryDeployment.isPending}
              stepName={retryInfo?.failed_step?.name || failedStep?.name}
            />
          )}

          {/* Deployment Logs */}
          <LogViewer
            logs={logsStream.logs}
            groupByStep={true}
            autoScroll={isActive}
            maxHeight="400px"
          />
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
                <CancelDialog
                  resources={progressData?.resources_created}
                  onCancel={handleCancel}
                  isLoading={cancelDeployment.isPending}
                >
                  <Button
                    variant="destructive"
                    className="w-full"
                    disabled={cancelDeployment.isPending}
                  >
                    Cancel Deployment
                  </Button>
                </CancelDialog>
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

              {isFailed && canRetry && (
                <Button
                  variant="default"
                  className="w-full"
                  onClick={handleRetry}
                  disabled={retryDeployment.isPending}
                >
                  {retryDeployment.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Retrying...
                    </>
                  ) : (
                    <>
                      <RefreshCw className="mr-2 h-4 w-4" />
                      Retry from {retryInfo?.failed_step?.name || "Failed Step"}
                    </>
                  )}
                </Button>
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
              {deployment.started_at && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Started</span>
                  <span>
                    {new Date(deployment.started_at).toLocaleTimeString()}
                  </span>
                </div>
              )}
              {deployment.completed_at && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Completed</span>
                  <span>
                    {new Date(deployment.completed_at).toLocaleTimeString()}
                  </span>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Created Resources (during deployment) */}
          {progressData?.resources_created && progressData.resources_created.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Created Resources</CardTitle>
                <CardDescription>
                  {progressData.resources_created.length} resource(s) created
                </CardDescription>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2 text-sm">
                  {progressData.resources_created.map((resource, index) => (
                    <li key={index} className="flex items-center gap-2">
                      <CheckCircle className="h-3 w-3 text-green-500" />
                      <span className="text-muted-foreground">{resource.type}:</span>
                      <span className="truncate">{resource.name || resource.id}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )}

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
