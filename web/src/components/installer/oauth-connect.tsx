"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { useStartOAuthFlow, useHasCredentials } from "@/lib/hooks/use-oauth"
import { Loader2, Check, ExternalLink } from "lucide-react"

interface OAuthConnectProps {
  provider: string
  providerName: string
  oauthSupported: boolean
  onManualEntry?: () => void
}

export function OAuthConnect({
  provider,
  providerName,
  oauthSupported,
  onManualEntry,
}: OAuthConnectProps) {
  const { startFlow, isLoading, error } = useStartOAuthFlow()
  const { hasCredentials, isLoading: checkingCredentials } = useHasCredentials(provider)
  const [localError, setLocalError] = useState<string | null>(null)

  const handleOAuthConnect = async () => {
    setLocalError(null)
    try {
      await startFlow(provider)
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Failed to start OAuth flow")
    }
  }

  if (checkingCredentials) {
    return (
      <div className="flex items-center gap-2 text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>Checking credentials...</span>
      </div>
    )
  }

  if (hasCredentials) {
    return (
      <div className="flex items-center gap-2">
        <Badge variant="outline" className="text-green-600 border-green-600">
          <Check className="h-3 w-3 mr-1" />
          Connected
        </Badge>
        <span className="text-sm text-muted-foreground">
          {providerName} credentials stored
        </span>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-col sm:flex-row gap-2">
        {oauthSupported && (
          <Button
            onClick={handleOAuthConnect}
            disabled={isLoading}
            className="flex items-center gap-2"
          >
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <ExternalLink className="h-4 w-4" />
            )}
            Connect with {providerName}
          </Button>
        )}

        <Button
          variant={oauthSupported ? "outline" : "default"}
          onClick={onManualEntry}
          className="flex items-center gap-2"
        >
          Enter API Key Manually
        </Button>
      </div>

      {(error || localError) && (
        <p className="text-sm text-destructive">
          {localError || (error instanceof Error ? error.message : "An error occurred")}
        </p>
      )}

      {oauthSupported && (
        <p className="text-xs text-muted-foreground">
          OAuth allows secure access without sharing your API keys directly.
        </p>
      )}
    </div>
  )
}
