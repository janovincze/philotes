"use client"

import { OnboardingWizard } from "@/components/onboarding"

export default function OnboardingPage() {
  return (
    <div className="container max-w-4xl py-8">
      <div className="text-center mb-8">
        <h1 className="text-3xl font-bold tracking-tight">Welcome to Philotes</h1>
        <p className="text-muted-foreground mt-2">
          Let&apos;s get you set up in just a few steps
        </p>
      </div>
      <OnboardingWizard />
    </div>
  )
}
