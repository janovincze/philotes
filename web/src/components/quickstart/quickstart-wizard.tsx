"use client"

import { useState, useCallback } from "react"
import { useRouter } from "next/navigation"
import { Card, CardContent } from "@/components/ui/card"
import { WizardProgress, type WizardStep } from "./wizard-progress"
import { StepWelcome } from "./step-welcome"
import { StepConnect } from "@/components/setup/step-connect"
import { StepTables } from "@/components/setup/step-tables"
import { StepVerify } from "./step-verify"
import { StepComplete } from "./step-complete"
import type { Source, TableInfo, Pipeline } from "@/lib/api/types"

const WIZARD_STEPS: WizardStep[] = [
  { id: 1, title: "Welcome" },
  { id: 2, title: "Connect" },
  { id: 3, title: "Tables" },
  { id: 4, title: "Verify" },
  { id: 5, title: "Complete" },
]

export interface SourceFormData {
  name: string
  host: string
  port: number
  database_name: string
  username: string
  password: string
  ssl_mode: string
}

export interface QuickstartState {
  currentStep: number
  sourceFormData: SourceFormData
  source: Source | null
  connectionTested: boolean
  availableTables: TableInfo[]
  selectedTables: string[]
  pipeline: Pipeline | null
}

const initialSourceFormData: SourceFormData = {
  name: "",
  host: "",
  port: 5432,
  database_name: "",
  username: "",
  password: "",
  ssl_mode: "prefer",
}

export function QuickstartWizard() {
  const router = useRouter()
  const [state, setState] = useState<QuickstartState>({
    currentStep: 1,
    sourceFormData: initialSourceFormData,
    source: null,
    connectionTested: false,
    availableTables: [],
    selectedTables: [],
    pipeline: null,
  })

  const nextStep = useCallback(() => {
    setState((prev) => ({ ...prev, currentStep: Math.min(prev.currentStep + 1, 5) }))
  }, [])

  const prevStep = useCallback(() => {
    setState((prev) => ({ ...prev, currentStep: Math.max(prev.currentStep - 1, 1) }))
  }, [])

  const updateSourceFormData = useCallback((data: Partial<SourceFormData>) => {
    setState((prev) => ({
      ...prev,
      sourceFormData: { ...prev.sourceFormData, ...data },
    }))
  }, [])

  const setSource = useCallback((source: Source) => {
    setState((prev) => ({ ...prev, source }))
  }, [])

  const setConnectionTested = useCallback((tested: boolean) => {
    setState((prev) => ({ ...prev, connectionTested: tested }))
  }, [])

  const setAvailableTables = useCallback((tables: TableInfo[]) => {
    // Auto-select all tables by default
    const allTableNames = tables.map((t) => `${t.schema}.${t.name}`)
    setState((prev) => ({
      ...prev,
      availableTables: tables,
      selectedTables: allTableNames,
    }))
  }, [])

  const setSelectedTables = useCallback((tables: string[]) => {
    setState((prev) => ({ ...prev, selectedTables: tables }))
  }, [])

  const setPipeline = useCallback((pipeline: Pipeline) => {
    setState((prev) => ({ ...prev, pipeline }))
  }, [])

  const handleViewPipeline = useCallback(() => {
    if (state.pipeline) {
      router.push(`/pipelines/${state.pipeline.id}`)
    }
  }, [state.pipeline, router])

  const handleGoToQuery = useCallback(() => {
    router.push("/query")
  }, [router])

  const handleCreateAnother = useCallback(() => {
    setState({
      currentStep: 1,
      sourceFormData: initialSourceFormData,
      source: null,
      connectionTested: false,
      availableTables: [],
      selectedTables: [],
      pipeline: null,
    })
  }, [])

  const renderStep = () => {
    switch (state.currentStep) {
      case 1:
        return <StepWelcome onNext={nextStep} />
      case 2:
        return (
          <StepConnect
            formData={state.sourceFormData}
            onFormDataChange={updateSourceFormData}
            source={state.source}
            onSourceCreated={setSource}
            connectionTested={state.connectionTested}
            onConnectionTested={setConnectionTested}
            onNext={nextStep}
            onBack={prevStep}
          />
        )
      case 3:
        return (
          <StepTables
            sourceId={state.source?.id ?? ""}
            availableTables={state.availableTables}
            onTablesLoaded={setAvailableTables}
            selectedTables={state.selectedTables}
            onSelectedTablesChange={setSelectedTables}
            onNext={nextStep}
            onBack={prevStep}
          />
        )
      case 4:
        return (
          <StepVerify
            source={state.source!}
            sourceName={state.sourceFormData.name}
            selectedTables={state.selectedTables}
            onPipelineCreated={setPipeline}
            onNext={nextStep}
            onBack={prevStep}
          />
        )
      case 5:
        return (
          <StepComplete
            pipeline={state.pipeline!}
            sourceName={state.sourceFormData.name}
            tableCount={state.selectedTables.length}
            onViewPipeline={handleViewPipeline}
            onGoToQuery={handleGoToQuery}
            onCreateAnother={handleCreateAnother}
          />
        )
      default:
        return null
    }
  }

  return (
    <div className="space-y-8">
      <div className="text-center">
        <h1 className="text-3xl font-bold">Quick Start</h1>
        <p className="text-muted-foreground mt-2">
          Get your data flowing in 5 simple steps
        </p>
      </div>
      <WizardProgress steps={WIZARD_STEPS} currentStep={state.currentStep} />
      <Card>
        <CardContent className="pt-6">{renderStep()}</CardContent>
      </Card>
    </div>
  )
}
