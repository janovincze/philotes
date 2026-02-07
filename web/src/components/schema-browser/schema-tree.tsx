"use client"

import { useState, useMemo } from "react"
import { ChevronRight, Database, Folder, FolderOpen, Table2, Star, Copy, Play } from "lucide-react"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useCatalogs, useSchemas, useTables, useTableInfo } from "@/lib/hooks/use-query"
import { toast } from "sonner"
import { cn, quoteIdent } from "@/lib/utils"

interface SchemaTreeProps {
  searchQuery: string
  showFavoritesOnly: boolean
  favorites: Set<string>
  onToggleFavorite: (qualifiedName: string) => void
  onSelectTable: (catalog: string, schema: string, table: string) => void
  onInsertSql: (sql: string) => void
  onInsertColumn: (columnName: string) => void
  selectedTable: { catalog: string; schema: string; table: string } | null
}

export function SchemaTree({
  searchQuery,
  showFavoritesOnly,
  favorites,
  onToggleFavorite,
  onSelectTable,
  onInsertSql,
  onInsertColumn,
  selectedTable,
}: SchemaTreeProps) {
  const { data: catalogData, isLoading, error } = useCatalogs()

  if (isLoading) {
    return (
      <div className="space-y-1 p-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-6 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <p className="p-2 text-sm text-destructive">Failed to load catalogs</p>
    )
  }

  const catalogs = catalogData?.catalogs ?? []

  if (catalogs.length === 0) {
    return (
      <p className="p-2 text-sm text-muted-foreground">
        No catalogs found. Check your Trino connection.
      </p>
    )
  }

  return (
    <div className="space-y-0.5">
      {catalogs.map((cat) => (
        <CatalogNode
          key={cat.name}
          name={cat.name}
          searchQuery={searchQuery}
          showFavoritesOnly={showFavoritesOnly}
          favorites={favorites}
          onToggleFavorite={onToggleFavorite}
          onSelectTable={onSelectTable}
          onInsertSql={onInsertSql}
          onInsertColumn={onInsertColumn}
          selectedTable={selectedTable}
        />
      ))}
    </div>
  )
}

interface CatalogNodeProps {
  name: string
  searchQuery: string
  showFavoritesOnly: boolean
  favorites: Set<string>
  onToggleFavorite: (qualifiedName: string) => void
  onSelectTable: (catalog: string, schema: string, table: string) => void
  onInsertSql: (sql: string) => void
  onInsertColumn: (columnName: string) => void
  selectedTable: { catalog: string; schema: string; table: string } | null
}

function CatalogNode({
  name,
  searchQuery,
  showFavoritesOnly,
  favorites,
  onToggleFavorite,
  onSelectTable,
  onInsertSql,
  onInsertColumn,
  selectedTable,
}: CatalogNodeProps) {
  const [open, setOpen] = useState(false)
  const shouldAutoExpand = searchQuery.length > 0 || showFavoritesOnly
  const isOpen = open || shouldAutoExpand

  return (
    <Collapsible open={isOpen} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex items-center gap-1 w-full px-1 py-0.5 text-sm hover:bg-accent rounded transition-colors">
        <ChevronRight className={cn("h-3.5 w-3.5 shrink-0 transition-transform", isOpen && "rotate-90")} />
        <Database className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate font-medium">{name}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        {isOpen && (
          <div className="ml-3 border-l pl-1">
            <SchemaNodes
              catalog={name}
              searchQuery={searchQuery}
              showFavoritesOnly={showFavoritesOnly}
              favorites={favorites}
              onToggleFavorite={onToggleFavorite}
              onSelectTable={onSelectTable}
              onInsertSql={onInsertSql}
              onInsertColumn={onInsertColumn}
              selectedTable={selectedTable}
            />
          </div>
        )}
      </CollapsibleContent>
    </Collapsible>
  )
}

interface SchemaNodesProps {
  catalog: string
  searchQuery: string
  showFavoritesOnly: boolean
  favorites: Set<string>
  onToggleFavorite: (qualifiedName: string) => void
  onSelectTable: (catalog: string, schema: string, table: string) => void
  onInsertSql: (sql: string) => void
  onInsertColumn: (columnName: string) => void
  selectedTable: { catalog: string; schema: string; table: string } | null
}

function SchemaNodes({ catalog, searchQuery, showFavoritesOnly, favorites, onToggleFavorite, onSelectTable, onInsertSql, onInsertColumn, selectedTable }: SchemaNodesProps) {
  const { data, isLoading } = useSchemas(catalog)

  if (isLoading) {
    return (
      <div className="space-y-1 py-1">
        {Array.from({ length: 2 }).map((_, i) => (
          <Skeleton key={i} className="h-5 w-full" />
        ))}
      </div>
    )
  }

  const schemas = (data?.schemas ?? []).filter((s) => s.name !== "information_schema")

  if (schemas.length === 0) {
    return <p className="px-1 py-0.5 text-xs text-muted-foreground">No schemas</p>
  }

  return (
    <>
      {schemas.map((s) => (
        <SchemaNode
          key={s.name}
          catalog={catalog}
          name={s.name}
          searchQuery={searchQuery}
          showFavoritesOnly={showFavoritesOnly}
          favorites={favorites}
          onToggleFavorite={onToggleFavorite}
          onSelectTable={onSelectTable}
          onInsertSql={onInsertSql}
          onInsertColumn={onInsertColumn}
          selectedTable={selectedTable}
        />
      ))}
    </>
  )
}

interface SchemaNodeProps {
  catalog: string
  name: string
  searchQuery: string
  showFavoritesOnly: boolean
  favorites: Set<string>
  onToggleFavorite: (qualifiedName: string) => void
  onSelectTable: (catalog: string, schema: string, table: string) => void
  onInsertSql: (sql: string) => void
  onInsertColumn: (columnName: string) => void
  selectedTable: { catalog: string; schema: string; table: string } | null
}

function SchemaNode({
  catalog,
  name,
  searchQuery,
  showFavoritesOnly,
  favorites,
  onToggleFavorite,
  onSelectTable,
  onInsertSql,
  onInsertColumn,
  selectedTable,
}: SchemaNodeProps) {
  const [open, setOpen] = useState(false)
  const shouldAutoExpand = searchQuery.length > 0 || showFavoritesOnly
  const isOpen = open || shouldAutoExpand

  return (
    <Collapsible open={isOpen} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex items-center gap-1 w-full px-1 py-0.5 text-sm hover:bg-accent rounded transition-colors">
        <ChevronRight className={cn("h-3.5 w-3.5 shrink-0 transition-transform", isOpen && "rotate-90")} />
        {isOpen ? (
          <FolderOpen className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        ) : (
          <Folder className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        )}
        <span className="truncate">{name}</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        {isOpen && (
          <div className="ml-3 border-l pl-1">
            <TableNodes
              catalog={catalog}
              schema={name}
              searchQuery={searchQuery}
              showFavoritesOnly={showFavoritesOnly}
              favorites={favorites}
              onToggleFavorite={onToggleFavorite}
              onSelectTable={onSelectTable}
              onInsertSql={onInsertSql}
              onInsertColumn={onInsertColumn}
              selectedTable={selectedTable}
            />
          </div>
        )}
      </CollapsibleContent>
    </Collapsible>
  )
}

interface TableNodesProps {
  catalog: string
  schema: string
  searchQuery: string
  showFavoritesOnly: boolean
  favorites: Set<string>
  onToggleFavorite: (qualifiedName: string) => void
  onSelectTable: (catalog: string, schema: string, table: string) => void
  onInsertSql: (sql: string) => void
  onInsertColumn: (columnName: string) => void
  selectedTable: { catalog: string; schema: string; table: string } | null
}

function TableNodes({ catalog, schema, searchQuery, showFavoritesOnly, favorites, onToggleFavorite, onSelectTable, onInsertSql, onInsertColumn, selectedTable }: TableNodesProps) {
  const { data, isLoading } = useTables(catalog, schema)

  const filteredTables = useMemo(() => {
    let tables = data?.tables ?? []
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      tables = tables.filter((t) => t.name.toLowerCase().includes(q))
    }
    if (showFavoritesOnly) {
      tables = tables.filter((t) => favorites.has(`${catalog}.${schema}.${t.name}`))
    }
    return tables
  }, [data, searchQuery, showFavoritesOnly, favorites, catalog, schema])

  if (isLoading) {
    return (
      <div className="space-y-1 py-1">
        {Array.from({ length: 2 }).map((_, i) => (
          <Skeleton key={i} className="h-5 w-full" />
        ))}
      </div>
    )
  }

  if (filteredTables.length === 0) {
    return <p className="px-1 py-0.5 text-xs text-muted-foreground">No tables</p>
  }

  return (
    <>
      {filteredTables.map((t) => (
        <TableNode
          key={t.name}
          catalog={catalog}
          schema={schema}
          name={t.name}
          isFavorite={favorites.has(`${catalog}.${schema}.${t.name}`)}
          isSelected={
            selectedTable?.catalog === catalog &&
            selectedTable?.schema === schema &&
            selectedTable?.table === t.name
          }
          onToggleFavorite={onToggleFavorite}
          onSelectTable={onSelectTable}
          onInsertSql={onInsertSql}
          onInsertColumn={onInsertColumn}
        />
      ))}
    </>
  )
}

interface TableNodeProps {
  catalog: string
  schema: string
  name: string
  isFavorite: boolean
  isSelected: boolean
  onToggleFavorite: (qualifiedName: string) => void
  onSelectTable: (catalog: string, schema: string, table: string) => void
  onInsertSql: (sql: string) => void
  onInsertColumn: (columnName: string) => void
}

function TableNode({
  catalog,
  schema,
  name,
  isFavorite,
  isSelected,
  onToggleFavorite,
  onSelectTable,
  onInsertSql,
  onInsertColumn,
}: TableNodeProps) {
  const [open, setOpen] = useState(false)
  const qualifiedName = `${catalog}.${schema}.${name}`
  const quotedName = `${quoteIdent(catalog)}.${quoteIdent(schema)}.${quoteIdent(name)}`

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div className={cn("group flex items-center gap-0.5 rounded transition-colors", isSelected && "bg-accent")}>
        <CollapsibleTrigger className="flex items-center gap-1 py-0.5 pl-1 hover:bg-accent rounded-l transition-colors" onClick={(e) => e.stopPropagation()}>
          <ChevronRight className={cn("h-3 w-3 shrink-0 transition-transform", open && "rotate-90")} />
        </CollapsibleTrigger>
        <button
          type="button"
          className="flex items-center gap-1 flex-1 py-0.5 text-sm hover:bg-accent rounded-r transition-colors min-w-0"
          onClick={() => onSelectTable(catalog, schema, name)}
        >
          <Table2 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
          <span className="truncate">{name}</span>
        </button>
        <button
          type="button"
          className="shrink-0 p-0.5 hover:bg-accent rounded"
          aria-label={isFavorite ? "Remove from favorites" : "Add to favorites"}
          onClick={(e) => {
            e.stopPropagation()
            onToggleFavorite(qualifiedName)
          }}
        >
          <Star className={cn("h-3 w-3", isFavorite ? "fill-yellow-400 text-yellow-400" : "text-muted-foreground opacity-0 group-hover:opacity-100")} />
        </button>
        <button
          type="button"
          className="shrink-0 p-0.5 hover:bg-accent rounded opacity-0 group-hover:opacity-100"
          aria-label="Generate SELECT query"
          onClick={(e) => {
            e.stopPropagation()
            onInsertSql(`SELECT * FROM ${quotedName} LIMIT 100`)
          }}
          title="Generate SELECT"
        >
          <Play className="h-3 w-3 text-muted-foreground" />
        </button>
        <button
          type="button"
          className="shrink-0 p-0.5 mr-1 hover:bg-accent rounded opacity-0 group-hover:opacity-100"
          aria-label="Copy qualified name"
          onClick={(e) => {
            e.stopPropagation()
            navigator.clipboard.writeText(qualifiedName).then(
              () => toast.success("Copied to clipboard"),
              () => toast.error("Failed to copy")
            )
          }}
          title="Copy qualified name"
        >
          <Copy className="h-3 w-3 text-muted-foreground" />
        </button>
      </div>
      <CollapsibleContent>
        {open && (
          <div className="ml-5 border-l pl-1">
            <ColumnLeaves
              catalog={catalog}
              schema={schema}
              table={name}
              onInsertColumn={onInsertColumn}
            />
          </div>
        )}
      </CollapsibleContent>
    </Collapsible>
  )
}

interface ColumnLeavesProps {
  catalog: string
  schema: string
  table: string
  onInsertColumn: (columnName: string) => void
}

function ColumnLeaves({ catalog, schema, table, onInsertColumn }: ColumnLeavesProps) {
  const { data, isLoading } = useTableInfo(catalog, schema, table)

  if (isLoading) {
    return (
      <div className="space-y-0.5 py-0.5">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-4 w-full" />
        ))}
      </div>
    )
  }

  const columns = data?.columns ?? []
  if (columns.length === 0) {
    return <p className="px-1 text-xs text-muted-foreground">No columns</p>
  }

  return (
    <div className="space-y-0">
      {columns.map((col) => (
        <button
          type="button"
          key={col.name}
          className="flex items-center gap-1.5 w-full px-1 py-0.5 text-xs hover:bg-accent rounded transition-colors"
          onClick={() => onInsertColumn(col.name)}
          title={`Click to insert "${col.name}"`}
        >
          <span className="font-mono truncate">{col.name}</span>
          <Badge variant="secondary" className="text-[9px] px-1 py-0 h-3.5 font-normal shrink-0">
            {col.type}
          </Badge>
        </button>
      ))}
    </div>
  )
}
