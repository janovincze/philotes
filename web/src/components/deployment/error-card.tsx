"use client"

import { AlertCircle, ExternalLink, RefreshCw, ChevronDown, ChevronUp } from "lucide-react"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { cn } from "@/lib/utils"
import type { StepError } from "@/lib/api/types"

interface ErrorCardProps {
  error: StepError
  stepName?: string
  onRetry?: () => void
  isRetrying?: boolean
  className?: string
}

export function ErrorCard({
  error,
  stepName,
  onRetry,
  isRetrying = false,
  className,
}: ErrorCardProps) {
  const [showDetails, setShowDetails] = useState(false)

  return (
    <Card className={cn("border-red-200 dark:border-red-900", className)}>
      <CardHeader className="pb-3">
        <div className="flex items-start gap-3">
          <div className="rounded-full bg-red-100 dark:bg-red-950 p-2">
            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
          </div>
          <div className="flex-1">
            <CardTitle className="text-red-600 dark:text-red-400 text-lg">
              {stepName ? `Failed: ${stepName}` : "Deployment Failed"}
            </CardTitle>
            <CardDescription className="text-red-500 dark:text-red-400/80 mt-1">
              {error.message}
            </CardDescription>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Suggestions */}
        {error.suggestions && error.suggestions.length > 0 && (
          <div>
            <h4 className="text-sm font-medium mb-2">Troubleshooting suggestions:</h4>
            <ul className="space-y-2">
              {error.suggestions.map((suggestion, index) => (
                <li
                  key={index}
                  className="flex items-start gap-2 text-sm text-muted-foreground"
                >
                  <span className="text-primary mt-0.5">•</span>
                  <span>{suggestion}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Error details (collapsible) */}
        {error.details && (
          <Collapsible open={showDetails} onOpenChange={setShowDetails}>
            <CollapsibleTrigger className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors">
              {showDetails ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
              <span>Technical details</span>
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="mt-2 rounded-md bg-muted p-3 font-mono text-xs overflow-x-auto">
                <pre className="whitespace-pre-wrap break-all">{error.details}</pre>
              </div>
            </CollapsibleContent>
          </Collapsible>
        )}

        {/* Error code */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Error code:</span>
          <code className="bg-muted px-1.5 py-0.5 rounded">{error.code}</code>
        </div>
      </CardContent>

      <CardFooter className="flex gap-3">
        {error.retryable && onRetry && (
          <Button onClick={onRetry} disabled={isRetrying} variant="default">
            {isRetrying ? (
              <>
                <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                Retrying...
              </>
            ) : (
              <>
                <RefreshCw className="mr-2 h-4 w-4" />
                Retry Deployment
              </>
            )}
          </Button>
        )}
        {error.docs_url && (
          <Button variant="outline" asChild>
            <a href={error.docs_url} target="_blank" rel="noopener noreferrer">
              <ExternalLink className="mr-2 h-4 w-4" />
              View Documentation
            </a>
          </Button>
        )}
      </CardFooter>
    </Card>
  )
}
