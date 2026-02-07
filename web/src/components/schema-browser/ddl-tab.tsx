"use client"

import { useMemo } from "react"
import { useTableInfo } from "@/lib/hooks/use-query"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Copy, FileCode, AlertCircle } from "lucide-react"
import { toast } from "sonner"

interface DdlTabProps {
  catalog: string
  schema: string
  table: string
  onInsertSql: (sql: string) => void
}

export function DdlTab({ catalog, schema, table, onInsertSql }: DdlTabProps) {
  const { data, isLoading, error } = useTableInfo(catalog, schema, table)

  const ddl = useMemo(() => {
    if (!data?.columns?.length) return ""

    const qualifiedName = `${catalog}.${schema}.${table}`
    const columnDefs = data.columns.map((col) => {
      let def = `  ${col.name} ${col.type}`
      if (col.nullable === false) def += " NOT NULL"
      if (col.comment) def += ` COMMENT '${col.comment.replace(/'/g, "''")}'`
      return def
    })

    return `CREATE TABLE ${qualifiedName} (\n${columnDefs.join(",\n")}\n);`
  }, [data, catalog, schema, table])

  if (isLoading) {
    return (
      <div className="space-y-2 p-2">
        {Array.from({ length: 8 }).map((_, i) => (
          <Skeleton key={i} className="h-4 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 p-2 text-sm text-destructive">
        <AlertCircle className="h-4 w-4" />
        Failed to generate DDL
      </div>
    )
  }

  if (!ddl) {
    return <p className="p-2 text-sm text-muted-foreground">No column data available</p>
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 p-1 border-b">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs"
          onClick={() => {
            navigator.clipboard.writeText(ddl).then(
              () => toast.success("DDL copied to clipboard"),
              () => toast.error("Failed to copy DDL")
            )
          }}
        >
          <Copy className="h-3 w-3 mr-1" />
          Copy
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs"
          onClick={() => onInsertSql(ddl)}
        >
          <FileCode className="h-3 w-3 mr-1" />
          Insert
        </Button>
      </div>
      <ScrollArea className="flex-1">
        <pre className="p-2 text-xs font-mono whitespace-pre-wrap">{ddl}</pre>
      </ScrollArea>
    </div>
  )
}
