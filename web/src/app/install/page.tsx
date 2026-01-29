"use client"

import { useRouter } from "next/navigation"
import { Cloud, Server, Globe, Zap, Shield } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { useProviders } from "@/lib/hooks/use-installer"

// Provider icons mapping
const providerIcons: Record<string, React.ReactNode> = {
  hetzner: <Server className="h-8 w-8" />,
  scaleway: <Cloud className="h-8 w-8" />,
  ovh: <Globe className="h-8 w-8" />,
  exoscale: <Shield className="h-8 w-8" />,
  contabo: <Zap className="h-8 w-8" />,
}

export default function InstallPage() {
  const router = useRouter()
  const { data: providers, isLoading, error } = useProviders()

  const handleSelectProvider = (providerId: string) => {
    router.push(`/install/${providerId}`)
  }

  return (
    <div className="container max-w-5xl py-8 space-y-8">
      {/* Header */}
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold">Deploy Philotes</h1>
        <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
          One-click deployment to your preferred European cloud provider.
          Get started in minutes with our guided installer.
        </p>
      </div>

      {/* Features */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <Card>
          <CardContent className="pt-6 text-center">
            <Zap className="h-8 w-8 mx-auto mb-2 text-primary" />
            <h3 className="font-semibold">Fast Setup</h3>
            <p className="text-sm text-muted-foreground">
              Deploy in under 15 minutes
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6 text-center">
            <Shield className="h-8 w-8 mx-auto mb-2 text-primary" />
            <h3 className="font-semibold">GDPR Compliant</h3>
            <p className="text-sm text-muted-foreground">
              European data residency
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6 text-center">
            <Globe className="h-8 w-8 mx-auto mb-2 text-primary" />
            <h3 className="font-semibold">Your Infrastructure</h3>
            <p className="text-sm text-muted-foreground">
              Data stays in your cloud
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Provider Selection */}
      <div>
        <h2 className="text-2xl font-semibold mb-4 text-center">
          Select Your Cloud Provider
        </h2>

        {error && (
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <p className="text-destructive text-center">
                Failed to load providers. Please try again later.
              </p>
            </CardContent>
          </Card>
        )}

        {isLoading && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <Card key={i}>
                <CardContent className="pt-6">
                  <Skeleton className="h-8 w-8 rounded mb-4" />
                  <Skeleton className="h-6 w-3/4 mb-2" />
                  <Skeleton className="h-4 w-full" />
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        {providers && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {providers.map((provider) => {
              // Get price range from sizes
              const minPrice = Math.min(
                ...provider.sizes.map((s) => s.monthly_cost_eur)
              )
              const maxPrice = Math.max(
                ...provider.sizes.map((s) => s.monthly_cost_eur)
              )

              return (
                <Card
                  key={provider.id}
                  className="cursor-pointer hover:border-primary transition-colors"
                  onClick={() => handleSelectProvider(provider.id)}
                >
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="text-primary">
                          {providerIcons[provider.id] || <Cloud className="h-8 w-8" />}
                        </div>
                        <CardTitle className="text-lg">{provider.name}</CardTitle>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <CardDescription>{provider.description}</CardDescription>

                    <div className="flex items-center justify-between">
                      <div className="text-sm text-muted-foreground">
                        {provider.regions.length} regions
                      </div>
                      <Badge variant="secondary">
                        From &euro;{minPrice.toFixed(0)}/mo
                      </Badge>
                    </div>

                    <Button className="w-full" variant="outline">
                      Select {provider.name}
                    </Button>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )}
      </div>

      {/* Footer info */}
      <div className="text-center text-sm text-muted-foreground">
        <p>
          Need help choosing? All providers offer similar performance.
          Choose based on your region preference and budget.
        </p>
      </div>
    </div>
  )
}
