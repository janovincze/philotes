"use client"

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { X } from "lucide-react"
import { ColumnsTab } from "./columns-tab"
import { PreviewTab } from "./preview-tab"
import { DdlTab } from "./ddl-tab"

interface TableDetailsProps {
  catalog: string
  schema: string
  table: string
  onInsertSql: (sql: string) => void
  onInsertColumn: (columnName: string) => void
  onClose: () => void
}

export function TableDetails({ catalog, schema, table, onInsertSql, onInsertColumn, onClose }: TableDetailsProps) {
  return (
    <div className="flex flex-col h-full border-t">
      <div className="flex items-center justify-between px-2 py-1 bg-muted/50">
        <span className="text-xs font-medium truncate" title={`${catalog}.${schema}.${table}`}>
          {schema}.{table}
        </span>
        <Button variant="ghost" size="icon" className="h-5 w-5" onClick={onClose} aria-label="Close table details">
          <X className="h-3 w-3" />
        </Button>
      </div>
      <Tabs defaultValue="columns" className="flex-1 flex flex-col min-h-0">
        <TabsList className="h-7 w-full justify-start rounded-none border-b bg-transparent px-1">
          <TabsTrigger value="columns" className="h-6 text-xs px-2 data-[state=active]:bg-background">
            Columns
          </TabsTrigger>
          <TabsTrigger value="preview" className="h-6 text-xs px-2 data-[state=active]:bg-background">
            Preview
          </TabsTrigger>
          <TabsTrigger value="ddl" className="h-6 text-xs px-2 data-[state=active]:bg-background">
            DDL
          </TabsTrigger>
        </TabsList>
        <TabsContent value="columns" className="flex-1 min-h-0 mt-0">
          <ColumnsTab catalog={catalog} schema={schema} table={table} onInsertColumn={onInsertColumn} />
        </TabsContent>
        <TabsContent value="preview" className="flex-1 min-h-0 mt-0">
          <PreviewTab catalog={catalog} schema={schema} table={table} />
        </TabsContent>
        <TabsContent value="ddl" className="flex-1 min-h-0 mt-0">
          <DdlTab catalog={catalog} schema={schema} table={table} onInsertSql={onInsertSql} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
