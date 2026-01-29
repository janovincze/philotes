"use client"

import { useState, useCallback } from "react"
import { useRouter } from "next/navigation"
import { Card, CardContent } from "@/components/ui/card"
import { WizardProgress, type WizardStep } from "./wizard-progress"
import { StepWelcome } from "./step-welcome"
import { StepConnect } from "./step-connect"
import { StepTables } from "./step-tables"
import { StepConfigure } from "./step-configure"
import { StepReview } from "./step-review"
import { StepSuccess } from "./step-success"
import type { Source, TableInfo } from "@/lib/api/types"

const WIZARD_STEPS: WizardStep[] = [
  { id: 1, title: "Welcome" },
  { id: 2, title: "Connect" },
  { id: 3, title: "Tables" },
  { id: 4, title: "Configure" },
  { id: 5, title: "Review" },
  { id: 6, title: "Done" },
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

export interface WizardState {
  currentStep: number
  sourceFormData: SourceFormData
  source: Source | null
  connectionTested: boolean
  availableTables: TableInfo[]
  selectedTables: string[]
  pipelineName: string
  pipelineId: string | null
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

export function SetupWizard() {
  const router = useRouter()
  const [state, setState] = useState<WizardState>({
    currentStep: 1,
    sourceFormData: initialSourceFormData,
    source: null,
    connectionTested: false,
    availableTables: [],
    selectedTables: [],
    pipelineName: "",
    pipelineId: null,
  })

  // goToStep can be used for direct navigation if needed in the future
  // const goToStep = useCallback((step: number) => {
  //   setState((prev) => ({ ...prev, currentStep: step }))
  // }, [])

  const nextStep = useCallback(() => {
    setState((prev) => ({ ...prev, currentStep: Math.min(prev.currentStep + 1, 6) }))
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
    setState((prev) => ({ ...prev, availableTables: tables }))
  }, [])

  const setSelectedTables = useCallback((tables: string[]) => {
    setState((prev) => ({ ...prev, selectedTables: tables }))
  }, [])

  const setPipelineName = useCallback((name: string) => {
    setState((prev) => ({ ...prev, pipelineName: name }))
  }, [])

  const setPipelineId = useCallback((id: string) => {
    setState((prev) => ({ ...prev, pipelineId: id }))
  }, [])

  const handleViewPipeline = useCallback(() => {
    if (state.pipelineId) {
      router.push(`/pipelines/${state.pipelineId}`)
    }
  }, [state.pipelineId, router])

  const handleCreateAnother = useCallback(() => {
    setState({
      currentStep: 1,
      sourceFormData: initialSourceFormData,
      source: null,
      connectionTested: false,
      availableTables: [],
      selectedTables: [],
      pipelineName: "",
      pipelineId: null,
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
          <StepConfigure
            pipelineName={state.pipelineName}
            onPipelineNameChange={setPipelineName}
            sourceName={state.sourceFormData.name}
            selectedTablesCount={state.selectedTables.length}
            onNext={nextStep}
            onBack={prevStep}
          />
        )
      case 5:
        return (
          <StepReview
            sourceFormData={state.sourceFormData}
            source={state.source}
            selectedTables={state.selectedTables}
            pipelineName={state.pipelineName}
            onPipelineCreated={setPipelineId}
            onNext={nextStep}
            onBack={prevStep}
          />
        )
      case 6:
        return (
          <StepSuccess
            pipelineName={state.pipelineName}
            pipelineId={state.pipelineId ?? ""}
            onViewPipeline={handleViewPipeline}
            onCreateAnother={handleCreateAnother}
          />
        )
      default:
        return null
    }
  }

  return (
    <div className="space-y-8">
      <WizardProgress steps={WIZARD_STEPS} currentStep={state.currentStep} />
      <Card>
        <CardContent className="pt-6">
          {renderStep()}
        </CardContent>
      </Card>
    </div>
  )
}
