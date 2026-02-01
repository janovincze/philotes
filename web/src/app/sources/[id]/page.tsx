"use client"

import { use, useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import {
  ArrowLeft,
  Database,
  CheckCircle,
  XCircle,
  AlertCircle,
  Loader2,
  Trash2,
  Edit,
  RefreshCw,
} from "lucide-react"
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
  useDeleteSource,
  useTestSourceConnection,
  useDiscoverTables,
} from "@/lib/hooks/use-sources"
import { usePipelines } from "@/lib/hooks/use-pipelines"
import type { SourceStatus } from "@/lib/api/types"

interface PageProps {
  params: Promise<{ id: string }>
}

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

function ConnectionInfo({
  source,
  isLoading,
}: {
  source?: {
    host: string
    port: number
    database_name: string
    username: string
    ssl_mode: string
    created_at: string
    updated_at: string
  }
  isLoading?: boolean
}) {
  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i}>
            <Skeleton className="h-4 w-20 mb-1" />
            <Skeleton className="h-5 w-32" />
          </div>
        ))}
      </div>
    )
  }

  if (!source) return null

  const fields = [
    { label: "Host", value: source.host },
    { label: "Port", value: source.port.toString() },
    { label: "Database", value: source.database_name },
    { label: "Username", value: source.username },
    { label: "SSL Mode", value: source.ssl_mode },
    {
      label: "Created",
      value: new Date(source.created_at).toLocaleDateString(),
    },
  ]

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {fields.map(({ label, value }) => (
        <div key={label}>
          <p className="text-sm text-muted-foreground">{label}</p>
          <p className="font-mono">{value}</p>
        </div>
      ))}
    </div>
  )
}

function ConnectionTestCard({ sourceId }: { sourceId: string }) {
  const testConnection = useTestSourceConnection()
  const [lastResult, setLastResult] = useState<{
    success: boolean
    message: string
    latency?: number
  } | null>(null)

  const handleTest = async () => {
    const result = await testConnection.mutateAsync(sourceId)
    setLastResult({
      success: result.success,
      message: result.message,
      latency: result.latency_ms,
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Connection Test</CardTitle>
        <CardDescription>
          Verify the database connection is working
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Button
          onClick={handleTest}
          disabled={testConnection.isPending}
          variant="outline"
        >
          {testConnection.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Testing...
            </>
          ) : (
            <>
              <RefreshCw className="mr-2 h-4 w-4" />
              Test Connection
            </>
          )}
        </Button>

        {lastResult && (
          <div
            className={`rounded-md p-3 text-sm ${
              lastResult.success
                ? "bg-green-500/10 text-green-700 dark:text-green-400"
                : "bg-destructive/10 text-destructive"
            }`}
          >
            <div className="flex items-center gap-2">
              {lastResult.success ? (
                <CheckCircle className="h-4 w-4" />
              ) : (
                <XCircle className="h-4 w-4" />
              )}
              <span>{lastResult.message}</span>
            </div>
            {lastResult.success && lastResult.latency && (
              <p className="mt-1 text-xs opacity-80">
                Latency: {lastResult.latency}ms
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function TablesCard({ sourceId }: { sourceId: string }) {
  const { data: tables, isLoading, error } = useDiscoverTables(sourceId)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Available Tables</CardTitle>
        <CardDescription>
          Tables discovered in the source database
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        ) : error ? (
          <p className="text-sm text-muted-foreground">
            Failed to discover tables. Test the connection first.
          </p>
        ) : tables && tables.tables.length > 0 ? (
          <div className="space-y-1">
            {tables.tables.slice(0, 10).map((table) => (
              <div
                key={`${table.schema}.${table.name}`}
                className="flex items-center justify-between rounded-md border px-3 py-2 text-sm"
              >
                <span className="font-mono">
                  {table.schema}.{table.name}
                </span>
                <Badge variant="outline" className="text-xs">
                  {table.columns?.length ?? "?"} cols
                </Badge>
              </div>
            ))}
            {tables.tables.length > 10 && (
              <p className="pt-2 text-center text-sm text-muted-foreground">
                +{tables.tables.length - 10} more tables
              </p>
            )}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            No tables found in the database
          </p>
        )}
      </CardContent>
    </Card>
  )
}

function AssociatedPipelines({ sourceId }: { sourceId: string }) {
  const { data: pipelines, isLoading } = usePipelines()

  const associatedPipelines = pipelines?.filter(
    (p) => p.source_id === sourceId
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Associated Pipelines</CardTitle>
        <CardDescription>
          Pipelines using this data source
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {[1, 2].map((i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : associatedPipelines && associatedPipelines.length > 0 ? (
          <div className="space-y-2">
            {associatedPipelines.map((pipeline) => (
              <Link
                key={pipeline.id}
                href={`/pipelines/${pipeline.id}`}
                className="flex items-center justify-between rounded-md border px-3 py-2 hover:bg-muted/50"
              >
                <span className="font-medium">{pipeline.name}</span>
                <Badge
                  variant={
                    pipeline.status === "running" ? "default" : "secondary"
                  }
                >
                  {pipeline.status}
                </Badge>
              </Link>
            ))}
          </div>
        ) : (
          <div className="text-center py-4">
            <p className="text-sm text-muted-foreground mb-3">
              No pipelines are using this source
            </p>
            <Button asChild variant="outline" size="sm">
              <Link href="/pipelines">Create Pipeline</Link>
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function SourceDetailPage({ params }: PageProps) {
  const { id } = use(params)
  const router = useRouter()
  const { data: source, isLoading, error } = useSource(id)
  const deleteSource = useDeleteSource()

  const handleDelete = async () => {
    await deleteSource.mutateAsync(id)
    router.push("/sources")
  }

  if (error) {
    return (
      <div className="space-y-6">
        <Link
          href="/sources"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Sources
        </Link>
        <Card>
          <CardContent className="py-12 text-center">
            <XCircle className="mx-auto h-12 w-12 text-destructive" />
            <h3 className="mt-4 text-lg font-medium">Source not found</h3>
            <p className="mt-2 text-muted-foreground">
              This source may have been deleted or you may not have access.
            </p>
            <Button asChild className="mt-4">
              <Link href="/sources">Return to Sources</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Link
        href="/sources"
        className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Sources
      </Link>

      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          {isLoading ? (
            <>
              <Skeleton className="h-14 w-14 rounded-lg" />
              <div className="space-y-2">
                <Skeleton className="h-7 w-48" />
                <Skeleton className="h-5 w-32" />
              </div>
            </>
          ) : (
            <>
              <div className="rounded-lg bg-primary/10 p-3">
                <Database className="h-8 w-8 text-primary" />
              </div>
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-2xl font-bold">{source?.name}</h1>
                  {source && <SourceStatusBadge status={source.status} />}
                </div>
                <p className="text-muted-foreground">
                  {source?.host}:{source?.port}/{source?.database_name}
                </p>
              </div>
            </>
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" asChild>
            <Link href={`/sources/${id}/edit`}>
              <Edit className="mr-2 h-4 w-4" />
              Edit
            </Link>
          </Button>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete source?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete the source configuration. Any
                  pipelines using this source will stop working.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleDelete}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  {deleteSource.isPending ? "Deleting..." : "Delete"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      {/* Main content grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Connection details */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Connection Details</CardTitle>
            <CardDescription>
              Database connection configuration
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ConnectionInfo source={source} isLoading={isLoading} />
          </CardContent>
        </Card>

        {/* Connection test */}
        <ConnectionTestCard sourceId={id} />
      </div>

      {/* Tables and pipelines */}
      <div className="grid gap-6 lg:grid-cols-2">
        <TablesCard sourceId={id} />
        <AssociatedPipelines sourceId={id} />
      </div>
    </div>
  )
}
