"use client"

import { useEffect, useRef, Suspense } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Loader2, CheckCircle, XCircle } from "lucide-react"

function OAuthCallbackContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const hasProcessed = useRef(false)

  // Parse search params
  const success = searchParams.get("success")
  const error = searchParams.get("error")
  const provider = searchParams.get("provider")
  const credentialId = searchParams.get("credential_id")

  // Determine status from URL params
  const isSuccess = success === "true" && provider && credentialId
  const status = isSuccess ? "success" : "error"
  const message = isSuccess
    ? `Successfully connected to ${provider}`
    : error || "Invalid callback parameters"

  // Handle side effects (storage and redirect) in useEffect
  useEffect(() => {
    if (hasProcessed.current) return
    hasProcessed.current = true

    if (isSuccess && provider && credentialId) {
      // Store the credential ID in session storage
      sessionStorage.setItem(`philotes_credential_${provider}`, credentialId)

      // Redirect after a short delay
      const timeoutId = setTimeout(() => {
        router.push(`/install/${provider}`)
      }, 2000)

      return () => clearTimeout(timeoutId)
    }
  }, [isSuccess, provider, credentialId, router])

  const handleRetry = () => {
    if (provider) {
      router.push(`/install/${provider}`)
    } else {
      router.push("/install")
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle>
            {status === "success" && "Connection Successful"}
            {status === "error" && "Connection Failed"}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-4">
          {status === "success" && (
            <>
              <CheckCircle className="h-12 w-12 text-green-500" />
              <p className="text-center">{message}</p>
              <p className="text-sm text-muted-foreground">
                Redirecting you back to the installer...
              </p>
            </>
          )}

          {status === "error" && (
            <>
              <XCircle className="h-12 w-12 text-destructive" />
              <p className="text-center text-destructive">{message}</p>
              <Button onClick={handleRetry}>Try Again</Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function OAuthCallbackPage() {
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
      <OAuthCallbackContent />
    </Suspense>
  )
}
