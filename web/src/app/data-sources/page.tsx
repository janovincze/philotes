"use client"

import { useState, useCallback, useEffect, useRef } from "react"
import {
  DatabaseZap,
  Plus,
  CheckCircle,
  XCircle,
  AlertCircle,
  MoreHorizontal,
  Pencil,
  Trash2,
  Plug,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { toast } from "sonner"
import { DataSourceForm } from "@/components/data-sources/data-source-form"
import {
  useQueryDataSources,
  useCreateQueryDataSource,
  useUpdateQueryDataSource,
  useDeleteQueryDataSource,
  useTestQueryDataSourceConnection,
} from "@/lib/hooks/use-query-data-sources"
import type { QueryDataSource, QueryDataSourceStatus } from "@/lib/api/types"

function StatusBadge({ status }: { status: QueryDataSourceStatus }) {
  const config = {
    active: { icon: CheckCircle, variant: "default" as const, color: "text-green-500" },
    inactive: { icon: AlertCircle, variant: "secondary" as const, color: "text-muted-foreground" },
    error: { icon: XCircle, variant: "destructive" as const, color: "text-red-500" },
  }

  const { icon: Icon, variant, color } = config[status]

  return (
    <Badge variant={variant} className="gap-1">
      <Icon className={`h-3 w-3 ${color}`} />
      <span className="capitalize">{status}</span>
    </Badge>
  )
}

function TypeBadge({ type }: { type: string }) {
  return (
    <Badge variant="outline" className="capitalize">
      {type}
    </Badge>
  )
}

function DataSourcesTableSkeleton() {
  return (
    <div className="space-y-3">
      {[1, 2, 3].map((i) => (
        <div key={i} className="flex items-center gap-4 p-4 border rounded-lg">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-5 w-20" />
          <Skeleton className="h-5 w-24" />
          <Skeleton className="h-5 w-48" />
          <Skeleton className="h-5 w-16" />
        </div>
      ))}
    </div>
  )
}

export default function DataSourcesPage() {
  const { data: dataSources, isLoading, error } = useQueryDataSources()
  const createMutation = useCreateQueryDataSource()
  const updateMutation = useUpdateQueryDataSource()
  const deleteMutation = useDeleteQueryDataSource()
  const testConnectionMutation = useTestQueryDataSourceConnection()

  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [editingSource, setEditingSource] = useState<QueryDataSource | null>(null)
  const [deletingSource, setDeletingSource] = useState<QueryDataSource | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)

  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (resetTimerRef.current) clearTimeout(resetTimerRef.current)
    }
  }, [])

  const handleCreate = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async (values: any) => {
      try {
        await createMutation.mutateAsync(values)
        toast.success("Data source created successfully")
        setCreateDialogOpen(false)
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Failed to create data source")
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [createMutation.mutateAsync]
  )

  const handleUpdate = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async (values: any) => {
      if (!editingSource) return
      try {
        // Only send changed fields for update
        const input: Record<string, unknown> = {}
        if (values.name !== editingSource.name) input.name = values.name
        if (values.host !== editingSource.host) input.host = values.host
        if (values.port !== editingSource.port) input.port = values.port
        if (values.database_name !== editingSource.database_name) input.database_name = values.database_name
        if (values.username !== editingSource.username) input.username = values.username
        if (values.password) input.password = values.password
        if (values.ssl_mode !== editingSource.ssl_mode) input.ssl_mode = values.ssl_mode

        await updateMutation.mutateAsync({ id: editingSource.id, input })
        toast.success("Data source updated successfully")
        setEditingSource(null)
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Failed to update data source")
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [editingSource, updateMutation.mutateAsync]
  )

  const handleDelete = useCallback(async () => {
    if (!deletingSource) return
    try {
      await deleteMutation.mutateAsync(deletingSource.id)
      toast.success("Data source deleted")
      setDeletingSource(null)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete data source")
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [deletingSource, deleteMutation.mutateAsync])

  const handleTestConnection = useCallback(
    async (id: string) => {
      if (resetTimerRef.current) clearTimeout(resetTimerRef.current)
      setTestingId(id)
      try {
        const result = await testConnectionMutation.mutateAsync(id)
        if (result.success) {
          toast.success(result.message || "Connection successful")
        } else {
          toast.error(result.message || "Connection failed")
        }
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Connection test failed")
      }
      resetTimerRef.current = setTimeout(() => setTestingId(null), 2000)
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [testConnectionMutation.mutateAsync]
  )

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Query Data Sources</h1>
          <p className="text-muted-foreground">
            Connect external databases for federated queries across your data lake
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Data Source
        </Button>
      </div>

      {/* Data sources table */}
      {isLoading ? (
        <DataSourcesTableSkeleton />
      ) : error ? (
        <Card>
          <CardContent className="py-8 text-center">
            <XCircle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-muted-foreground">
              Failed to load data sources. Please try again.
            </p>
          </CardContent>
        </Card>
      ) : dataSources && dataSources.length > 0 ? (
        <Card>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Catalog</TableHead>
                <TableHead>Connection</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-[100px]">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dataSources.map((ds) => (
                <TableRow key={ds.id}>
                  <TableCell className="font-medium">{ds.name}</TableCell>
                  <TableCell>
                    <TypeBadge type={ds.type} />
                  </TableCell>
                  <TableCell>
                    <code className="text-sm bg-muted px-1.5 py-0.5 rounded">
                      {ds.catalog_name}
                    </code>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {ds.host}:{ds.port}/{ds.database_name}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <StatusBadge status={ds.status} />
                      {ds.error_message && (
                        <span className="text-xs text-destructive truncate max-w-[200px]" title={ds.error_message}>
                          {ds.error_message}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="h-4 w-4" />
                          <span className="sr-only">Actions</span>
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() => handleTestConnection(ds.id)}
                          disabled={testingId === ds.id}
                        >
                          <Plug className="h-4 w-4 mr-2" />
                          {testingId === ds.id ? "Testing..." : "Test Connection"}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => setEditingSource(ds)}>
                          <Pencil className="h-4 w-4 mr-2" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-destructive"
                          onClick={() => setDeletingSource(ds)}
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
        </Card>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <DatabaseZap className="mx-auto h-12 w-12 text-muted-foreground" />
            <h3 className="mt-4 text-lg font-medium">No query data sources</h3>
            <p className="mt-2 text-muted-foreground max-w-md mx-auto">
              Connect external PostgreSQL or MySQL databases to run federated queries
              alongside your Iceberg data lake.
            </p>
            <Button className="mt-4" onClick={() => setCreateDialogOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Add Data Source
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Built-in source info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Built-in Iceberg Catalog</CardTitle>
          <CardDescription>
            Your Philotes data lake is always available as the <code className="bg-muted px-1 py-0.5 rounded text-xs">iceberg</code> catalog.
            External data sources added here appear as additional Trino catalogs in the query editor and schema browser.
          </CardDescription>
        </CardHeader>
      </Card>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Add Data Source</DialogTitle>
            <DialogDescription>
              Connect an external database for federated queries.
            </DialogDescription>
          </DialogHeader>
          <DataSourceForm
            onSubmit={handleCreate}
            isSubmitting={createMutation.isPending}
          />
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editingSource} onOpenChange={(open) => !open && setEditingSource(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit Data Source</DialogTitle>
            <DialogDescription>
              Update connection settings. Type and catalog name cannot be changed.
            </DialogDescription>
          </DialogHeader>
          {editingSource && (
            <DataSourceForm
              existingSource={editingSource}
              onSubmit={handleUpdate}
              isSubmitting={updateMutation.isPending}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <AlertDialog open={!!deletingSource} onOpenChange={(open) => !open && setDeletingSource(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Data Source</AlertDialogTitle>
            <AlertDialogDescription>
              This will remove the <strong>{deletingSource?.name}</strong> data source and drop the{" "}
              <code className="bg-muted px-1 py-0.5 rounded">{deletingSource?.catalog_name}</code>{" "}
              Trino catalog. Existing queries referencing this catalog will stop working.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
