"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  ArrowLeft,
  ArrowRight,
  SkipForward,
  Shield,
  Info,
  Plus,
  Loader2,
  CheckCircle2,
  ChevronDown,
  Trash2,
  AlertCircle,
} from "lucide-react"
import {
  useOIDCProviders,
  useCreateOIDCProvider,
  useDeleteOIDCProvider,
} from "@/lib/hooks/use-oidc"
import { oidcApi } from "@/lib/api"
import type { OIDCProviderType, OIDCProviderSummary } from "@/lib/api/types"
import { toast } from "sonner"

// Provider templates for quick setup
const providerTemplates: {
  type: OIDCProviderType
  label: string
  description: string
  issuerPlaceholder: string
  defaultIssuer?: string
}[] = [
  {
    type: "google",
    label: "Google Workspace",
    description: "Sign in with Google accounts",
    issuerPlaceholder: "https://accounts.google.com",
    defaultIssuer: "https://accounts.google.com",
  },
  {
    type: "okta",
    label: "Okta",
    description: "Enterprise identity management",
    issuerPlaceholder: "https://your-org.okta.com",
  },
  {
    type: "azure_ad",
    label: "Microsoft Entra ID",
    description: "Azure AD / Microsoft 365",
    issuerPlaceholder: "https://login.microsoftonline.com/{tenant-id}/v2.0",
  },
  {
    type: "auth0",
    label: "Auth0",
    description: "Flexible authentication platform",
    issuerPlaceholder: "https://your-tenant.auth0.com",
  },
  {
    type: "generic",
    label: "Other OIDC Provider",
    description: "Any OIDC-compliant provider",
    issuerPlaceholder: "https://your-idp.example.com",
  },
]

// Form schema for quick provider setup
const quickSetupSchema = z.object({
  provider_type: z.enum(["google", "okta", "azure_ad", "auth0", "generic"]),
  display_name: z.string().min(1, "Display name is required"),
  issuer_url: z.string().url("Must be a valid URL"),
  client_id: z.string().min(1, "Client ID is required"),
  client_secret: z.string().min(1, "Client secret is required"),
})

type QuickSetupValues = z.infer<typeof quickSetupSchema>

interface StepSSOConfigProps {
  onNext: (data?: Record<string, unknown>) => void
  onBack: () => void
  onSkip: () => void
  onConfigured: (configured: boolean) => void
}

export function StepSSOConfig({ onNext, onBack, onSkip, onConfigured }: StepSSOConfigProps) {
  const { data: providersData, isLoading: isLoadingProviders } = useOIDCProviders()
  const createProvider = useCreateOIDCProvider()
  const deleteProvider = useDeleteOIDCProvider()

  const [showAddForm, setShowAddForm] = useState(false)
  const [advancedOpen, setAdvancedOpen] = useState(false)

  const providers = providersData?.providers || []
  const hasProviders = providers.length > 0

  const form = useForm<QuickSetupValues>({
    resolver: zodResolver(quickSetupSchema),
    defaultValues: {
      provider_type: "google",
      display_name: "",
      issuer_url: "",
      client_id: "",
      client_secret: "",
    },
  })

  const selectedProviderType = form.watch("provider_type")
  const template = providerTemplates.find((t) => t.type === selectedProviderType)

  const handleTemplateSelect = (type: OIDCProviderType) => {
    setShowAddForm(true)
    const tmpl = providerTemplates.find((t) => t.type === type)
    form.reset({
      provider_type: type,
      display_name: tmpl?.label || "",
      issuer_url: tmpl?.defaultIssuer || "",
      client_id: "",
      client_secret: "",
    })
  }

  const onSubmit = async (values: QuickSetupValues) => {
    try {
      // Generate a URL-safe name from the display name
      const name = values.display_name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "")

      await createProvider.mutateAsync({
        name,
        display_name: values.display_name,
        provider_type: values.provider_type,
        issuer_url: values.issuer_url,
        client_id: values.client_id,
        client_secret: values.client_secret,
        scopes: ["openid", "profile", "email"],
        enabled: true,
        auto_create_users: true,
      })

      toast.success("Provider configured", {
        description: `${values.display_name} has been added as an SSO provider.`,
      })

      setShowAddForm(false)
      form.reset()
    } catch (err) {
      toast.error("Failed to add provider", {
        description: err instanceof Error ? err.message : "Please check your configuration.",
      })
    }
  }

  const handleDeleteProvider = async (provider: OIDCProviderSummary) => {
    try {
      await deleteProvider.mutateAsync(provider.id)
      toast.success("Provider removed", {
        description: `${provider.display_name} has been removed.`,
      })
    } catch {
      toast.error("Failed to remove provider")
    }
  }

  const handleContinue = () => {
    onConfigured(hasProviders)
    onNext({ sso_configured: hasProviders, provider_count: providers.length })
  }

  const handleSkip = () => {
    onSkip()
  }

  // If adding a provider, show the form
  if (showAddForm) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Add SSO Provider</h2>
          <p className="text-muted-foreground mt-2">
            Configure {template?.label || "your identity provider"} for Single Sign-On.
          </p>
        </div>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="provider_type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Provider Type</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={(value) => {
                      field.onChange(value)
                      const tmpl = providerTemplates.find((t) => t.type === value)
                      if (tmpl?.defaultIssuer) {
                        form.setValue("issuer_url", tmpl.defaultIssuer)
                      }
                    }}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {providerTemplates.map((tmpl) => (
                        <SelectItem key={tmpl.type} value={tmpl.type}>
                          {tmpl.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="display_name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Display Name</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g., Company Google" {...field} />
                  </FormControl>
                  <FormDescription>
                    This name will appear on the login button.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="issuer_url"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Issuer URL</FormLabel>
                  <FormControl>
                    <Input placeholder={template?.issuerPlaceholder} {...field} />
                  </FormControl>
                  <FormDescription>
                    The OIDC issuer URL from your identity provider.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="client_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Client ID</FormLabel>
                  <FormControl>
                    <Input placeholder="Your OAuth client ID" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="client_secret"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Client Secret</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="Your OAuth client secret" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" size="sm" className="gap-2">
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${advancedOpen ? "rotate-180" : ""}`}
                  />
                  Advanced Settings
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent className="mt-4 space-y-4 border-l-2 pl-4">
                <p className="text-sm text-muted-foreground">
                  Advanced settings like scopes, group claims, and role mappings can be
                  configured later in Settings → Authentication.
                </p>
              </CollapsibleContent>
            </Collapsible>

            {createProvider.error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  {createProvider.error instanceof Error
                    ? createProvider.error.message
                    : "Failed to add provider"}
                </AlertDescription>
              </Alert>
            )}

            <div className="flex justify-between pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setShowAddForm(false)
                  form.reset()
                }}
              >
                <ArrowLeft className="mr-2 h-4 w-4" />
                Cancel
              </Button>
              <Button type="submit" disabled={createProvider.isPending}>
                {createProvider.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Adding...
                  </>
                ) : (
                  <>
                    Add Provider
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </>
                )}
              </Button>
            </div>
          </form>
        </Form>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">SSO Configuration</h2>
        <p className="text-muted-foreground mt-2">
          Configure Single Sign-On (SSO) to allow users to authenticate with your identity provider.
        </p>
      </div>

      <Alert>
        <Info className="h-4 w-4" />
        <AlertTitle>Optional Step</AlertTitle>
        <AlertDescription>
          SSO configuration is optional. You can skip this step and configure it later from Settings.
        </AlertDescription>
      </Alert>

      {/* Existing providers */}
      {isLoadingProviders ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : hasProviders ? (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5 text-green-600" />
            <span className="font-medium">Configured Providers</span>
          </div>
          <div className="space-y-2">
            {providers.map((provider) => (
              <div
                key={provider.id}
                className="flex items-center justify-between rounded-lg border p-4"
              >
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary/10 rounded-lg">
                    <Shield className="h-4 w-4 text-primary" />
                  </div>
                  <div>
                    <p className="font-medium">{provider.display_name}</p>
                    <p className="text-sm text-muted-foreground">
                      {oidcApi.getProviderTypeLabel(provider.provider_type)}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant={provider.enabled ? "default" : "secondary"}>
                    {provider.enabled ? "Enabled" : "Disabled"}
                  </Badge>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleDeleteProvider(provider)}
                    disabled={deleteProvider.isPending}
                  >
                    <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
          <Button variant="outline" onClick={() => setShowAddForm(true)} className="w-full">
            <Plus className="mr-2 h-4 w-4" />
            Add Another Provider
          </Button>
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-sm font-medium text-muted-foreground">
            Choose an identity provider to get started:
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            {providerTemplates.slice(0, 4).map((tmpl) => (
              <button
                key={tmpl.type}
                onClick={() => handleTemplateSelect(tmpl.type)}
                className="flex items-start gap-3 rounded-lg border p-4 text-left hover:bg-muted/50 transition-colors"
              >
                <div className="p-2 bg-primary/10 rounded-lg shrink-0">
                  <Shield className="h-4 w-4 text-primary" />
                </div>
                <div>
                  <p className="font-medium">{tmpl.label}</p>
                  <p className="text-sm text-muted-foreground">{tmpl.description}</p>
                </div>
              </button>
            ))}
          </div>
          <Button
            variant="ghost"
            onClick={() => handleTemplateSelect("generic")}
            className="w-full"
          >
            <Plus className="mr-2 h-4 w-4" />
            Other OIDC Provider
          </Button>
        </div>
      )}

      <div className="flex justify-between pt-4">
        <Button variant="outline" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <div className="flex gap-2">
          {!hasProviders && (
            <Button variant="ghost" onClick={handleSkip}>
              <SkipForward className="mr-2 h-4 w-4" />
              Skip for Now
            </Button>
          )}
          <Button onClick={handleContinue}>
            {hasProviders ? "Continue" : "Configure Later"}
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
