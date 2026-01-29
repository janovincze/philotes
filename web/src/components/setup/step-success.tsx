"use client"

import Link from "next/link"
import { Button } from "@/components/ui/button"
import { CheckCircle2, ArrowRight, Plus, ExternalLink } from "lucide-react"

interface StepSuccessProps {
  pipelineName: string
  pipelineId: string
  onViewPipeline: () => void
  onCreateAnother: () => void
}

export function StepSuccess({
  pipelineName,
  onViewPipeline,
  onCreateAnother,
}: StepSuccessProps) {
  return (
    <div className="space-y-8 py-8">
      {/* Success Icon */}
      <div className="flex justify-center">
        <div className="rounded-full bg-green-100 dark:bg-green-900/30 p-6">
          <CheckCircle2 className="h-16 w-16 text-green-600 dark:text-green-400" />
        </div>
      </div>

      {/* Message */}
      <div className="text-center space-y-2">
        <h2 className="text-2xl font-bold text-green-700 dark:text-green-300">
          Pipeline Created Successfully!
        </h2>
        <p className="text-muted-foreground max-w-md mx-auto">
          Your pipeline <span className="font-medium">{pipelineName}</span> has been
          created and is now running. Data will start flowing to your Iceberg tables
          shortly.
        </p>
      </div>

      {/* What's happening */}
      <div className="bg-muted/50 rounded-lg p-4 max-w-md mx-auto">
        <h3 className="font-medium mb-2">What&apos;s happening now:</h3>
        <ul className="text-sm text-muted-foreground space-y-1">
          <li>1. Connecting to your PostgreSQL database</li>
          <li>2. Creating replication slot for CDC</li>
          <li>3. Performing initial table snapshots</li>
          <li>4. Streaming changes to Iceberg in real-time</li>
        </ul>
      </div>

      {/* Actions */}
      <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
        <Button size="lg" onClick={onViewPipeline}>
          View Pipeline
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
        <Button variant="outline" size="lg" onClick={onCreateAnother}>
          <Plus className="mr-2 h-4 w-4" />
          Create Another
        </Button>
      </div>

      {/* Next steps */}
      <div className="text-center">
        <h3 className="font-medium mb-2">Next steps:</h3>
        <div className="flex flex-wrap justify-center gap-4 text-sm">
          <Link
            href="/pipelines"
            className="flex items-center gap-1 text-primary hover:underline"
          >
            Monitor all pipelines
            <ExternalLink className="h-3 w-3" />
          </Link>
          <Link
            href="/scaling"
            className="flex items-center gap-1 text-primary hover:underline"
          >
            Configure auto-scaling
            <ExternalLink className="h-3 w-3" />
          </Link>
          <Link
            href="/sources"
            className="flex items-center gap-1 text-primary hover:underline"
          >
            Manage sources
            <ExternalLink className="h-3 w-3" />
          </Link>
        </div>
      </div>
    </div>
  )
}
