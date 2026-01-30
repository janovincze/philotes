"use client"

import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { ArrowLeft, ArrowRight, SkipForward, Shield, Info } from "lucide-react"

interface StepSSOConfigProps {
  onNext: (data?: Record<string, unknown>) => void
  onBack: () => void
  onSkip: () => void
  onConfigured: (configured: boolean) => void
}

export function StepSSOConfig({ onNext, onBack, onSkip, onConfigured }: StepSSOConfigProps) {
  const handleConfigure = () => {
    onConfigured(true)
    onNext({ sso_configured: true })
  }

  const handleSkip = () => {
    onSkip()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">SSO Configuration</h2>
        <p className="text-muted-foreground mt-2">
          Configure Single Sign-On (SSO) to allow users to authenticate with your identity provider.
        </p>
      </div>

      <Alert>
        <Info className="h-4 w-4" />
        <AlertTitle>Optional Step</AlertTitle>
        <AlertDescription>
          SSO configuration is optional. You can skip this step and configure it later from Settings.
        </AlertDescription>
      </Alert>

      <div className="rounded-lg border p-6 space-y-4">
        <div className="flex items-center gap-3">
          <div className="p-3 bg-primary/10 rounded-lg">
            <Shield className="h-6 w-6 text-primary" />
          </div>
          <div>
            <h3 className="font-medium">OIDC / OAuth 2.0</h3>
            <p className="text-sm text-muted-foreground">
              Connect to providers like Okta, Auth0, Google Workspace, or Azure AD
            </p>
          </div>
        </div>

        <div className="text-sm text-muted-foreground">
          <p>SSO configuration will be available in Settings → Authentication.</p>
          <p className="mt-2">
            For now, users can authenticate using email and password created in the previous step.
          </p>
        </div>
      </div>

      <div className="flex justify-between pt-4">
        <Button variant="outline" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          <Button variant="ghost" onClick={handleSkip}>
            <SkipForward className="mr-2 h-4 w-4" />
            Skip for Now
          </Button>
          <Button onClick={handleConfigure}>
            Configure Later
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
