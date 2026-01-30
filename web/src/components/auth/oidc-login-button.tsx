"use client"

import { Button } from "@/components/ui/button"
import { Loader2 } from "lucide-react"
import { useStartOIDCFlow } from "@/lib/hooks/use-oidc"
import type { OIDCProviderSummary } from "@/lib/api/types"
import { cn } from "@/lib/utils"

// Provider brand colors for styling
const providerStyles: Record<string, { bg: string; hover: string; text: string }> = {
  google: {
    bg: "bg-white",
    hover: "hover:bg-gray-50",
    text: "text-gray-700",
  },
  okta: {
    bg: "bg-[#007DC1]",
    hover: "hover:bg-[#006BA1]",
    text: "text-white",
  },
  azure_ad: {
    bg: "bg-[#0078D4]",
    hover: "hover:bg-[#006CBC]",
    text: "text-white",
  },
  auth0: {
    bg: "bg-[#EB5424]",
    hover: "hover:bg-[#D4421A]",
    text: "text-white",
  },
  generic: {
    bg: "bg-gray-700",
    hover: "hover:bg-gray-600",
    text: "text-white",
  },
}

// Simple SVG icons for providers
function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
    </svg>
  )
}

function MicrosoftIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M11.4 24H0V12.6h11.4V24z" fill="#00A4EF"/>
      <path d="M24 24H12.6V12.6H24V24z" fill="#FFB900"/>
      <path d="M11.4 11.4H0V0h11.4v11.4z" fill="#F25022"/>
      <path d="M24 11.4H12.6V0H24v11.4z" fill="#7FBA00"/>
    </svg>
  )
}

function OktaIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0C5.373 0 0 5.373 0 12s5.373 12 12 12 12-5.373 12-12S18.627 0 12 0zm0 18.75c-3.728 0-6.75-3.022-6.75-6.75S8.272 5.25 12 5.25s6.75 3.022 6.75 6.75-3.022 6.75-6.75 6.75z" fill="#fff"/>
    </svg>
  )
}

function Auth0Icon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M17.664 14.58l2.16-6.66h-8.49l2.16 6.66-4.335 3.15L12 24l2.835-6.27 2.829 2.055V14.58zM6.336 14.58l-2.16-6.66h8.49l-2.16 6.66 4.335 3.15L12 24l-2.835-6.27-2.829 2.055V14.58zM12 0l4.335 3.15H7.665L12 0z" fill="#fff"/>
    </svg>
  )
}

function KeyIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4"/>
    </svg>
  )
}

function ProviderIcon({
  providerType,
  className
}: {
  providerType: OIDCProviderSummary["provider_type"]
  className?: string
}) {
  switch (providerType) {
    case "google":
      return <GoogleIcon className={className} />
    case "okta":
      return <OktaIcon className={className} />
    case "azure_ad":
      return <MicrosoftIcon className={className} />
    case "auth0":
      return <Auth0Icon className={className} />
    default:
      return <KeyIcon className={className} />
  }
}

interface OIDCLoginButtonProps {
  provider: OIDCProviderSummary
  className?: string
  variant?: "default" | "outline"
}

export function OIDCLoginButton({
  provider,
  className,
  variant = "default"
}: OIDCLoginButtonProps) {
  const { startFlow, isLoading } = useStartOIDCFlow()

  const styles = providerStyles[provider.provider_type] || providerStyles.generic

  const handleClick = async () => {
    await startFlow(provider.name)
  }

  if (variant === "outline") {
    return (
      <Button
        variant="outline"
        className={cn("w-full justify-start gap-3", className)}
        onClick={handleClick}
        disabled={isLoading}
      >
        {isLoading ? (
          <Loader2 className="h-5 w-5 animate-spin" />
        ) : (
          <ProviderIcon providerType={provider.provider_type} className="h-5 w-5" />
        )}
        <span>Sign in with {provider.display_name}</span>
      </Button>
    )
  }

  return (
    <Button
      className={cn(
        "w-full justify-start gap-3 border",
        styles.bg,
        styles.hover,
        styles.text,
        className
      )}
      onClick={handleClick}
      disabled={isLoading}
    >
      {isLoading ? (
        <Loader2 className="h-5 w-5 animate-spin" />
      ) : (
        <ProviderIcon providerType={provider.provider_type} className="h-5 w-5" />
      )}
      <span>Sign in with {provider.display_name}</span>
    </Button>
  )
}
