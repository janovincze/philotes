"use client"

import { useState, useCallback } from "react"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { ArrowLeft, ArrowRight, Database, CheckCircle2 } from "lucide-react"
import { StepConnect } from "@/components/setup/step-connect"
import type { Source } from "@/lib/api/types"
import type { SourceFormData } from "@/components/setup/setup-wizard"

const initialSourceFormData: SourceFormData = {
  name: "",
  host: "",
  port: 5432,
  database_name: "",
  username: "",
  password: "",
  ssl_mode: "prefer",
}

interface StepSourceWrapperProps {
  source: Source | null
  onSourceCreated: (source: Source | null) => void
  onNext: (data?: Record<string, unknown>) => void
  onBack: () => void
}

export function StepSourceWrapper({
  source,
  onSourceCreated,
  onNext,
  onBack,
}: StepSourceWrapperProps) {
  const [formData, setFormData] = useState<SourceFormData>(initialSourceFormData)
  const [connectionTested, setConnectionTested] = useState(false)

  const updateFormData = useCallback((data: Partial<SourceFormData>) => {
    setFormData((prev) => ({ ...prev, ...data }))
  }, [])

  const handleSourceCreated = useCallback(
    (newSource: Source) => {
      onSourceCreated(newSource)
    },
    [onSourceCreated]
  )

  const handleNext = useCallback(() => {
    onNext({
      source_id: source?.id,
      source_name: source?.name,
    })
  }, [source, onNext])

  // If source already created, show summary
  if (source) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Connect Source Database</h2>
          <p className="text-muted-foreground mt-2">
            Connect your PostgreSQL database to start capturing changes.
          </p>
        </div>

        <Alert className="bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-800">
          <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
          <AlertTitle>Source connected</AlertTitle>
          <AlertDescription>
            <span className="font-medium">{source.name}</span> is connected and ready.
          </AlertDescription>
        </Alert>

        <div className="rounded-lg border p-4">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Database className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="font-medium">{source.name}</p>
              <p className="text-sm text-muted-foreground">
                {source.host}:{source.port}/{source.database_name}
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

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">Connect Source Database</h2>
        <p className="text-muted-foreground mt-2">
          Connect your PostgreSQL database to start capturing changes. Philotes will use logical
          replication to stream changes to your data lake.
        </p>
      </div>

      <StepConnect
        formData={formData}
        onFormDataChange={updateFormData}
        source={source}
        onSourceCreated={handleSourceCreated}
        connectionTested={connectionTested}
        onConnectionTested={setConnectionTested}
        onNext={handleNext}
        onBack={onBack}
      />
    </div>
  )
}
