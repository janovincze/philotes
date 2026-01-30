"use client"

import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { ArrowLeft, ArrowRight, SkipForward, Bell, Info, Slack, Mail } from "lucide-react"

interface StepAlertConfigProps {
  onNext: (data?: Record<string, unknown>) => void
  onBack: () => void
  onSkip: () => void
  onConfigured: (configured: boolean) => void
}

export function StepAlertConfig({ onNext, onBack, onSkip, onConfigured }: StepAlertConfigProps) {
  const handleConfigure = () => {
    onConfigured(true)
    onNext({ alerts_configured: true })
  }

  const handleSkip = () => {
    onSkip()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">Configure Alerts</h2>
        <p className="text-muted-foreground mt-2">
          Set up notifications to stay informed about pipeline health and data issues.
        </p>
      </div>

      <Alert>
        <Info className="h-4 w-4" />
        <AlertTitle>Optional Step</AlertTitle>
        <AlertDescription>
          Alert configuration is optional. You can skip this step and configure alerts later from
          the Alerts page.
        </AlertDescription>
      </Alert>

      <div className="space-y-4">
        {/* Slack Option */}
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="p-3 bg-[#4A154B]/10 rounded-lg">
              <Slack className="h-5 w-5 text-[#4A154B]" />
            </div>
            <div>
              <h3 className="font-medium">Slack Notifications</h3>
              <p className="text-sm text-muted-foreground">
                Receive alerts in your Slack workspace
              </p>
            </div>
          </div>
          <p className="text-sm text-muted-foreground">
            Connect to Slack to receive real-time notifications about pipeline status, errors, and
            data lag warnings.
          </p>
        </div>

        {/* Email Option */}
        <div className="rounded-lg border p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="p-3 bg-primary/10 rounded-lg">
              <Mail className="h-5 w-5 text-primary" />
            </div>
            <div>
              <h3 className="font-medium">Email Notifications</h3>
              <p className="text-sm text-muted-foreground">Receive alerts via email</p>
            </div>
          </div>
          <p className="text-sm text-muted-foreground">
            Configure email recipients for critical alerts and daily summary reports.
          </p>
        </div>

        {/* Default Alerts Info */}
        <div className="rounded-lg border p-4 space-y-3 bg-muted/50">
          <div className="flex items-center gap-3">
            <div className="p-3 bg-primary/10 rounded-lg">
              <Bell className="h-5 w-5 text-primary" />
            </div>
            <div>
              <h3 className="font-medium">Default Alert Rules</h3>
              <p className="text-sm text-muted-foreground">
                The following alerts will be configured automatically
              </p>
            </div>
          </div>
          <ul className="text-sm text-muted-foreground space-y-1 ml-14">
            <li>• Pipeline lag exceeds 5 minutes</li>
            <li>• Error rate exceeds 10 errors/minute</li>
            <li>• Pipeline stops unexpectedly</li>
            <li>• Connection to source lost</li>
          </ul>
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
