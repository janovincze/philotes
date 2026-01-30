"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  Plus,
  MoreHorizontal,
  Pencil,
  Trash2,
  TestTube,
  AlertCircle,
  CheckCircle,
  XCircle,
  Loader2,
} from "lucide-react"
import {
  useOIDCProviders,
  useDeleteOIDCProvider,
  useTestOIDCProvider,
  useUpdateOIDCProvider,
} from "@/lib/hooks/use-oidc"
import { oidcApi } from "@/lib/api"
import type { OIDCProviderSummary } from "@/lib/api/types"
import { OIDCProviderForm } from "./oidc-provider-form"

interface OIDCProvidersListProps {
  className?: string
}

export function OIDCProvidersList({ className }: OIDCProvidersListProps) {
  const { data, isLoading, error } = useOIDCProviders()
  const deleteProvider = useDeleteOIDCProvider()
  const testProvider = useTestOIDCProvider()
  const updateProvider = useUpdateOIDCProvider()

  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingProvider, setEditingProvider] = useState<OIDCProviderSummary | null>(
    null
  )
  const [deletingProvider, setDeletingProvider] = useState<OIDCProviderSummary | null>(
    null
  )
  const [testResult, setTestResult] = useState<{
    providerId: string
    success: boolean
    message: string
  } | null>(null)

  const handleTest = async (provider: OIDCProviderSummary) => {
    setTestResult(null)
    try {
      const result = await testProvider.mutateAsync(provider.id)
      setTestResult({
        providerId: provider.id,
        success: result.success,
        message: result.message,
      })
    } catch (err) {
      setTestResult({
        providerId: provider.id,
        success: false,
        message: err instanceof Error ? err.message : "Test failed",
      })
    }
  }

  const handleDelete = async () => {
    if (!deletingProvider) return
    try {
      await deleteProvider.mutateAsync(deletingProvider.id)
      setDeletingProvider(null)
    } catch (err) {
      console.error("Failed to delete provider:", err)
    }
  }

  const handleToggleEnabled = async (provider: OIDCProviderSummary) => {
    try {
      await updateProvider.mutateAsync({
        id: provider.id,
        request: { enabled: !provider.enabled },
      })
    } catch (err) {
      console.error("Failed to toggle provider:", err)
    }
  }

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>OIDC Providers</CardTitle>
          <CardDescription>Loading providers...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>OIDC Providers</CardTitle>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Failed to load providers. Please try again.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    )
  }

  const providers = data?.providers || []

  return (
    <>
      <Card className={className}>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>OIDC Providers</CardTitle>
            <CardDescription>
              Configure identity providers for Single Sign-On
            </CardDescription>
          </div>
          <Button onClick={() => setIsCreateOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Add Provider
          </Button>
        </CardHeader>
        <CardContent>
          {providers.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <p>No OIDC providers configured.</p>
              <p className="text-sm mt-1">
                Add a provider to enable Single Sign-On.
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Provider</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Issuer</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-[100px]">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {providers.map((provider) => (
                  <TableRow key={provider.id}>
                    <TableCell>
                      <div>
                        <p className="font-medium">{provider.display_name}</p>
                        <p className="text-sm text-muted-foreground">
                          {provider.name}
                        </p>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {oidcApi.getProviderTypeLabel(provider.provider_type)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-muted-foreground truncate max-w-[200px] block">
                        {provider.issuer_url}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {provider.enabled ? (
                          <Badge variant="default" className="bg-green-600">
                            Enabled
                          </Badge>
                        ) : (
                          <Badge variant="secondary">Disabled</Badge>
                        )}
                        {testResult?.providerId === provider.id && (
                          <span
                            className={
                              testResult.success
                                ? "text-green-600"
                                : "text-destructive"
                            }
                          >
                            {testResult.success ? (
                              <CheckCircle className="h-4 w-4" />
                            ) : (
                              <XCircle className="h-4 w-4" />
                            )}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                            <span className="sr-only">Actions</span>
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => setEditingProvider(provider)}
                          >
                            <Pencil className="h-4 w-4 mr-2" />
                            Edit
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleTest(provider)}
                            disabled={testProvider.isPending}
                          >
                            {testProvider.isPending ? (
                              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                            ) : (
                              <TestTube className="h-4 w-4 mr-2" />
                            )}
                            Test Connection
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleToggleEnabled(provider)}
                          >
                            {provider.enabled ? (
                              <>
                                <XCircle className="h-4 w-4 mr-2" />
                                Disable
                              </>
                            ) : (
                              <>
                                <CheckCircle className="h-4 w-4 mr-2" />
                                Enable
                              </>
                            )}
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            onClick={() => setDeletingProvider(provider)}
                            className="text-destructive focus:text-destructive"
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Create Provider Dialog */}
      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Add OIDC Provider</DialogTitle>
            <DialogDescription>
              Configure a new identity provider for Single Sign-On
            </DialogDescription>
          </DialogHeader>
          <OIDCProviderForm
            onSuccess={() => setIsCreateOpen(false)}
            onCancel={() => setIsCreateOpen(false)}
          />
        </DialogContent>
      </Dialog>

      {/* Edit Provider Dialog */}
      <Dialog
        open={!!editingProvider}
        onOpenChange={(open: boolean) => !open && setEditingProvider(null)}
      >
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit OIDC Provider</DialogTitle>
            <DialogDescription>
              Update the configuration for {editingProvider?.display_name}
            </DialogDescription>
          </DialogHeader>
          {editingProvider && (
            <OIDCProviderForm
              provider={editingProvider}
              onSuccess={() => setEditingProvider(null)}
              onCancel={() => setEditingProvider(null)}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        open={!!deletingProvider}
        onOpenChange={(open) => !open && setDeletingProvider(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Provider</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the provider &quot;
              {deletingProvider?.display_name}&quot;? This action cannot be undone.
              Users who sign in with this provider will no longer be able to access
              the application.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteProvider.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : null}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
