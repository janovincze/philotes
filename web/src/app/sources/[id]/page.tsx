"use client"

import { useState } from "react"
import Link from "next/link"
import { useParams, useRouter } from "next/navigation"
import {
  ArrowLeft,
  CheckCircle,
  XCircle,
  AlertCircle,
  Database,
  Loader2,
  Trash2,
  Plug,
  KeyRound,
} from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import {
  useSource,
  useTestSourceConnection,
  useDeleteSource,
  useDiscoverTables,
} from "@/lib/hooks/use-sources"
import type { SourceStatus } from "@/lib/api/types"

function SourceStatusBadge({ status }: { status: SourceStatus }) {
  const config = {
    active: {
      icon: CheckCircle,
      variant: "default" as const,
      color: "text-green-500",
    },
    inactive: {
      icon: AlertCircle,
      variant: "secondary" as const,
      color: "text-muted-foreground",
    },
    error: {
      icon: XCircle,
      variant: "destructive" as const,
      color: "text-red-500",
    },
  }

  const { icon: Icon, variant, color } = config[status]

  return (
    <Badge variant={variant} className="gap-1">
      <Icon className={`h-3 w-3 ${color}`} />
      <span className="capitalize">{status}</span>
    </Badge>
  )
}

function SourceDetailSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Skeleton className="h-9 w-24" />
        <Skeleton className="h-8 w-48" />
      </div>
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-64" />
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            {[1, 2, 3, 4, 5, 6].map((i) => (
              <div key={i} className="space-y-1">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-5 w-40" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-40 w-full" />
        </CardContent>
      </Card>
    </div>
  )
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export default function SourceDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string

  const { data: source, isLoading, error } = useSource(id)
  const testConnection = useTestSourceConnection()
  const deleteSource = useDeleteSource()
  const {
    data: discoveredTables,
    isLoading: isDiscoveringTables,
  } = useDiscoverTables(id, "public")

  const [expandedTable, setExpandedTable] = useState<string | null>(null)

  const handleTestConnection = async () => {
    try {
      const result = await testConnection.mutateAsync(id)
      if (result.success) {
        toast.success("Connection successful", {
          description: result.message,
        })
      } else {
        toast.error("Connection failed", {
          description: result.message,
        })
      }
    } catch (err) {
      toast.error("Connection test failed", {
        description:
          err instanceof Error ? err.message : "An unexpected error occurred",
      })
    }
  }

  const handleDelete = async () => {
    try {
      await deleteSource.mutateAsync(id)
      toast.success("Source deleted", {
        description: "The data source has been removed.",
      })
      router.push("/sources")
    } catch (err) {
      toast.error("Failed to delete source", {
        description:
          err instanceof Error ? err.message : "An unexpected error occurred",
      })
    }
  }

  if (isLoading) {
    return <SourceDetailSkeleton />
  }

  if (error || !source) {
    return (
      <div className="space-y-6">
        <Button variant="outline" size="sm" asChild>
          <Link href="/sources">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Sources
          </Link>
        </Button>
        <Card>
          <CardContent className="py-8 text-center">
            <XCircle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-muted-foreground">
              Failed to load source. It may have been deleted or you may not
              have access.
            </p>
            <Button className="mt-4" variant="outline" asChild>
              <Link href="/sources">Return to Sources</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" asChild>
            <Link href="/sources">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back
            </Link>
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{source.name}</h1>
            <p className="text-muted-foreground">
              Source details and table discovery
            </p>
          </div>
        </div>
        <SourceStatusBadge status={source.status} />
      </div>

      {/* Source info card */}
      <Card>
        <CardHeader className="flex flex-row items-start justify-between space-y-0">
          <div className="flex items-start gap-4">
            <div className="rounded-lg bg-primary/10 p-2">
              <Database className="h-6 w-6 text-primary" />
            </div>
            <div>
              <CardTitle className="text-lg">Connection Details</CardTitle>
              <CardDescription>
                {source.host}:{source.port}/{source.database_name}
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">Host</p>
              <p className="text-sm">{source.host}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">Port</p>
              <p className="text-sm">{source.port}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Database
              </p>
              <p className="text-sm">{source.database_name}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Username
              </p>
              <p className="text-sm">{source.username}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                SSL Mode
              </p>
              <p className="text-sm capitalize">{source.ssl_mode || "disable"}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Status
              </p>
              <SourceStatusBadge status={source.status} />
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Created
              </p>
              <p className="text-sm">{formatDate(source.created_at)}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Updated
              </p>
              <p className="text-sm">{formatDate(source.updated_at)}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Actions card */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Actions</CardTitle>
          <CardDescription>
            Test the connection or manage this source
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-3">
            <Button
              variant="outline"
              onClick={handleTestConnection}
              disabled={testConnection.isPending}
            >
              {testConnection.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Testing...
                </>
              ) : (
                <>
                  <Plug className="mr-2 h-4 w-4" />
                  Test Connection
                </>
              )}
            </Button>

            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="destructive" disabled={deleteSource.isPending}>
                  {deleteSource.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Deleting...
                    </>
                  ) : (
                    <>
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete Source
                    </>
                  )}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete Source</AlertDialogTitle>
                  <AlertDialogDescription>
                    Are you sure you want to delete &quot;{source.name}&quot;?
                    This action cannot be undone. Any pipelines using this source
                    will also be affected.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction variant="destructive" onClick={handleDelete}>
                    Delete
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </CardContent>
      </Card>

      {/* Discovered tables */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Discovered Tables</CardTitle>
          <CardDescription>
            Tables found in the &quot;public&quot; schema of this database
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isDiscoveringTables ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : discoveredTables && discoveredTables.tables.length > 0 ? (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                {discoveredTables.count} table{discoveredTables.count !== 1 ? "s" : ""} found
              </p>
              {discoveredTables.tables.map((table) => {
                const tableKey = `${table.schema}.${table.name}`
                const isExpanded = expandedTable === tableKey

                return (
                  <div key={tableKey} className="rounded-lg border">
                    <button
                      type="button"
                      className="flex w-full items-center justify-between p-4 text-left hover:bg-muted/50 transition-colors"
                      onClick={() =>
                        setExpandedTable(isExpanded ? null : tableKey)
                      }
                    >
                      <div className="flex items-center gap-3">
                        <Database className="h-4 w-4 text-muted-foreground" />
                        <div>
                          <p className="text-sm font-medium">{table.name}</p>
                          <p className="text-xs text-muted-foreground">
                            {table.schema} &middot; {table.columns.length} column
                            {table.columns.length !== 1 ? "s" : ""}
                          </p>
                        </div>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {isExpanded ? "Hide columns" : "Show columns"}
                      </span>
                    </button>

                    {isExpanded && (
                      <div className="border-t">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Column</TableHead>
                              <TableHead>Type</TableHead>
                              <TableHead>Nullable</TableHead>
                              <TableHead>Primary Key</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {table.columns.map((column) => (
                              <TableRow key={column.name}>
                                <TableCell className="font-medium">
                                  <div className="flex items-center gap-2">
                                    {column.primary_key && (
                                      <KeyRound className="h-3 w-3 text-amber-500" />
                                    )}
                                    {column.name}
                                  </div>
                                </TableCell>
                                <TableCell>
                                  <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                                    {column.type}
                                  </code>
                                </TableCell>
                                <TableCell>
                                  {column.nullable ? (
                                    <Badge variant="secondary" className="text-xs">
                                      nullable
                                    </Badge>
                                  ) : (
                                    <Badge variant="outline" className="text-xs">
                                      not null
                                    </Badge>
                                  )}
                                </TableCell>
                                <TableCell>
                                  {column.primary_key ? (
                                    <Badge variant="default" className="text-xs gap-1">
                                      <KeyRound className="h-3 w-3" />
                                      PK
                                    </Badge>
                                  ) : (
                                    <span className="text-muted-foreground">-</span>
                                  )}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          ) : (
            <div className="py-8 text-center">
              <Database className="mx-auto h-8 w-8 text-muted-foreground" />
              <p className="mt-2 text-sm text-muted-foreground">
                No tables found in the public schema, or the connection could
                not be established.
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
