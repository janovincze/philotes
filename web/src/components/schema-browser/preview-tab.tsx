"use client"

import { useEffect, useCallback } from "react"
import { useQueryExecute } from "@/lib/hooks/use-query"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { AlertCircle, RefreshCw } from "lucide-react"

interface PreviewTabProps {
  catalog: string
  schema: string
  table: string
}

const IDENTIFIER_RE = /^[A-Za-z_][A-Za-z0-9_]*$/

export function PreviewTab({ catalog, schema, table }: PreviewTabProps) {
  const mutation = useQueryExecute()

  const runPreview = useCallback(() => {
    if (!IDENTIFIER_RE.test(catalog) || !IDENTIFIER_RE.test(schema) || !IDENTIFIER_RE.test(table)) {
      return
    }
    mutation.mutate({
      sql: `SELECT * FROM ${catalog}.${schema}.${table} LIMIT 100`,
      limit: 100,
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [catalog, schema, table])

  useEffect(() => {
    runPreview()
  }, [runPreview])

  if (mutation.isPending) {
    return (
      <div className="space-y-2 p-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-6 w-full" />
        ))}
      </div>
    )
  }

  if (mutation.error || mutation.data?.error) {
    return (
      <div className="p-2 space-y-2">
        <div className="flex items-center gap-2 text-sm text-destructive">
          <AlertCircle className="h-4 w-4" />
          {mutation.data?.error || "Preview failed"}
        </div>
        <Button variant="outline" size="sm" className="h-7 text-xs" onClick={runPreview}>
          <RefreshCw className="h-3 w-3 mr-1" />
          Retry
        </Button>
      </div>
    )
  }

  const data = mutation.data
  if (!data?.columns?.length) {
    return <p className="p-2 text-sm text-muted-foreground">No data</p>
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-2 py-1 border-b">
        <span className="text-xs text-muted-foreground">
          {data.row_count} row{data.row_count !== 1 ? "s" : ""}
          {data.truncated ? " (truncated)" : ""}
        </span>
        <Button variant="ghost" size="sm" className="h-6 text-xs" onClick={runPreview} aria-label="Refresh preview">
          <RefreshCw className="h-3 w-3" />
        </Button>
      </div>
      <ScrollArea className="flex-1">
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b">
                {data.columns.map((col) => (
                  <th key={col.name} className="px-2 py-1 text-left font-medium text-muted-foreground whitespace-nowrap">
                    {col.name}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.rows.map((row, i) => (
                <tr key={i} className="border-b last:border-0 hover:bg-accent/50">
                  {data.columns.map((col) => (
                    <td key={col.name} className="px-2 py-1 whitespace-nowrap max-w-[200px] truncate font-mono">
                      {row[col.name] === null ? (
                        <span className="text-muted-foreground italic">null</span>
                      ) : (
                        String(row[col.name])
                      )}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </ScrollArea>
    </div>
  )
}
