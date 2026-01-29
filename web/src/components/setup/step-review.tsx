"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { ArrowLeft, Rocket, Loader2, Database, Table, GitBranch } from "lucide-react"
import { useCreatePipeline, useStartPipeline } from "@/lib/hooks/use-pipelines"
import type { Source } from "@/lib/api/types"
import type { SourceFormData } from "./setup-wizard"

interface StepReviewProps {
  sourceFormData: SourceFormData
  source: Source | null
  selectedTables: string[]
  pipelineName: string
  onPipelineCreated: (id: string) => void
  onNext: () => void
  onBack: () => void
}

export function StepReview({
  sourceFormData,
  source,
  selectedTables,
  pipelineName,
  onPipelineCreated,
  onNext,
  onBack,
}: StepReviewProps) {
  const [error, setError] = useState<string | null>(null)
  const createPipeline = useCreatePipeline()
  const startPipeline = useStartPipeline()

  const handleCreate = async () => {
    if (!source) {
      setError("Source not found")
      return
    }

    setError(null)

    try {
      // Create pipeline with table mappings
      const tableMappings = selectedTables.map((fullName) => {
        const [schema, table] = fullName.split(".")
        return {
          schema: schema || "public",
          table: table,
          enabled: true,
        }
      })

      const pipeline = await createPipeline.mutateAsync({
        name: pipelineName,
        source_id: source.id,
        tables: tableMappings,
      })

      onPipelineCreated(pipeline.id)

      // Start the pipeline
      await startPipeline.mutateAsync(pipeline.id)

      onNext()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create pipeline")
    }
  }

  const isLoading = createPipeline.isPending || startPipeline.isPending

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Rocket className="h-5 w-5 text-primary" />
          <h2 className="text-xl font-semibold">Review & Create</h2>
        </div>
        <p className="text-sm text-muted-foreground">
          Review your configuration before creating the pipeline.
        </p>
      </div>

      {/* Configuration Summary */}
      <div className="space-y-4">
        {/* Source Details */}
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2">
            <Database className="h-4 w-4 text-primary" />
            <h3 className="font-medium">Source Database</h3>
          </div>
          <div className="grid gap-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Name</span>
              <span className="font-medium">{sourceFormData.name}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Host</span>
              <span className="font-mono text-xs">
                {sourceFormData.host}:{sourceFormData.port}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Database</span>
              <span>{sourceFormData.database_name}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">User</span>
              <span>{sourceFormData.username}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">SSL</span>
              <Badge variant="outline" className="text-xs">
                {sourceFormData.ssl_mode}
              </Badge>
            </div>
          </div>
        </div>

        {/* Tables */}
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2">
            <Table className="h-4 w-4 text-primary" />
            <h3 className="font-medium">
              Tables ({selectedTables.length})
            </h3>
          </div>
          <div className="flex flex-wrap gap-2">
            {selectedTables.slice(0, 10).map((table) => (
              <Badge key={table} variant="secondary" className="font-mono text-xs">
                {table}
              </Badge>
            ))}
            {selectedTables.length > 10 && (
              <Badge variant="outline" className="text-xs">
                +{selectedTables.length - 10} more
              </Badge>
            )}
          </div>
        </div>

        {/* Pipeline */}
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-2">
            <GitBranch className="h-4 w-4 text-primary" />
            <h3 className="font-medium">Pipeline</h3>
          </div>
          <div className="grid gap-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Name</span>
              <span className="font-medium">{pipelineName}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Destination</span>
              <span>Apache Iceberg</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Auto-start</span>
              <Badge variant="outline" className="text-xs">Yes</Badge>
            </div>
          </div>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-destructive/10 text-destructive rounded-lg p-4 text-sm">
          {error}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between pt-4">
        <Button variant="outline" onClick={onBack} disabled={isLoading}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Button onClick={handleCreate} disabled={isLoading}>
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Creating...
            </>
          ) : (
            <>
              <Rocket className="mr-2 h-4 w-4" />
              Create Pipeline
            </>
          )}
        </Button>
      </div>
    </div>
  )
}
