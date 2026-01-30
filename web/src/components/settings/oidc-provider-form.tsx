"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Loader2, Plus, Trash2, AlertCircle, CheckCircle } from "lucide-react"
import {
  useCreateOIDCProvider,
  useUpdateOIDCProvider,
  useTestOIDCProvider,
} from "@/lib/hooks/use-oidc"
import type { OIDCProviderSummary, OIDCProviderType } from "@/lib/api/types"

const providerTypes: { value: OIDCProviderType; label: string }[] = [
  { value: "google", label: "Google" },
  { value: "okta", label: "Okta" },
  { value: "azure_ad", label: "Azure AD / Entra ID" },
  { value: "auth0", label: "Auth0" },
  { value: "generic", label: "Generic OIDC" },
]

const roles = [
  { value: "admin", label: "Admin" },
  { value: "operator", label: "Operator" },
  { value: "viewer", label: "Viewer" },
] as const

// Form schema - use z.input for default values in form
const formSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .max(100, "Name must be at most 100 characters")
    .regex(
      /^[a-z0-9-]+$/,
      "Name must contain only lowercase letters, numbers, and hyphens"
    )
    .refine((val) => !val.startsWith("-") && !val.endsWith("-"), {
      message: "Name cannot start or end with a hyphen",
    }),
  display_name: z
    .string()
    .min(1, "Display name is required")
    .max(255, "Display name must be at most 255 characters"),
  provider_type: z.enum(["google", "okta", "azure_ad", "auth0", "generic"]),
  issuer_url: z.string().url("Must be a valid URL"),
  client_id: z.string().min(1, "Client ID is required"),
  client_secret: z.string().optional().default(""),
  scopes: z.string().default("openid profile email"),
  groups_claim: z.string().default("groups"),
  default_role: z.enum(["admin", "operator", "viewer"]).default("viewer"),
  enabled: z.boolean().default(true),
  auto_create_users: z.boolean().default(true),
})

type FormData = z.input<typeof formSchema>

// Role mapping entry
interface RoleMappingEntry {
  group: string
  role: "admin" | "operator" | "viewer"
}

interface OIDCProviderFormProps {
  provider?: OIDCProviderSummary
  onSuccess?: () => void
  onCancel?: () => void
}

export function OIDCProviderForm({
  provider,
  onSuccess,
  onCancel,
}: OIDCProviderFormProps) {
  const isEditing = !!provider
  const createProvider = useCreateOIDCProvider()
  const updateProvider = useUpdateOIDCProvider()
  const testProvider = useTestOIDCProvider()

  const [roleMappings, setRoleMappings] = useState<RoleMappingEntry[]>(
    provider?.role_mapping
      ? Object.entries(provider.role_mapping).map(([group, role]) => ({
          group,
          role,
        }))
      : []
  )
  const [testResult, setTestResult] = useState<{
    success: boolean
    message: string
  } | null>(null)

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<FormData>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: provider?.name || "",
      display_name: provider?.display_name || "",
      provider_type: provider?.provider_type || "generic",
      issuer_url: provider?.issuer_url || "",
      client_id: provider?.client_id || "",
      client_secret: "",
      scopes: provider?.scopes?.join(" ") || "openid profile email",
      groups_claim: provider?.groups_claim || "groups",
      default_role: provider?.default_role || "viewer",
      enabled: provider?.enabled ?? true,
      auto_create_users: provider?.auto_create_users ?? true,
    },
  })

  const providerType = watch("provider_type")

  // Update issuer URL placeholder based on provider type
  const getIssuerPlaceholder = () => {
    switch (providerType) {
      case "google":
        return "https://accounts.google.com"
      case "okta":
        return "https://your-tenant.okta.com"
      case "azure_ad":
        return "https://login.microsoftonline.com/{tenant-id}/v2.0"
      case "auth0":
        return "https://your-tenant.auth0.com"
      default:
        return "https://your-idp.example.com"
    }
  }

  const onSubmit = async (data: FormData) => {
    const roleMapping = roleMappings.reduce(
      (acc, { group, role }) => ({ ...acc, [group]: role }),
      {} as Record<string, "admin" | "operator" | "viewer">
    )

    const payload = {
      ...data,
      scopes: (data.scopes || "openid profile email").split(/\s+/).filter(Boolean),
      role_mapping: roleMapping,
    }

    try {
      if (isEditing && provider) {
        // For updates, only send client_secret if it was changed
        const updatePayload = {
          display_name: payload.display_name,
          issuer_url: payload.issuer_url,
          client_id: payload.client_id,
          ...(payload.client_secret && { client_secret: payload.client_secret }),
          scopes: payload.scopes,
          groups_claim: payload.groups_claim,
          role_mapping: payload.role_mapping,
          default_role: payload.default_role,
          enabled: payload.enabled,
          auto_create_users: payload.auto_create_users,
        }
        await updateProvider.mutateAsync({ id: provider.id, request: updatePayload })
      } else {
        // For creates, client_secret is required
        if (!payload.client_secret) {
          throw new Error("Please provide a client secret. This is required when creating a new OIDC provider.")
        }
        await createProvider.mutateAsync({
          name: payload.name,
          display_name: payload.display_name,
          provider_type: payload.provider_type,
          issuer_url: payload.issuer_url,
          client_id: payload.client_id,
          client_secret: payload.client_secret,
          scopes: payload.scopes,
          groups_claim: payload.groups_claim,
          role_mapping: payload.role_mapping,
          default_role: payload.default_role,
          enabled: payload.enabled,
          auto_create_users: payload.auto_create_users,
        })
      }
      onSuccess?.()
    } catch (error) {
      console.error("Failed to save provider:", error)
    }
  }

  const handleTest = async () => {
    if (!provider) return
    setTestResult(null)
    try {
      const result = await testProvider.mutateAsync(provider.id)
      setTestResult(result)
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : "Test failed",
      })
    }
  }

  const addRoleMapping = () => {
    setRoleMappings([...roleMappings, { group: "", role: "viewer" }])
  }

  const removeRoleMapping = (index: number) => {
    setRoleMappings(roleMappings.filter((_, i) => i !== index))
  }

  const updateRoleMapping = (
    index: number,
    field: "group" | "role",
    value: string
  ) => {
    const updated = [...roleMappings]
    if (field === "role") {
      updated[index].role = value as "admin" | "operator" | "viewer"
    } else {
      updated[index].group = value
    }
    setRoleMappings(updated)
  }

  const isLoading = createProvider.isPending || updateProvider.isPending
  const error = createProvider.error || updateProvider.error

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            {error instanceof Error ? error.message : "Failed to save provider"}
          </AlertDescription>
        </Alert>
      )}

      {/* Basic Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Settings</CardTitle>
          <CardDescription>
            Configure the OIDC provider connection details
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="name">Name *</Label>
              <Input
                id="name"
                {...register("name")}
                placeholder="my-provider"
                disabled={isEditing}
              />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
              <p className="text-xs text-muted-foreground">
                Unique identifier (lowercase, hyphens only)
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="display_name">Display Name *</Label>
              <Input
                id="display_name"
                {...register("display_name")}
                placeholder="My Identity Provider"
              />
              {errors.display_name && (
                <p className="text-sm text-destructive">
                  {errors.display_name.message}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="provider_type">Provider Type *</Label>
            <Select
              value={providerType}
              onValueChange={(value) =>
                setValue("provider_type", value as OIDCProviderType)
              }
              disabled={isEditing}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select provider type" />
              </SelectTrigger>
              <SelectContent>
                {providerTypes.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    {type.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="issuer_url">Issuer URL *</Label>
            <Input
              id="issuer_url"
              {...register("issuer_url")}
              placeholder={getIssuerPlaceholder()}
            />
            {errors.issuer_url && (
              <p className="text-sm text-destructive">
                {errors.issuer_url.message}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              The OIDC discovery endpoint (/.well-known/openid-configuration will
              be appended)
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="client_id">Client ID *</Label>
              <Input id="client_id" {...register("client_id")} />
              {errors.client_id && (
                <p className="text-sm text-destructive">
                  {errors.client_id.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="client_secret">
                Client Secret {isEditing ? "(leave empty to keep current)" : "*"}
              </Label>
              <Input
                id="client_secret"
                type="password"
                {...register("client_secret")}
                placeholder={isEditing ? "••••••••" : ""}
              />
              {errors.client_secret && (
                <p className="text-sm text-destructive">
                  {errors.client_secret.message}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="scopes">Scopes</Label>
            <Input
              id="scopes"
              {...register("scopes")}
              placeholder="openid profile email"
            />
            <p className="text-xs text-muted-foreground">
              Space-separated list of OIDC scopes
            </p>
          </div>
        </CardContent>
      </Card>

      {/* User Provisioning */}
      <Card>
        <CardHeader>
          <CardTitle>User Provisioning</CardTitle>
          <CardDescription>
            Configure how users are created and assigned roles
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Auto-create Users</Label>
              <p className="text-sm text-muted-foreground">
                Automatically create user accounts on first login
              </p>
            </div>
            <Switch
              checked={watch("auto_create_users")}
              onCheckedChange={(checked) => setValue("auto_create_users", checked)}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="default_role">Default Role</Label>
              <Select
                value={watch("default_role")}
                onValueChange={(value) =>
                  setValue("default_role", value as "admin" | "operator" | "viewer")
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {roles.map((role) => (
                    <SelectItem key={role.value} value={role.value}>
                      {role.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Role assigned when no group mapping matches
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="groups_claim">Groups Claim</Label>
              <Input
                id="groups_claim"
                {...register("groups_claim")}
                placeholder="groups"
              />
              <p className="text-xs text-muted-foreground">
                ID token claim containing user groups
              </p>
            </div>
          </div>

          {/* Role Mappings */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Group to Role Mappings</Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={addRoleMapping}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add Mapping
              </Button>
            </div>
            <p className="text-sm text-muted-foreground">
              Map identity provider groups to Philotes roles
            </p>
            {roleMappings.length > 0 && (
              <div className="space-y-2">
                {roleMappings.map((mapping, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      value={mapping.group}
                      onChange={(e) =>
                        updateRoleMapping(index, "group", e.target.value)
                      }
                      placeholder="Group name (e.g., admins)"
                      className="flex-1"
                    />
                    <Select
                      value={mapping.role}
                      onValueChange={(value) =>
                        updateRoleMapping(index, "role", value)
                      }
                    >
                      <SelectTrigger className="w-32">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {roles.map((role) => (
                          <SelectItem key={role.value} value={role.value}>
                            {role.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeRoleMapping(index)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Status */}
      <Card>
        <CardHeader>
          <CardTitle>Status</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Enabled</Label>
              <p className="text-sm text-muted-foreground">
                Allow users to sign in with this provider
              </p>
            </div>
            <Switch
              checked={watch("enabled")}
              onCheckedChange={(checked) => setValue("enabled", checked)}
            />
          </div>

          {isEditing && provider && (
            <div className="pt-4 border-t">
              <Button
                type="button"
                variant="outline"
                onClick={handleTest}
                disabled={testProvider.isPending}
              >
                {testProvider.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Testing...
                  </>
                ) : (
                  "Test Connection"
                )}
              </Button>
              {testResult && (
                <Alert
                  variant={testResult.success ? "default" : "destructive"}
                  className="mt-3"
                >
                  {testResult.success ? (
                    <CheckCircle className="h-4 w-4" />
                  ) : (
                    <AlertCircle className="h-4 w-4" />
                  )}
                  <AlertDescription>{testResult.message}</AlertDescription>
                </Alert>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex justify-end gap-3">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button type="submit" disabled={isLoading}>
          {isLoading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
          {isEditing ? "Save Changes" : "Create Provider"}
        </Button>
      </div>
    </form>
  )
}
