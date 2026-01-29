"use client"

import { useState, useEffect } from "react"
import { useRouter, useParams } from "next/navigation"
import { ArrowLeft, Check, Loader2 } from "lucide-react"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useProvider, useCostEstimate, useCreateDeployment } from "@/lib/hooks/use-installer"
import type { DeploymentSize, ProviderRegion } from "@/lib/api/types"

export default function ProviderConfigPage() {
  const router = useRouter()
  const params = useParams()
  const providerId = params.provider as string

  const { data: provider, isLoading: providerLoading } = useProvider(providerId)
  const createDeployment = useCreateDeployment()

  // Form state
  const [name, setName] = useState("")
  const [region, setRegion] = useState("")
  const [size, setSize] = useState<DeploymentSize>("small")
  const [domain, setDomain] = useState("")
  const [sshPublicKey, setSshPublicKey] = useState("")

  const { data: costEstimate, isLoading: costLoading } = useCostEstimate(
    providerId,
    size
  )

  // Set default region when provider loads
  useEffect(() => {
    if (provider && !region) {
      const defaultRegion = provider.regions.find((r) => r.is_default)
      if (defaultRegion) {
        setRegion(defaultRegion.id)
      } else if (provider.regions.length > 0) {
        setRegion(provider.regions[0].id)
      }
    }
  }, [provider, region])

  const selectedSize = provider?.sizes.find((s) => s.id === size)

  const handleDeploy = async () => {
    if (!name || !region || !size) return

    try {
      const deployment = await createDeployment.mutateAsync({
        name,
        provider: providerId,
        region,
        size,
        domain: domain || undefined,
        ssh_public_key: sshPublicKey || undefined,
      })

      router.push(`/install/deploy/${deployment.id}`)
    } catch (error) {
      console.error("Failed to create deployment:", error)
    }
  }

  if (providerLoading) {
    return (
      <div className="container max-w-4xl py-8 space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (!provider) {
    return (
      <div className="container max-w-4xl py-8">
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive text-center">
              Provider not found. Please select a valid provider.
            </p>
            <Button
              variant="outline"
              className="mt-4 mx-auto block"
              onClick={() => router.push("/install")}
            >
              Back to Providers
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="container max-w-4xl py-8 space-y-6">
      {/* Back button */}
      <Button
        variant="ghost"
        className="gap-2"
        onClick={() => router.push("/install")}
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Providers
      </Button>

      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Deploy to {provider.name}</h1>
        <p className="text-muted-foreground">{provider.description}</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Configuration Form */}
        <div className="lg:col-span-2 space-y-6">
          {/* Basic Info */}
          <Card>
            <CardHeader>
              <CardTitle>Deployment Details</CardTitle>
              <CardDescription>
                Configure your Philotes deployment
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Deployment Name</Label>
                <Input
                  id="name"
                  placeholder="my-philotes-cluster"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="region">Region</Label>
                <Select value={region} onValueChange={setRegion}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select region" />
                  </SelectTrigger>
                  <SelectContent>
                    {provider.regions.map((r: ProviderRegion) => (
                      <SelectItem
                        key={r.id}
                        value={r.id}
                        disabled={!r.is_available}
                      >
                        {r.name} ({r.location})
                        {r.is_default && " - Default"}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>

          {/* Size Selection */}
          <Card>
            <CardHeader>
              <CardTitle>Deployment Size</CardTitle>
              <CardDescription>
                Choose the size based on your expected workload
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {provider.sizes.map((s) => (
                  <Card
                    key={s.id}
                    className={`cursor-pointer transition-colors ${
                      size === s.id
                        ? "border-primary bg-primary/5"
                        : "hover:border-muted-foreground"
                    }`}
                    onClick={() => setSize(s.id)}
                  >
                    <CardContent className="pt-6 space-y-2">
                      <div className="flex items-center justify-between">
                        <h4 className="font-semibold">{s.name}</h4>
                        {size === s.id && (
                          <Check className="h-5 w-5 text-primary" />
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {s.description}
                      </p>
                      <div className="text-sm space-y-1">
                        <div>
                          {s.vcpu} vCPU &bull; {s.memory_gb} GB RAM
                        </div>
                        <div>
                          {s.worker_count} workers &bull; {s.storage_size_gb} GB
                          storage
                        </div>
                      </div>
                      <Badge variant="secondary" className="mt-2">
                        &euro;{s.monthly_cost_eur.toFixed(0)}/mo
                      </Badge>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Optional Configuration */}
          <Card>
            <CardHeader>
              <CardTitle>Optional Configuration</CardTitle>
              <CardDescription>
                Additional settings for your deployment
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="domain">Custom Domain (optional)</Label>
                <Input
                  id="domain"
                  placeholder="philotes.example.com"
                  value={domain}
                  onChange={(e) => setDomain(e.target.value)}
                />
                <p className="text-sm text-muted-foreground">
                  Leave empty to use the default IP address
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="ssh">SSH Public Key (optional)</Label>
                <Textarea
                  id="ssh"
                  placeholder="ssh-rsa AAAA..."
                  value={sshPublicKey}
                  onChange={(e) => setSshPublicKey(e.target.value)}
                  rows={3}
                />
                <p className="text-sm text-muted-foreground">
                  For SSH access to the cluster nodes
                </p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Summary Sidebar */}
        <div className="space-y-6">
          <Card className="sticky top-6">
            <CardHeader>
              <CardTitle>Summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Provider</span>
                  <span className="font-medium">{provider.name}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Region</span>
                  <span className="font-medium">
                    {provider.regions.find((r) => r.id === region)?.name ||
                      region}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Size</span>
                  <span className="font-medium">{selectedSize?.name}</span>
                </div>
              </div>

              <hr />

              <div className="space-y-2">
                <h4 className="font-semibold">Infrastructure</h4>
                {selectedSize && (
                  <>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">vCPU</span>
                      <span>{selectedSize.vcpu}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Memory</span>
                      <span>{selectedSize.memory_gb} GB</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Workers</span>
                      <span>{selectedSize.worker_count}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Storage</span>
                      <span>{selectedSize.storage_size_gb} GB</span>
                    </div>
                  </>
                )}
              </div>

              <hr />

              <div className="space-y-2">
                <h4 className="font-semibold">Estimated Cost</h4>
                {costLoading ? (
                  <Skeleton className="h-8 w-full" />
                ) : costEstimate ? (
                  <>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">
                        Control Plane
                      </span>
                      <span>&euro;{costEstimate.control_plane.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Workers</span>
                      <span>&euro;{costEstimate.workers.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Storage</span>
                      <span>&euro;{costEstimate.storage.toFixed(2)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">
                        Load Balancer
                      </span>
                      <span>&euro;{costEstimate.load_balancer.toFixed(2)}</span>
                    </div>
                    <hr />
                    <div className="flex justify-between font-semibold">
                      <span>Total</span>
                      <span>
                        &euro;{costEstimate.total.toFixed(2)}/month
                      </span>
                    </div>
                  </>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    Unable to calculate cost
                  </p>
                )}
              </div>

              <Button
                className="w-full"
                size="lg"
                onClick={handleDeploy}
                disabled={!name || !region || createDeployment.isPending}
              >
                {createDeployment.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  "Deploy Philotes"
                )}
              </Button>

              {createDeployment.isError && (
                <p className="text-sm text-destructive text-center">
                  Failed to create deployment. Please try again.
                </p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
