"use client"

import { Button } from "@/components/ui/button"
import { Database, ArrowRight, GitBranch, Table, Zap } from "lucide-react"

interface StepWelcomeProps {
  onNext: () => void
}

export function StepWelcome({ onNext }: StepWelcomeProps) {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center space-y-4">
        <div className="flex justify-center">
          <div className="rounded-full bg-primary/10 p-4">
            <Zap className="h-12 w-12 text-primary" />
          </div>
        </div>
        <h2 className="text-2xl font-bold">Welcome to Philotes</h2>
        <p className="text-muted-foreground max-w-md mx-auto">
          Set up your first CDC pipeline in just a few minutes. We&apos;ll guide you
          through connecting your database and selecting tables to replicate.
        </p>
      </div>

      {/* What you'll do */}
      <div className="grid gap-4 sm:grid-cols-3 max-w-2xl mx-auto">
        <div className="flex flex-col items-center text-center p-4 rounded-lg border bg-card">
          <Database className="h-8 w-8 text-primary mb-2" />
          <h3 className="font-medium">Connect Database</h3>
          <p className="text-sm text-muted-foreground">
            Enter your PostgreSQL credentials
          </p>
        </div>
        <div className="flex flex-col items-center text-center p-4 rounded-lg border bg-card">
          <Table className="h-8 w-8 text-primary mb-2" />
          <h3 className="font-medium">Select Tables</h3>
          <p className="text-sm text-muted-foreground">
            Choose which tables to replicate
          </p>
        </div>
        <div className="flex flex-col items-center text-center p-4 rounded-lg border bg-card">
          <GitBranch className="h-8 w-8 text-primary mb-2" />
          <h3 className="font-medium">Start Pipeline</h3>
          <p className="text-sm text-muted-foreground">
            Watch your data flow to Iceberg
          </p>
        </div>
      </div>

      {/* What you'll need */}
      <div className="bg-muted/50 rounded-lg p-4 max-w-md mx-auto">
        <h3 className="font-medium mb-2">What you&apos;ll need:</h3>
        <ul className="text-sm text-muted-foreground space-y-1">
          <li>- PostgreSQL database hostname and port</li>
          <li>- Database username and password</li>
          <li>- User with SELECT and REPLICATION permissions</li>
        </ul>
      </div>

      {/* Actions */}
      <div className="flex justify-center">
        <Button size="lg" onClick={onNext}>
          Get Started
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
