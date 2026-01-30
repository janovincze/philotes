"use client"

import { useState, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { ArrowLeft, ArrowRight, GitBranch, CheckCircle2, AlertCircle, Loader2 } from "lucide-react"
import { StepTables } from "@/components/setup/step-tables"
import { StepConfigure } from "@/components/setup/step-configure"
import { pipelinesApi } from "@/lib/api"
import { toast } from "sonner"
import type { Source, Pipeline, TableInfo } from "@/lib/api/types"

interface StepPipelineWrapperProps {
  source: Source | null
  pipeline: Pipeline | null
  onPipelineCreated: (pipeline: Pipeline | null) => void
  onNext: (data?: Record<string, unknown>) => void
  onBack: () => void
}

type SubStep = "tables" | "configure" | "review"

export function StepPipelineWrapper({
  source,
  pipeline,
  onPipelineCreated,
  onNext,
  onBack,
}: StepPipelineWrapperProps) {
  const [subStep, setSubStep] = useState<SubStep>("tables")
  const [availableTables, setAvailableTables] = useState<TableInfo[]>([])
  const [selectedTables, setSelectedTables] = useState<string[]>([])
  const [pipelineName, setPipelineName] = useState("")
  const [isCreating, setIsCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleTablesNext = useCallback(() => {
    setSubStep("configure")
  }, [])

  const handleConfigureNext = useCallback(() => {
    setSubStep("review")
  }, [])

  const handleCreatePipeline = useCallback(async () => {
    if (!source) return

    setIsCreating(true)
    setError(null)

    try {
      // Create pipeline
      const newPipeline = await pipelinesApi.create({
        name: pipelineName,
        source_id: source.id,
        tables: selectedTables.map((table) => ({
          table,
          enabled: true,
        })),
      })

      onPipelineCreated(newPipeline)

      // Try to start the pipeline
      try {
        await pipelinesApi.start(newPipeline.id)
      } catch {
        // Pipeline created but failed to start - still continue
        console.warn("Pipeline created but failed to auto-start")
      }

      toast.success("Pipeline created", {
        description: `Pipeline "${pipelineName}" has been created successfully.`,
      })

      onNext({
        pipeline_id: newPipeline.id,
        pipeline_name: newPipeline.name,
        tables_count: selectedTables.length,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create pipeline")
      toast.error("Failed to create pipeline", {
        description: err instanceof Error ? err.message : "An error occurred",
      })
    } finally {
      setIsCreating(false)
    }
  }, [source, pipelineName, selectedTables, onPipelineCreated, onNext])

  const handleNext = useCallback(() => {
    onNext({
      pipeline_id: pipeline?.id,
      pipeline_name: pipeline?.name,
    })
  }, [pipeline, onNext])

  // If pipeline already created, show summary
  if (pipeline) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Create Pipeline</h2>
          <p className="text-muted-foreground mt-2">
            Configure and create a CDC pipeline to replicate data to your data lake.
          </p>
        </div>

        <Alert className="bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-800">
          <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
          <AlertTitle>Pipeline created</AlertTitle>
          <AlertDescription>
            <span className="font-medium">{pipeline.name}</span> is ready.
          </AlertDescription>
        </Alert>

        <div className="rounded-lg border p-4">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary/10 rounded-lg">
              <GitBranch className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="font-medium">{pipeline.name}</p>
              <p className="text-sm text-muted-foreground">
                Status: {pipeline.status} • {pipeline.tables?.length || 0} tables
              </p>
            </div>
          </div>
        </div>

        <div className="flex justify-between pt-4">
          <Button variant="outline" onClick={onBack}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <Button onClick={handleNext}>
            Continue
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    )
  }

  // No source selected
  if (!source) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Create Pipeline</h2>
          <p className="text-muted-foreground mt-2">
            Configure and create a CDC pipeline to replicate data to your data lake.
          </p>
        </div>

        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>No source database</AlertTitle>
          <AlertDescription>
            Please go back and connect a source database first.
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

  // Sub-step: Select tables
  if (subStep === "tables") {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Select Tables</h2>
          <p className="text-muted-foreground mt-2">
            Choose which tables to replicate from your source database.
          </p>
        </div>

        <StepTables
          sourceId={source.id}
          availableTables={availableTables}
          onTablesLoaded={setAvailableTables}
          selectedTables={selectedTables}
          onSelectedTablesChange={setSelectedTables}
          onNext={handleTablesNext}
          onBack={onBack}
        />
      </div>
    )
  }

  // Sub-step: Configure pipeline
  if (subStep === "configure") {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Configure Pipeline</h2>
          <p className="text-muted-foreground mt-2">
            Name your pipeline and configure its settings.
          </p>
        </div>

        <StepConfigure
          pipelineName={pipelineName}
          onPipelineNameChange={setPipelineName}
          sourceName={source.name}
          selectedTablesCount={selectedTables.length}
          onNext={handleConfigureNext}
          onBack={() => setSubStep("tables")}
        />
      </div>
    )
  }

  // Sub-step: Review and create
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">Review & Create</h2>
        <p className="text-muted-foreground mt-2">
          Review your pipeline configuration before creating it.
        </p>
      </div>

      <div className="space-y-4">
        <div className="rounded-lg border p-4 space-y-3">
          <h3 className="font-medium">Pipeline Configuration</h3>
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt className="text-muted-foreground">Name:</dt>
            <dd className="font-medium">{pipelineName}</dd>
            <dt className="text-muted-foreground">Source:</dt>
            <dd className="font-medium">{source.name}</dd>
            <dt className="text-muted-foreground">Tables:</dt>
            <dd className="font-medium">{selectedTables.length} selected</dd>
          </dl>
        </div>

        {selectedTables.length > 0 && (
          <div className="rounded-lg border p-4">
            <h3 className="font-medium mb-2">Selected Tables</h3>
            <div className="flex flex-wrap gap-2">
              {selectedTables.slice(0, 10).map((table) => (
                <span
                  key={table}
                  className="px-2 py-1 bg-muted rounded text-xs font-mono"
                >
                  {table}
                </span>
              ))}
              {selectedTables.length > 10 && (
                <span className="px-2 py-1 text-xs text-muted-foreground">
                  +{selectedTables.length - 10} more
                </span>
              )}
            </div>
          </div>
        )}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex justify-between pt-4">
        <Button variant="outline" onClick={() => setSubStep("configure")}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Button onClick={handleCreatePipeline} disabled={isCreating || !pipelineName}>
          {isCreating ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Creating...
            </>
          ) : (
            <>
              Create Pipeline
              <ArrowRight className="ml-2 h-4 w-4" />
            </>
          )}
        </Button>
      </div>
    </div>
  )
}
