"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useStoreCredential } from "@/lib/hooks/use-oauth"
import type { ProviderCredentials } from "@/lib/api/types"
import { Loader2, Eye, EyeOff } from "lucide-react"

interface ManualCredentialsProps {
  provider: string
  providerName: string
  onSuccess?: () => void
  onCancel?: () => void
}

export function ManualCredentials({
  provider,
  providerName,
  onSuccess,
  onCancel,
}: ManualCredentialsProps) {
  const { mutateAsync: storeCredential, isPending, error } = useStoreCredential()
  const [showSecrets, setShowSecrets] = useState(false)
  const [credentials, setCredentials] = useState<ProviderCredentials>({})

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      await storeCredential({
        provider,
        credentials,
        expiresIn: 24 * 60 * 60, // 24 hours
      })
      onSuccess?.()
    } catch {
      // Error is handled by the mutation
    }
  }

  const updateCredential = (key: keyof ProviderCredentials, value: string) => {
    setCredentials((prev) => ({ ...prev, [key]: value }))
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Enter {providerName} Credentials</h3>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => setShowSecrets(!showSecrets)}
        >
          {showSecrets ? (
            <EyeOff className="h-4 w-4" />
          ) : (
            <Eye className="h-4 w-4" />
          )}
        </Button>
      </div>

      {renderProviderFields(provider, credentials, updateCredential, showSecrets)}

      {error && (
        <p className="text-sm text-destructive">
          {error instanceof Error ? error.message : "Failed to store credentials"}
        </p>
      )}

      <div className="flex gap-2 justify-end">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button type="submit" disabled={isPending}>
          {isPending && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
          Save Credentials
        </Button>
      </div>

      <p className="text-xs text-muted-foreground">
        Credentials are encrypted at rest and will expire after 24 hours.
      </p>
    </form>
  )
}

function renderProviderFields(
  provider: string,
  credentials: ProviderCredentials,
  updateCredential: (key: keyof ProviderCredentials, value: string) => void,
  showSecrets: boolean
) {
  const inputType = showSecrets ? "text" : "password"

  switch (provider) {
    case "hetzner":
      return (
        <div className="space-y-3">
          <div>
            <Label htmlFor="hetzner_token">API Token</Label>
            <Input
              id="hetzner_token"
              type={inputType}
              placeholder="Enter your Hetzner API token"
              value={credentials.hetzner_token || ""}
              onChange={(e) => updateCredential("hetzner_token", e.target.value)}
              required
            />
            <p className="text-xs text-muted-foreground mt-1">
              Generate a token at{" "}
              <a
                href="https://console.hetzner.cloud/projects"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                Hetzner Cloud Console
              </a>
              {" → Security → API Tokens"}
            </p>
          </div>
        </div>
      )

    case "scaleway":
      return (
        <div className="space-y-3">
          <div>
            <Label htmlFor="scaleway_access_key">Access Key</Label>
            <Input
              id="scaleway_access_key"
              type={inputType}
              placeholder="SCWXXXXXXXXXXXXXXXXX"
              value={credentials.scaleway_access_key || ""}
              onChange={(e) => updateCredential("scaleway_access_key", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="scaleway_secret_key">Secret Key</Label>
            <Input
              id="scaleway_secret_key"
              type={inputType}
              placeholder="Enter your secret key"
              value={credentials.scaleway_secret_key || ""}
              onChange={(e) => updateCredential("scaleway_secret_key", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="scaleway_project_id">Project ID</Label>
            <Input
              id="scaleway_project_id"
              type="text"
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              value={credentials.scaleway_project_id || ""}
              onChange={(e) => updateCredential("scaleway_project_id", e.target.value)}
              required
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Generate credentials at{" "}
            <a
              href="https://console.scaleway.com/iam/api-keys"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline"
            >
              Scaleway Console
            </a>
          </p>
        </div>
      )

    case "ovh":
      return (
        <div className="space-y-3">
          <div>
            <Label htmlFor="ovh_endpoint">API Endpoint</Label>
            <Input
              id="ovh_endpoint"
              type="text"
              placeholder="ovh-eu"
              value={credentials.ovh_endpoint || "ovh-eu"}
              onChange={(e) => updateCredential("ovh_endpoint", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="ovh_application_key">Application Key</Label>
            <Input
              id="ovh_application_key"
              type={inputType}
              placeholder="Enter application key"
              value={credentials.ovh_application_key || ""}
              onChange={(e) => updateCredential("ovh_application_key", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="ovh_application_secret">Application Secret</Label>
            <Input
              id="ovh_application_secret"
              type={inputType}
              placeholder="Enter application secret"
              value={credentials.ovh_application_secret || ""}
              onChange={(e) => updateCredential("ovh_application_secret", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="ovh_consumer_key">Consumer Key</Label>
            <Input
              id="ovh_consumer_key"
              type={inputType}
              placeholder="Enter consumer key"
              value={credentials.ovh_consumer_key || ""}
              onChange={(e) => updateCredential("ovh_consumer_key", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="ovh_service_name">Service Name</Label>
            <Input
              id="ovh_service_name"
              type="text"
              placeholder="Your OVH project name"
              value={credentials.ovh_service_name || ""}
              onChange={(e) => updateCredential("ovh_service_name", e.target.value)}
              required
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Create API credentials at{" "}
            <a
              href="https://api.ovh.com/createToken/"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline"
            >
              OVH API Console
            </a>
          </p>
        </div>
      )

    case "exoscale":
      return (
        <div className="space-y-3">
          <div>
            <Label htmlFor="exoscale_api_key">API Key</Label>
            <Input
              id="exoscale_api_key"
              type={inputType}
              placeholder="EXO..."
              value={credentials.exoscale_api_key || ""}
              onChange={(e) => updateCredential("exoscale_api_key", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="exoscale_api_secret">API Secret</Label>
            <Input
              id="exoscale_api_secret"
              type={inputType}
              placeholder="Enter your API secret"
              value={credentials.exoscale_api_secret || ""}
              onChange={(e) => updateCredential("exoscale_api_secret", e.target.value)}
              required
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Generate credentials at{" "}
            <a
              href="https://portal.exoscale.com/iam/api-keys"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline"
            >
              Exoscale Portal
            </a>
          </p>
        </div>
      )

    case "contabo":
      return (
        <div className="space-y-3">
          <div>
            <Label htmlFor="contabo_client_id">Client ID</Label>
            <Input
              id="contabo_client_id"
              type={inputType}
              placeholder="Enter client ID"
              value={credentials.contabo_client_id || ""}
              onChange={(e) => updateCredential("contabo_client_id", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="contabo_client_secret">Client Secret</Label>
            <Input
              id="contabo_client_secret"
              type={inputType}
              placeholder="Enter client secret"
              value={credentials.contabo_client_secret || ""}
              onChange={(e) => updateCredential("contabo_client_secret", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="contabo_api_user">API User</Label>
            <Input
              id="contabo_api_user"
              type="text"
              placeholder="Enter API user"
              value={credentials.contabo_api_user || ""}
              onChange={(e) => updateCredential("contabo_api_user", e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="contabo_api_password">API Password</Label>
            <Input
              id="contabo_api_password"
              type={inputType}
              placeholder="Enter API password"
              value={credentials.contabo_api_password || ""}
              onChange={(e) => updateCredential("contabo_api_password", e.target.value)}
              required
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Generate credentials in the{" "}
            <a
              href="https://my.contabo.com/api/details"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline"
            >
              Contabo Customer Control Panel
            </a>
          </p>
        </div>
      )

    default:
      return (
        <p className="text-muted-foreground">
          Unknown provider: {provider}
        </p>
      )
  }
}
