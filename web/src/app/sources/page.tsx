"use client"

import Link from "next/link"
import { Database, Plus, CheckCircle, XCircle, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useSources } from "@/lib/hooks/use-sources"
import type { SourceStatus } from "@/lib/api/types"

function SourceStatusBadge({ status }: { status: SourceStatus }) {
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

function SourceCard({
  source,
}: {
  source: { id: string; name: string; host: string; port: number; database_name: string; status: SourceStatus }
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between space-y-0">
        <div className="flex items-start gap-4">
          <div className="rounded-lg bg-primary/10 p-2">
            <Database className="h-6 w-6 text-primary" />
          </div>
          <div>
            <CardTitle className="text-lg">{source.name}</CardTitle>
            <CardDescription>
              {source.host}:{source.port}/{source.database_name}
            </CardDescription>
          </div>
        </div>
        <SourceStatusBadge status={source.status} />
      </CardHeader>
      <CardContent>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" asChild>
            <Link href={`/sources/${source.id}`}>View Details</Link>
          </Button>
          <Button variant="outline" size="sm">
            Test Connection
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function SourcesListSkeleton() {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {[1, 2, 3, 4].map((i) => (
        <Card key={i}>
          <CardHeader className="flex flex-row items-start gap-4">
            <Skeleton className="h-10 w-10 rounded-lg" />
            <div className="space-y-2">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-48" />
            </div>
          </CardHeader>
          <CardContent>
            <Skeleton className="h-9 w-24" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export default function SourcesPage() {
  const { data: sources, isLoading, error } = useSources()

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Data Sources</h1>
          <p className="text-muted-foreground">
            Manage your PostgreSQL database connections
          </p>
        </div>
        <Button asChild>
          <Link href="/sources/new">
            <Plus className="mr-2 h-4 w-4" />
            Add Source
          </Link>
        </Button>
      </div>

      {/* Sources list */}
      {isLoading ? (
        <SourcesListSkeleton />
      ) : error ? (
        <Card>
          <CardContent className="py-8 text-center">
            <XCircle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-muted-foreground">
              Failed to load sources. Please try again.
            </p>
          </CardContent>
        </Card>
      ) : sources && sources.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2">
          {sources.map((source) => (
            <SourceCard key={source.id} source={source} />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <Database className="mx-auto h-12 w-12 text-muted-foreground" />
            <h3 className="mt-4 text-lg font-medium">No data sources</h3>
            <p className="mt-2 text-muted-foreground">
              Add your first PostgreSQL database to start replicating data.
            </p>
            <Button className="mt-4" asChild>
              <Link href="/sources/new">
                <Plus className="mr-2 h-4 w-4" />
                Add Source
              </Link>
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
