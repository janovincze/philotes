"use client"

import { useTableInfo } from "@/lib/hooks/use-query"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { ScrollArea } from "@/components/ui/scroll-area"
import { AlertCircle } from "lucide-react"

interface ColumnsTabProps {
  catalog: string
  schema: string
  table: string
  onInsertColumn: (columnName: string) => void
}

function columnTypeCategory(type: string): string {
  const t = type.toLowerCase()
  if (t.includes("int") || t.includes("decimal") || t.includes("float") || t.includes("double") || t.includes("real")) return "numeric"
  if (t.includes("varchar") || t.includes("char") || t.includes("text")) return "text"
  if (t.includes("date") || t.includes("time") || t.includes("timestamp")) return "datetime"
  if (t.includes("bool")) return "boolean"
  if (t.includes("json")) return "json"
  return "other"
}

function typeVariant(category: string): "default" | "secondary" | "outline" | "destructive" {
  switch (category) {
    case "numeric": return "default"
    case "text": return "secondary"
    case "datetime": return "outline"
    default: return "secondary"
  }
}

export function ColumnsTab({ catalog, schema, table, onInsertColumn }: ColumnsTabProps) {
  const { data, isLoading, error } = useTableInfo(catalog, schema, table)

  if (isLoading) {
    return (
      <div className="space-y-2 p-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-6 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 p-2 text-sm text-destructive">
        <AlertCircle className="h-4 w-4" />
        Failed to load columns
      </div>
    )
  }

  if (!data?.columns?.length) {
    return <p className="p-2 text-sm text-muted-foreground">No columns found</p>
  }

  return (
    <ScrollArea className="h-full">
      <div className="space-y-0.5 p-1">
        {data.columns.map((col) => {
          const category = columnTypeCategory(col.type)
          return (
            <button
              key={col.name}
              className="flex items-center gap-2 w-full rounded px-2 py-1 text-left text-sm hover:bg-accent transition-colors"
              onClick={() => onInsertColumn(col.name)}
              title={`Click to insert "${col.name}"`}
            >
              <span className="font-mono text-xs flex-1 truncate">{col.name}</span>
              <Badge variant={typeVariant(category)} className="text-[10px] px-1.5 py-0 h-4 font-normal shrink-0">
                {col.type}
              </Badge>
              {col.nullable === false && (
                <span className="text-[10px] text-muted-foreground shrink-0">NOT NULL</span>
              )}
            </button>
          )
        })}
      </div>
    </ScrollArea>
  )
}
