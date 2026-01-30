"use client"

import { useState, useCallback, useMemo } from "react"
import { useRouter } from "next/navigation"
import { Card, CardContent } from "@/components/ui/card"
import { OnboardingProgress, type OnboardingStepInfo } from "./onboarding-progress"
import { StepClusterHealth } from "./step-cluster-health"
import { StepAdminUser } from "./step-admin-user"
import { StepSSOConfig } from "./step-sso-config"
import { StepSourceWrapper } from "./step-source-wrapper"
import { StepPipelineWrapper } from "./step-pipeline-wrapper"
import { StepDataVerification } from "./step-data-verification"
import { StepAlertConfig } from "./step-alert-config"
import { StepComplete } from "./step-complete"
import {
  useOnboardingProgress,
  useSaveOnboardingProgress,
  getOnboardingSessionId,
} from "@/lib/hooks/use-onboarding"
import type { Source, Pipeline } from "@/lib/api/types"

const ONBOARDING_STEPS: OnboardingStepInfo[] = [
  { id: 1, title: "Health Check" },
  { id: 2, title: "Admin User" },
  { id: 3, title: "SSO Setup", optional: true },
  { id: 4, title: "Source DB" },
  { id: 5, title: "Pipeline" },
  { id: 6, title: "Verify Data" },
  { id: 7, title: "Alerts", optional: true },
]

export interface OnboardingState {
  currentStep: number
  completedSteps: number[]
  skippedSteps: number[]
  stepData: Record<string, unknown>
  stepStartTime: number
  // Step-specific data
  adminCreated: boolean
  adminApiKey?: string
  ssoConfigured: boolean
  source: Source | null
  pipeline: Pipeline | null
  dataVerified: boolean
  alertsConfigured: boolean
  // Track if initial state has been applied
  initialStateApplied: boolean
}

export function OnboardingWizard() {
  const router = useRouter()
  const sessionId = getOnboardingSessionId()
  const { data: progressData, isLoading: isLoadingProgress } = useOnboardingProgress(sessionId)
  const saveProgressMutation = useSaveOnboardingProgress()

  // Compute initial state based on progress data
  const computedInitialState = useMemo(() => {
    const base = {
      currentStep: 1,
      completedSteps: [] as number[],
      skippedSteps: [] as number[],
      stepData: {} as Record<string, unknown>,
      stepStartTime: 0, // Will be set on first interaction
      adminCreated: false,
      ssoConfigured: false,
      source: null as Source | null,
      pipeline: null as Pipeline | null,
      dataVerified: false,
      alertsConfigured: false,
      initialStateApplied: false,
    }

    if (progressData?.progress) {
      const progress = progressData.progress
      return {
        ...base,
        currentStep: progress.current_step,
        completedSteps: progress.completed_steps,
        stepData: progress.step_data as Record<string, unknown>,
        skippedSteps: (progress.metrics?.steps_skipped || []) as number[],
        initialStateApplied: true,
      }
    }

    return base
  }, [progressData])

  // Initialize state - will only use computedInitialState on first render
  const [state, setState] = useState<OnboardingState>(() => computedInitialState)

  // Determine if we're ready to show content
  const isReady = useMemo(() => {
    // Ready if not loading and either:
    // 1. We have progress data and it's been applied to state
    // 2. We don't have progress data (new user)
    if (isLoadingProgress) return false
    if (progressData?.progress) {
      return state.initialStateApplied || state.currentStep === computedInitialState.currentStep
    }
    return true
  }, [isLoadingProgress, progressData, state.initialStateApplied, state.currentStep, computedInitialState.currentStep])

  // Handle state update when progress data arrives after initial render
  // This is a controlled way to sync external data - using the query's onSuccess would be cleaner
  // but this works with the current hook structure
  const syncedState = useMemo(() => {
    if (!state.initialStateApplied && progressData?.progress && !isLoadingProgress) {
      const progress = progressData.progress
      return {
        ...state,
        currentStep: progress.current_step,
        completedSteps: progress.completed_steps,
        stepData: progress.step_data as Record<string, unknown>,
        skippedSteps: (progress.metrics?.steps_skipped || []) as number[],
        initialStateApplied: true,
      }
    }
    return state
  }, [state, progressData, isLoadingProgress])

  // Use syncedState for all reads, but keep setState for updates
  const currentState = syncedState

  // Save progress to backend
  const saveProgress = useCallback(
    async (
      newStep: number,
      newCompletedSteps: number[],
      additionalData?: Record<string, unknown>,
      stepSkipped?: number
    ) => {
      const stepTimeMs = Date.now() - currentState.stepStartTime

      try {
        await saveProgressMutation.mutateAsync({
          session_id: sessionId,
          current_step: newStep,
          completed_steps: newCompletedSteps,
          step_data: additionalData,
          step_skipped: stepSkipped,
          step_time_ms: stepTimeMs,
        })
      } catch (error) {
        console.error("Failed to save onboarding progress:", error)
      }
    },
    [sessionId, saveProgressMutation, currentState.stepStartTime]
  )

  const nextStep = useCallback(
    async (additionalData?: Record<string, unknown>) => {
      const newCompletedSteps = [...currentState.completedSteps]
      if (!newCompletedSteps.includes(currentState.currentStep)) {
        newCompletedSteps.push(currentState.currentStep)
      }

      const newStep = Math.min(currentState.currentStep + 1, 7)

      await saveProgress(newStep, newCompletedSteps, additionalData)

      setState((prev) => ({
        ...prev,
        currentStep: newStep,
        completedSteps: newCompletedSteps,
        stepData: { ...prev.stepData, ...additionalData },
        stepStartTime: Date.now(),
        initialStateApplied: true,
      }))
    },
    [currentState.currentStep, currentState.completedSteps, saveProgress]
  )

  const skipStep = useCallback(async () => {
    const newSkippedSteps = [...currentState.skippedSteps, currentState.currentStep]
    const newStep = Math.min(currentState.currentStep + 1, 7)

    await saveProgress(newStep, currentState.completedSteps, undefined, currentState.currentStep)

    setState((prev) => ({
      ...prev,
      currentStep: newStep,
      skippedSteps: newSkippedSteps,
      stepStartTime: Date.now(),
      initialStateApplied: true,
    }))
  }, [currentState.currentStep, currentState.completedSteps, currentState.skippedSteps, saveProgress])

  const prevStep = useCallback(() => {
    setState((prev) => ({
      ...prev,
      currentStep: Math.max(prev.currentStep - 1, 1),
      stepStartTime: Date.now(),
    }))
  }, [])

  const setAdminCreated = useCallback((created: boolean, apiKey?: string) => {
    setState((prev) => ({
      ...prev,
      adminCreated: created,
      adminApiKey: apiKey,
    }))
  }, [])

  const setSSOConfigured = useCallback((configured: boolean) => {
    setState((prev) => ({ ...prev, ssoConfigured: configured }))
  }, [])

  const setSource = useCallback((source: Source | null) => {
    setState((prev) => ({ ...prev, source }))
  }, [])

  const setPipeline = useCallback((pipeline: Pipeline | null) => {
    setState((prev) => ({ ...prev, pipeline }))
  }, [])

  const setDataVerified = useCallback((verified: boolean) => {
    setState((prev) => ({ ...prev, dataVerified: verified }))
  }, [])

  const setAlertsConfigured = useCallback((configured: boolean) => {
    setState((prev) => ({ ...prev, alertsConfigured: configured }))
  }, [])

  const handleComplete = useCallback(() => {
    router.push("/dashboard")
  }, [router])

  // Show loading state while initializing
  if (!isReady) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto" />
          <p className="mt-4 text-muted-foreground">Loading onboarding progress...</p>
        </div>
      </div>
    )
  }

  const renderStep = () => {
    switch (currentState.currentStep) {
      case 1:
        return <StepClusterHealth onNext={nextStep} />
      case 2:
        return (
          <StepAdminUser
            onNext={nextStep}
            onBack={prevStep}
            onAdminCreated={setAdminCreated}
          />
        )
      case 3:
        return (
          <StepSSOConfig
            onNext={nextStep}
            onBack={prevStep}
            onSkip={skipStep}
            onConfigured={setSSOConfigured}
          />
        )
      case 4:
        return (
          <StepSourceWrapper
            source={currentState.source}
            onSourceCreated={setSource}
            onNext={nextStep}
            onBack={prevStep}
          />
        )
      case 5:
        return (
          <StepPipelineWrapper
            source={currentState.source}
            pipeline={currentState.pipeline}
            onPipelineCreated={setPipeline}
            onNext={nextStep}
            onBack={prevStep}
          />
        )
      case 6:
        return (
          <StepDataVerification
            pipeline={currentState.pipeline}
            onNext={nextStep}
            onBack={prevStep}
            onDataVerified={setDataVerified}
          />
        )
      case 7:
        return (
          <StepAlertConfig
            onNext={nextStep}
            onBack={prevStep}
            onSkip={skipStep}
            onConfigured={setAlertsConfigured}
          />
        )
      default:
        return (
          <StepComplete
            adminApiKey={currentState.adminApiKey}
            sourceName={currentState.source?.name}
            pipelineName={currentState.pipeline?.name}
            pipelineId={currentState.pipeline?.id}
            onComplete={handleComplete}
          />
        )
    }
  }

  // Show completion screen after step 7
  if (currentState.currentStep > 7) {
    return (
      <div className="space-y-8">
        <OnboardingProgress
          steps={ONBOARDING_STEPS}
          currentStep={8}
          completedSteps={currentState.completedSteps}
          skippedSteps={currentState.skippedSteps}
        />
        <Card>
          <CardContent className="pt-6">
            <StepComplete
              adminApiKey={currentState.adminApiKey}
              sourceName={currentState.source?.name}
              pipelineName={currentState.pipeline?.name}
              pipelineId={currentState.pipeline?.id}
              onComplete={handleComplete}
            />
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <OnboardingProgress
        steps={ONBOARDING_STEPS}
        currentStep={currentState.currentStep}
        completedSteps={currentState.completedSteps}
        skippedSteps={currentState.skippedSteps}
      />
      <Card>
        <CardContent className="pt-6">{renderStep()}</CardContent>
      </Card>
    </div>
  )
}
