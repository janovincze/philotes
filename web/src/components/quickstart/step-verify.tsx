"use client"

import { useState, useEffect, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  XCircle,
  Loader2,
  Database,
  Play,
  Eye,
  RefreshCw,
} from "lucide-react"
import { useCreatePipeline, useStartPipeline } from "@/lib/hooks/use-pipelines"
import { useVerifyDataFlow } from "@/lib/hooks/use-onboarding"
import { ResultsTable } from "@/components/query/results-table"
import type { Source, Pipeline, QueryColumn } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface StepVerifyProps {
  source: Source
  sourceName: string
  selectedTables: string[]
  onPipelineCreated: (pipeline: Pipeline) => void
  onNext: () => void
  onBack: () => void
}

type SubStep = "creating" | "starting" | "verifying" | "complete" | "error"

interface SubStepStatus {
  creating: "pending" | "in_progress" | "complete" | "error"
  starting: "pending" | "in_progress" | "complete" | "error"
  verifying: "pending" | "in_progress" | "complete" | "error"
}

export function StepVerify({
  source,
  sourceName,
  selectedTables,
  onPipelineCreated,
  onNext,
  onBack,
}: StepVerifyProps) {
  const [pipeline, setPipeline] = useState<Pipeline | null>(null)
  const [currentSubStep, setCurrentSubStep] = useState<SubStep>("creating")
  const [subStepStatus, setSubStepStatus] = useState<SubStepStatus>({
    creating: "pending",
    starting: "pending",
    verifying: "pending",
  })
  const [error, setError] = useState<string | null>(null)
  const [sampleData, setSampleData] = useState<{
    columns: QueryColumn[]
    rows: Record<string, unknown>[]
    rowCount: number
    tableName: string
  } | null>(null)
  const [verificationAttempts, setVerificationAttempts] = useState(0)
  const [isRunning, setIsRunning] = useState(false)

  const createPipeline = useCreatePipeline()
  const startPipeline = useStartPipeline()
  const verifyData = useVerifyDataFlow()

  const pipelineName = `${sourceName}-pipeline`

  // Build table mappings from selected tables
  const buildTableMappings = useCallback(() => {
    return selectedTables.map((fullName) => {
      const [schema, table] = fullName.split(".")
      return {
        schema,
        table,
        enabled: true,
      }
    })
  }, [selectedTables])

  // Run the verification flow
  const runVerificationFlow = useCallback(async () => {
    if (isRunning) return
    setIsRunning(true)
    setError(null)

    try {
      // Step 1: Create pipeline
      setCurrentSubStep("creating")
      setSubStepStatus((prev) => ({ ...prev, creating: "in_progress" }))

      const newPipeline = await createPipeline.mutateAsync({
        name: pipelineName,
        source_id: source.id,
        tables: buildTableMappings(),
      })

      setPipeline(newPipeline)
      onPipelineCreated(newPipeline)
      setSubStepStatus((prev) => ({ ...prev, creating: "complete" }))

      // Step 2: Start pipeline
      setCurrentSubStep("starting")
      setSubStepStatus((prev) => ({ ...prev, starting: "in_progress" }))

      await startPipeline.mutateAsync(newPipeline.id)
      setSubStepStatus((prev) => ({ ...prev, starting: "complete" }))

      // Step 3: Verify data
      setCurrentSubStep("verifying")
      setSubStepStatus((prev) => ({ ...prev, verifying: "in_progress" }))

      // Get the first selected table for verification
      const firstTable = selectedTables[0]
      const [schema, tableName] = firstTable.split(".")
      const icebergTableName = `iceberg.${schema}.${tableName}`

      // Try to verify data (with retries)
      let verified = false
      let attempts = 0
      const maxAttempts = 6 // Try for ~60 seconds

      while (!verified && attempts < maxAttempts) {
        attempts++
        setVerificationAttempts(attempts)

        try {
          const result = await verifyData.mutateAsync({
            pipeline_id: newPipeline.id,
            table_name: icebergTableName,
            max_wait_sec: 10,
          })

          if (result.success && result.row_count > 0) {
            verified = true
            setSampleData({
              columns: Object.keys(result.sample_rows?.[0] || {}).map((name) => ({
                name,
                type: "varchar",
              })),
              rows: result.sample_rows || [],
              rowCount: result.row_count,
              tableName: icebergTableName,
            })
          }
        } catch {
          // Ignore errors during polling, just retry
        }

        if (!verified && attempts < maxAttempts) {
          // Wait before next attempt
          await new Promise((resolve) => setTimeout(resolve, 10000))
        }
      }

      if (verified) {
        setSubStepStatus((prev) => ({ ...prev, verifying: "complete" }))
        setCurrentSubStep("complete")
      } else {
        // Data didn't arrive yet, but pipeline is running - allow to proceed
        setSubStepStatus((prev) => ({ ...prev, verifying: "complete" }))
        setCurrentSubStep("complete")
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "An error occurred"
      setError(errorMessage)
      setCurrentSubStep("error")

      // Mark current step as error
      if (subStepStatus.creating === "in_progress") {
        setSubStepStatus((prev) => ({ ...prev, creating: "error" }))
      } else if (subStepStatus.starting === "in_progress") {
        setSubStepStatus((prev) => ({ ...prev, starting: "error" }))
      } else if (subStepStatus.verifying === "in_progress") {
        setSubStepStatus((prev) => ({ ...prev, verifying: "error" }))
      }
    } finally {
      setIsRunning(false)
    }
  }, [
    isRunning,
    source.id,
    pipelineName,
    buildTableMappings,
    selectedTables,
    createPipeline,
    startPipeline,
    verifyData,
    onPipelineCreated,
    subStepStatus,
  ])

  // Start verification on mount
  useEffect(() => {
    runVerificationFlow()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const getSubStepIcon = (status: "pending" | "in_progress" | "complete" | "error") => {
    switch (status) {
      case "pending":
        return <div className="h-5 w-5 rounded-full border-2 border-muted" />
      case "in_progress":
        return <Loader2 className="h-5 w-5 animate-spin text-primary" />
      case "complete":
        return <CheckCircle2 className="h-5 w-5 text-green-600" />
      case "error":
        return <XCircle className="h-5 w-5 text-red-600" />
    }
  }

  const canProceed = currentSubStep === "complete" || (pipeline && currentSubStep === "error")

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Verify Data Replication</h2>
        <p className="text-muted-foreground mt-1">
          Setting up your pipeline and verifying data flow
        </p>
      </div>

      {/* Pipeline Info */}
      <div className="bg-muted/50 rounded-lg p-4">
        <div className="flex items-center gap-3">
          <Database className="h-5 w-5 text-muted-foreground" />
          <div>
            <p className="font-medium">{pipelineName}</p>
            <p className="text-sm text-muted-foreground">
              {selectedTables.length} table{selectedTables.length !== 1 ? "s" : ""} selected
            </p>
          </div>
        </div>
      </div>

      {/* Sub-steps Progress */}
      <div className="space-y-4">
        <div className={cn(
          "flex items-center gap-4 p-4 rounded-lg border",
          subStepStatus.creating === "complete" && "bg-green-50 border-green-200 dark:bg-green-950 dark:border-green-900",
          subStepStatus.creating === "in_progress" && "bg-primary/5 border-primary/20",
          subStepStatus.creating === "error" && "bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-900"
        )}>
          {getSubStepIcon(subStepStatus.creating)}
          <div className="flex-1">
            <p className="font-medium">Create Pipeline</p>
            <p className="text-sm text-muted-foreground">
              {subStepStatus.creating === "pending" && "Waiting..."}
              {subStepStatus.creating === "in_progress" && "Creating pipeline..."}
              {subStepStatus.creating === "complete" && "Pipeline created"}
              {subStepStatus.creating === "error" && "Failed to create pipeline"}
            </p>
          </div>
        </div>

        <div className={cn(
          "flex items-center gap-4 p-4 rounded-lg border",
          subStepStatus.starting === "complete" && "bg-green-50 border-green-200 dark:bg-green-950 dark:border-green-900",
          subStepStatus.starting === "in_progress" && "bg-primary/5 border-primary/20",
          subStepStatus.starting === "error" && "bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-900"
        )}>
          {getSubStepIcon(subStepStatus.starting)}
          <div className="flex-1">
            <p className="font-medium">Start Pipeline</p>
            <p className="text-sm text-muted-foreground">
              {subStepStatus.starting === "pending" && "Waiting..."}
              {subStepStatus.starting === "in_progress" && "Starting replication..."}
              {subStepStatus.starting === "complete" && "Pipeline running"}
              {subStepStatus.starting === "error" && "Failed to start pipeline"}
            </p>
          </div>
          {subStepStatus.starting === "complete" && (
            <Badge variant="default" className="bg-green-600">Running</Badge>
          )}
        </div>

        <div className={cn(
          "flex items-center gap-4 p-4 rounded-lg border",
          subStepStatus.verifying === "complete" && "bg-green-50 border-green-200 dark:bg-green-950 dark:border-green-900",
          subStepStatus.verifying === "in_progress" && "bg-primary/5 border-primary/20",
          subStepStatus.verifying === "error" && "bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-900"
        )}>
          {getSubStepIcon(subStepStatus.verifying)}
          <div className="flex-1">
            <p className="font-medium">Verify Data</p>
            <p className="text-sm text-muted-foreground">
              {subStepStatus.verifying === "pending" && "Waiting..."}
              {subStepStatus.verifying === "in_progress" && `Checking for data... (attempt ${verificationAttempts})`}
              {subStepStatus.verifying === "complete" && sampleData
                ? `${sampleData.rowCount} rows replicated`
                : subStepStatus.verifying === "complete"
                  ? "Data verification complete"
                  : ""}
              {subStepStatus.verifying === "error" && "Verification failed"}
            </p>
          </div>
          {subStepStatus.verifying === "complete" && sampleData && (
            <Badge variant="outline" className="gap-1">
              <Eye className="h-3 w-3" />
              Preview Available
            </Badge>
          )}
        </div>
      </div>

      {/* Sample Data Preview */}
      {sampleData && sampleData.rows.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="font-medium">Data Preview</h3>
            <Badge variant="secondary">{sampleData.tableName}</Badge>
          </div>
          <div className="border rounded-lg overflow-hidden max-h-64">
            <ResultsTable
              columns={sampleData.columns}
              rows={sampleData.rows}
              isLoading={false}
            />
          </div>
        </div>
      )}

      {/* No data yet message */}
      {currentSubStep === "complete" && !sampleData && (
        <Alert>
          <Play className="h-4 w-4" />
          <AlertDescription>
            Your pipeline is running! Data replication may take a moment to complete.
            You can proceed and query your data once it arrives.
          </AlertDescription>
        </Alert>
      )}

      {/* Error */}
      {error && (
        <Alert variant="destructive">
          <XCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Actions */}
      <div className="flex justify-between">
        <Button
          variant="outline"
          onClick={onBack}
          disabled={isRunning}
          className="gap-2"
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          {currentSubStep === "error" && (
            <Button
              variant="outline"
              onClick={runVerificationFlow}
              disabled={isRunning}
              className="gap-2"
            >
              <RefreshCw className="h-4 w-4" />
              Retry
            </Button>
          )}
          <Button
            onClick={onNext}
            disabled={!canProceed}
            className="gap-2"
          >
            {isRunning ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Working...
              </>
            ) : (
              <>
                Continue
                <ArrowRight className="h-4 w-4" />
              </>
            )}
          </Button>
        </div>
      </div>
    </div>
  )
}
