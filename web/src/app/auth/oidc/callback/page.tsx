"use client"

import { useEffect, useRef, Suspense, useState, useMemo } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Loader2, CheckCircle, XCircle } from "lucide-react"
import { useOIDCCallback } from "@/lib/hooks/use-oidc"

function OIDCCallbackContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const hasProcessed = useRef(false)
  const callback = useOIDCCallback()
  const [callbackResult, setCallbackResult] = useState<{
    status: "processing" | "success" | "error"
    message: string
  } | null>(null)

  // Parse search params from IdP callback
  const code = searchParams.get("code")
  const state = searchParams.get("state")
  const error = searchParams.get("error")
  const errorDescription = searchParams.get("error_description")

  // Compute initial state from URL params (synchronous)
  const initialState = useMemo(() => {
    if (error) {
      return {
        status: "error" as const,
        message: errorDescription || error || "Authentication failed",
      }
    }
    if (!code || !state) {
      return {
        status: "error" as const,
        message: "Missing required parameters (code or state)",
      }
    }
    return null // needs async processing
  }, [error, errorDescription, code, state])

  // Use initial state or callback result
  const currentState = callbackResult || initialState || { status: "processing" as const, message: "" }
  const status = currentState.status
  const message = currentState.message

  useEffect(() => {
    if (hasProcessed.current) return
    if (initialState) return // Already handled synchronously
    hasProcessed.current = true

    // Process the callback
    const processCallback = async () => {
      try {
        const response = await callback.mutateAsync({
          code: code!,
          state: state!,
        })

        if (response.success && response.token) {
          // Store the token
          if (typeof window !== "undefined") {
            localStorage.setItem("philotes_token", response.token)
            if (response.expires_at) {
              localStorage.setItem("philotes_token_expires_at", response.expires_at)
            }
            if (response.user) {
              localStorage.setItem("philotes_user", JSON.stringify(response.user))
            }
          }

          setCallbackResult({
            status: "success",
            message: "Authentication successful!",
          })

          // Redirect to dashboard after a short delay
          setTimeout(() => {
            const redirectTo = response.redirect_uri || "/dashboard"
            router.push(redirectTo)
          }, 1500)
        } else {
          setCallbackResult({
            status: "error",
            message: response.error || "Authentication failed",
          })
        }
      } catch (err) {
        setCallbackResult({
          status: "error",
          message: err instanceof Error ? err.message : "An unexpected error occurred",
        })
      }
    }

    processCallback()
  }, [code, state, initialState, callback, router])

  const handleRetry = () => {
    router.push("/login")
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle>
            {status === "processing" && "Signing you in..."}
            {status === "success" && "Authentication Successful"}
            {status === "error" && "Authentication Failed"}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-4">
          {status === "processing" && (
            <>
              <Loader2 className="h-12 w-12 animate-spin text-primary" />
              <p className="text-muted-foreground">
                Please wait while we verify your identity...
              </p>
            </>
          )}

          {status === "success" && (
            <>
              <CheckCircle className="h-12 w-12 text-green-500" />
              <p className="text-center">{message}</p>
              <p className="text-sm text-muted-foreground">
                Redirecting you to the dashboard...
              </p>
            </>
          )}

          {status === "error" && (
            <>
              <XCircle className="h-12 w-12 text-destructive" />
              <p className="text-center text-destructive">{message}</p>
              <Button onClick={handleRetry}>Back to Login</Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function OIDCCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center p-4">
          <Card className="w-full max-w-md">
            <CardContent className="flex flex-col items-center gap-4 py-8">
              <Loader2 className="h-12 w-12 animate-spin text-primary" />
              <p className="text-muted-foreground">Loading...</p>
            </CardContent>
          </Card>
        </div>
      }
    >
      <OIDCCallbackContent />
    </Suspense>
  )
}
