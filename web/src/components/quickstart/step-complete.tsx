"use client"

import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  CheckCircle2,
  Database,
  Search,
  ArrowRight,
  Plus,
  ExternalLink,
} from "lucide-react"
import { SuccessCelebration } from "@/components/deployment/success-celebration"
import type { Pipeline } from "@/lib/api/types"

interface StepCompleteProps {
  pipeline: Pipeline
  sourceName: string
  tableCount: number
  onViewPipeline: () => void
  onGoToQuery: () => void
  onCreateAnother: () => void
}

export function StepComplete({
  pipeline,
  sourceName,
  tableCount,
  onViewPipeline,
  onGoToQuery,
  onCreateAnother,
}: StepCompleteProps) {
  const [showCelebration, setShowCelebration] = useState(false)

  useEffect(() => {
    // Trigger celebration after a brief delay
    const timer = setTimeout(() => {
      setShowCelebration(true)
    }, 200)

    return () => clearTimeout(timer)
  }, [])

  return (
    <div className="space-y-8">
      <SuccessCelebration show={showCelebration} />

      {/* Success Header */}
      <div className="text-center space-y-4">
        <div className="flex justify-center">
          <div className="rounded-full bg-green-100 dark:bg-green-900 p-4">
            <CheckCircle2 className="h-12 w-12 text-green-600 dark:text-green-400" />
          </div>
        </div>
        <div>
          <h2 className="text-2xl font-bold text-green-600 dark:text-green-400">
            You&apos;re All Set!
          </h2>
          <p className="text-muted-foreground mt-2">
            Your data is now flowing from PostgreSQL to your Iceberg data lake.
          </p>
        </div>
      </div>

      {/* Summary */}
      <div className="bg-muted/50 rounded-lg p-6 space-y-4">
        <h3 className="font-semibold">What we set up:</h3>
        <div className="grid gap-4">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
              <Database className="h-4 w-4 text-primary" />
            </div>
            <div className="flex-1">
              <p className="font-medium">Source: {sourceName}</p>
              <p className="text-sm text-muted-foreground">PostgreSQL database connected</p>
            </div>
            <Badge variant="outline" className="text-green-600">Connected</Badge>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
              <ExternalLink className="h-4 w-4 text-primary" />
            </div>
            <div className="flex-1">
              <p className="font-medium">Pipeline: {pipeline.name}</p>
              <p className="text-sm text-muted-foreground">
                {tableCount} table{tableCount !== 1 ? "s" : ""} replicating
              </p>
            </div>
            <Badge variant="default" className="bg-green-600">Running</Badge>
          </div>
        </div>
      </div>

      {/* Next Steps */}
      <div className="space-y-4">
        <h3 className="font-semibold">What&apos;s next?</h3>
        <div className="grid gap-3">
          <Button
            size="lg"
            onClick={onGoToQuery}
            className="w-full justify-between gap-2"
          >
            <span className="flex items-center gap-2">
              <Search className="h-5 w-5" />
              Query Your Data
            </span>
            <ArrowRight className="h-4 w-4" />
          </Button>
          <p className="text-sm text-muted-foreground text-center">
            Use SQL to explore your replicated data in the Iceberg data lake
          </p>
        </div>
      </div>

      {/* Other Actions */}
      <div className="flex flex-col sm:flex-row gap-3">
        <Button
          variant="outline"
          onClick={onViewPipeline}
          className="flex-1 gap-2"
        >
          <Database className="h-4 w-4" />
          View Pipeline Details
        </Button>
        <Button
          variant="outline"
          onClick={onCreateAnother}
          className="flex-1 gap-2"
        >
          <Plus className="h-4 w-4" />
          Add Another Source
        </Button>
      </div>

      {/* Tip */}
      <div className="text-center text-sm text-muted-foreground">
        <p>
          Tip: Your data will continue to sync automatically. Check the Pipelines
          page to monitor progress and metrics.
        </p>
      </div>
    </div>
  )
}
