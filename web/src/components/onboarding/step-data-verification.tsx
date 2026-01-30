"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import {
  ArrowLeft,
  ArrowRight,
  Loader2,
  CheckCircle2,
  XCircle,
  AlertCircle,
  RefreshCw,
  Database,
  Table,
} from "lucide-react"
import { useVerifyDataFlow } from "@/lib/hooks/use-onboarding"
import { usePipelineStatus } from "@/lib/hooks/use-pipelines"
import type { Pipeline } from "@/lib/api/types"

interface StepDataVerificationProps {
  pipeline: Pipeline | null
  onNext: (data?: Record<string, unknown>) => void
  onBack: () => void
  onDataVerified: (verified: boolean) => void
}

export function StepDataVerification({
  pipeline,
  onNext,
  onBack,
  onDataVerified,
}: StepDataVerificationProps) {
  const [verified, setVerified] = useState(false)
  const [attempts, setAttempts] = useState(0)
  const maxAttempts = 3

  const verifyMutation = useVerifyDataFlow()
  const { data: statusData, isLoading: isLoadingStatus } = usePipelineStatus(
    pipeline?.id ?? "",
    !!pipeline && !verified
  )

  const pipelineStatus = statusData?.status ?? pipeline?.status ?? "unknown"
  const isRunning = pipelineStatus === "running"

  const handleVerify = () => {
    if (!pipeline || !pipeline.tables?.length) return

    setAttempts((prev) => prev + 1)

    const firstTable = pipeline.tables[0]
    const tableName = firstTable.target_table || firstTable.source_table

    verifyMutation
      .mutateAsync({
        pipeline_id: pipeline.id,
        table_name: tableName,
        max_wait_sec: 60,
      })
      .then((result) => {
        if (result.success) {
          setVerified(true)
          onDataVerified(true)
        }
      })
      .catch((error) => {
        console.error("Verification failed:", error)
      })
  }

  const handleNext = () => {
    onNext({
      data_verified: verified,
      verification_attempts: attempts,
    })
  }

  const handleSkipVerification = () => {
    onNext({
      data_verified: false,
      verification_skipped: true,
    })
  }

  // No pipeline
  if (!pipeline) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Verify Data Flow</h2>
          <p className="text-muted-foreground mt-2">
            Confirm that data is flowing from your source database to the Iceberg data lake.
          </p>
        </div>

        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>No pipeline configured</AlertTitle>
          <AlertDescription>
            Please go back and create a pipeline first.
          </AlertDescription>
        </Alert>

        <div className="flex justify-between pt-4">
          <Button variant="outline" onClick={onBack}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">Verify Data Flow</h2>
        <p className="text-muted-foreground mt-2">
          Confirm that data is flowing from your source database to the Iceberg data lake.
        </p>
      </div>

      {/* Pipeline Status */}
      <div className="rounded-lg border p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Database className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="font-medium">{pipeline.name}</p>
              <p className="text-sm text-muted-foreground">
                {pipeline.tables?.length || 0} tables configured
              </p>
            </div>
          </div>
          <Badge
            variant="outline"
            className={
              isRunning
                ? "bg-green-100 text-green-700 border-green-200 dark:bg-green-900/30 dark:text-green-400"
                : "bg-yellow-100 text-yellow-700 border-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400"
            }
          >
            {isLoadingStatus ? "Checking..." : pipelineStatus}
          </Badge>
        </div>
      </div>

      {/* Verification Status */}
      {verified ? (
        <Alert className="bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-800">
          <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
          <AlertTitle>Data verified successfully</AlertTitle>
          <AlertDescription>
            Data is flowing correctly from your source to the Iceberg data lake.
            {verifyMutation.data && (
              <span className="block mt-1">
                Found {verifyMutation.data.row_count} rows in {verifyMutation.data.query_time_ms}ms
              </span>
            )}
          </AlertDescription>
        </Alert>
      ) : verifyMutation.isPending ? (
        <Alert>
          <Loader2 className="h-4 w-4 animate-spin" />
          <AlertTitle>Verifying data flow...</AlertTitle>
          <AlertDescription>
            Waiting for initial data to appear in the data lake. This may take a moment.
          </AlertDescription>
        </Alert>
      ) : verifyMutation.isError ? (
        <Alert variant="destructive">
          <XCircle className="h-4 w-4" />
          <AlertTitle>Verification failed</AlertTitle>
          <AlertDescription>
            {attempts < maxAttempts ? (
              <>
                Unable to verify data flow. This could be normal if no changes have occurred yet.
                <Button
                  variant="link"
                  size="sm"
                  className="p-0 h-auto ml-2"
                  onClick={handleVerify}
                >
                  <RefreshCw className="h-3 w-3 mr-1" />
                  Retry ({attempts}/{maxAttempts})
                </Button>
              </>
            ) : (
              "Max verification attempts reached. You can continue and check the dashboard later."
            )}
          </AlertDescription>
        </Alert>
      ) : !isRunning ? (
        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Pipeline not running</AlertTitle>
          <AlertDescription>
            The pipeline is not yet running. Click &quot;Verify Now&quot; when the pipeline status shows as running.
          </AlertDescription>
        </Alert>
      ) : (
        <Alert>
          <Table className="h-4 w-4" />
          <AlertTitle>Ready to verify</AlertTitle>
          <AlertDescription>
            The pipeline is running. Click &quot;Verify Now&quot; to check if data has been replicated to
            the data lake.
          </AlertDescription>
        </Alert>
      )}

      {/* Verification Details */}
      {verifyMutation.data && verified && (
        <div className="rounded-lg border p-4 space-y-3">
          <h3 className="font-medium">Verification Results</h3>
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt className="text-muted-foreground">Rows found:</dt>
            <dd className="font-medium">{verifyMutation.data.row_count}</dd>
            <dt className="text-muted-foreground">Query time:</dt>
            <dd className="font-medium">{verifyMutation.data.query_time_ms}ms</dd>
          </dl>
        </div>
      )}

      <div className="flex justify-between pt-4">
        <Button variant="outline" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          {!verified && !verifyMutation.isPending && (
            <Button
              variant="outline"
              onClick={handleVerify}
              disabled={verifyMutation.isPending || !isRunning}
            >
              <RefreshCw className="mr-2 h-4 w-4" />
              Verify Now
            </Button>
          )}
          {!verified && (
            <Button variant="ghost" onClick={handleSkipVerification}>
              Skip Verification
            </Button>
          )}
          <Button onClick={handleNext} disabled={!verified && verifyMutation.isPending}>
            {verified ? (
              <>
                Continue
                <ArrowRight className="ml-2 h-4 w-4" />
              </>
            ) : (
              "Continue Anyway"
            )}
          </Button>
        </div>
      </div>
    </div>
  )
}
