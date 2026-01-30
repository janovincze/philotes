"use client"

import { useEffect, useRef } from "react"
import Link from "next/link"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  CheckCircle2,
  ArrowRight,
  LayoutDashboard,
  GitBranch,
  BookOpen,
  MessageCircle,
  Key,
  Copy,
} from "lucide-react"
import { toast } from "sonner"
import { clearOnboardingSession } from "@/lib/hooks/use-onboarding"
import confetti from "canvas-confetti"

interface StepCompleteProps {
  adminApiKey?: string
  sourceName?: string
  pipelineName?: string
  pipelineId?: string
  onComplete: () => void
}

export function StepComplete({
  adminApiKey,
  sourceName,
  pipelineName,
  pipelineId,
  onComplete,
}: StepCompleteProps) {
  // Use ref to track if confetti has been fired (runs once on mount)
  const confettiFiredRef = useRef(false)

  // Fire confetti on mount
  useEffect(() => {
    if (confettiFiredRef.current) return
    confettiFiredRef.current = true

    const duration = 3 * 1000
    const animationEnd = Date.now() + duration
    const defaults = { startVelocity: 30, spread: 360, ticks: 60, zIndex: 0 }

    function randomInRange(min: number, max: number) {
      return Math.random() * (max - min) + min
    }

    const interval = setInterval(function () {
      const timeLeft = animationEnd - Date.now()

      if (timeLeft <= 0) {
        clearInterval(interval)
        return
      }

      const particleCount = 50 * (timeLeft / duration)

      confetti({
        ...defaults,
        particleCount,
        origin: { x: randomInRange(0.1, 0.3), y: Math.random() - 0.2 },
      })
      confetti({
        ...defaults,
        particleCount,
        origin: { x: randomInRange(0.7, 0.9), y: Math.random() - 0.2 },
      })
    }, 250)

    return () => clearInterval(interval)
  }, [])

  const copyApiKey = () => {
    if (adminApiKey) {
      navigator.clipboard.writeText(adminApiKey)
      toast.success("API key copied", {
        description: "The API key has been copied to your clipboard.",
      })
    }
  }

  const handleComplete = () => {
    clearOnboardingSession()
    onComplete()
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30 mb-4">
          <CheckCircle2 className="h-8 w-8 text-green-600 dark:text-green-400" />
        </div>
        <h2 className="text-2xl font-semibold tracking-tight">Setup Complete!</h2>
        <p className="text-muted-foreground mt-2">
          Congratulations! Philotes is now configured and ready to use.
        </p>
      </div>

      {/* Summary */}
      <div className="rounded-lg border p-4 space-y-3">
        <h3 className="font-medium">Configuration Summary</h3>
        <ul className="space-y-2 text-sm">
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-green-600" />
            <span>Cluster health verified</span>
          </li>
          <li className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-green-600" />
            <span>Admin account created</span>
          </li>
          {sourceName && (
            <li className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span>
                Source connected: <span className="font-medium">{sourceName}</span>
              </span>
            </li>
          )}
          {pipelineName && (
            <li className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span>
                Pipeline created: <span className="font-medium">{pipelineName}</span>
              </span>
            </li>
          )}
        </ul>
      </div>

      {/* API Key reminder */}
      {adminApiKey && (
        <Alert>
          <Key className="h-4 w-4" />
          <AlertTitle>Remember to save your API key</AlertTitle>
          <AlertDescription className="mt-2">
            <div className="flex items-center gap-2">
              <code className="flex-1 p-2 bg-muted rounded text-xs font-mono break-all">
                {adminApiKey}
              </code>
              <Button variant="outline" size="icon" onClick={copyApiKey}>
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      {/* Quick Links */}
      <div className="grid gap-3 sm:grid-cols-2">
        <Link href="/dashboard" className="block">
          <div className="rounded-lg border p-4 hover:bg-muted/50 transition-colors">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary/10 rounded-lg">
                <LayoutDashboard className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="font-medium">Dashboard</p>
                <p className="text-sm text-muted-foreground">View pipeline metrics</p>
              </div>
            </div>
          </div>
        </Link>

        {pipelineId && (
          <Link href={`/pipelines/${pipelineId}`} className="block">
            <div className="rounded-lg border p-4 hover:bg-muted/50 transition-colors">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-primary/10 rounded-lg">
                  <GitBranch className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="font-medium">View Pipeline</p>
                  <p className="text-sm text-muted-foreground">Monitor your pipeline</p>
                </div>
              </div>
            </div>
          </Link>
        )}

        <a
          href="https://philotes.io/docs"
          target="_blank"
          rel="noopener noreferrer"
          className="block"
        >
          <div className="rounded-lg border p-4 hover:bg-muted/50 transition-colors">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary/10 rounded-lg">
                <BookOpen className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="font-medium">Documentation</p>
                <p className="text-sm text-muted-foreground">Learn more about Philotes</p>
              </div>
            </div>
          </div>
        </a>

        <a
          href="https://github.com/janovincze/philotes/discussions"
          target="_blank"
          rel="noopener noreferrer"
          className="block"
        >
          <div className="rounded-lg border p-4 hover:bg-muted/50 transition-colors">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary/10 rounded-lg">
                <MessageCircle className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="font-medium">Community</p>
                <p className="text-sm text-muted-foreground">Get help and share feedback</p>
              </div>
            </div>
          </div>
        </a>
      </div>

      <div className="flex justify-center pt-4">
        <Button size="lg" onClick={handleComplete}>
          Go to Dashboard
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
