"use client"

import { useEnabledOIDCProviders } from "@/lib/hooks/use-oidc"
import { OIDCLoginButton } from "./oidc-login-button"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { AlertCircle } from "lucide-react"

interface OIDCLoginSectionProps {
  className?: string
  variant?: "default" | "outline"
  showDivider?: boolean
}

export function OIDCLoginSection({
  className,
  variant = "default",
  showDivider = true,
}: OIDCLoginSectionProps) {
  const { data, isLoading, error } = useEnabledOIDCProviders()

  // Don't render anything if no providers
  if (!isLoading && (!data?.providers || data.providers.length === 0)) {
    return null
  }

  if (isLoading) {
    return (
      <div className={className}>
        {showDivider && (
          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-background px-2 text-muted-foreground">
                Or continue with
              </span>
            </div>
          </div>
        )}
        <div className="space-y-3">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <Alert variant="destructive" className={className}>
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>
          Failed to load SSO providers. Please try again.
        </AlertDescription>
      </Alert>
    )
  }

  const providers = data?.providers || []

  return (
    <div className={className}>
      {showDivider && (
        <div className="relative my-6">
          <div className="absolute inset-0 flex items-center">
            <span className="w-full border-t" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">
              Or continue with
            </span>
          </div>
        </div>
      )}
      <div className="space-y-3">
        {providers.map((provider) => (
          <OIDCLoginButton
            key={provider.id}
            provider={provider}
            variant={variant}
          />
        ))}
      </div>
    </div>
  )
}
