"use client"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ArrowLeft, ArrowRight, Settings, Lightbulb } from "lucide-react"

interface StepConfigureProps {
  pipelineName: string
  onPipelineNameChange: (name: string) => void
  sourceName: string
  selectedTablesCount: number
  onNext: () => void
  onBack: () => void
}

export function StepConfigure({
  pipelineName,
  onPipelineNameChange,
  sourceName,
  selectedTablesCount,
  onNext,
  onBack,
}: StepConfigureProps) {
  const suggestedName = sourceName ? `${sourceName} Pipeline` : ""

  const handleUseSuggestion = () => {
    onPipelineNameChange(suggestedName)
  }

  const isValid = pipelineName.trim().length > 0 && pipelineName.length <= 255

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Settings className="h-5 w-5 text-primary" />
          <h2 className="text-xl font-semibold">Configure Your Pipeline</h2>
        </div>
        <p className="text-sm text-muted-foreground">
          Give your pipeline a name. You can adjust additional settings later.
        </p>
      </div>

      {/* Summary */}
      <div className="bg-muted/50 rounded-lg p-4 space-y-2">
        <p className="text-sm">
          <span className="text-muted-foreground">Source:</span>{" "}
          <span className="font-medium">{sourceName || "Unknown"}</span>
        </p>
        <p className="text-sm">
          <span className="text-muted-foreground">Tables to replicate:</span>{" "}
          <span className="font-medium">{selectedTablesCount}</span>
        </p>
      </div>

      {/* Pipeline Name */}
      <div className="space-y-2">
        <Label htmlFor="pipeline_name">Pipeline Name</Label>
        <Input
          id="pipeline_name"
          placeholder="e.g., Production CDC Pipeline"
          value={pipelineName}
          onChange={(e) => onPipelineNameChange(e.target.value)}
          maxLength={255}
        />
        <p className="text-xs text-muted-foreground">
          A descriptive name to identify this replication pipeline
        </p>
        {suggestedName && pipelineName !== suggestedName && (
          <button
            type="button"
            onClick={handleUseSuggestion}
            className="flex items-center gap-1 text-xs text-primary hover:underline"
          >
            <Lightbulb className="h-3 w-3" />
            Suggestion: {suggestedName}
          </button>
        )}
      </div>

      {/* Smart Defaults Info */}
      <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
        <h3 className="font-medium text-blue-800 dark:text-blue-200 mb-2">
          Smart Defaults Applied
        </h3>
        <ul className="text-sm text-blue-700 dark:text-blue-300 space-y-1">
          <li>- Batch size: 1,000 events</li>
          <li>- Flush interval: 10 seconds</li>
          <li>- Checkpoint frequency: 30 seconds</li>
          <li>- Automatic retry on failures</li>
        </ul>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2">
          These settings work well for most use cases. You can customize them after creation.
        </p>
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between pt-4">
        <Button variant="outline" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Button onClick={onNext} disabled={!isValid}>
          Review & Create
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
