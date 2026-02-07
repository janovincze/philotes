"use client"

import { useState, useEffect, useRef, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { Database, RefreshCw, Star, PanelLeftClose } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { SchemaSearch } from "./schema-search"
import { SchemaTree } from "./schema-tree"
import { TableDetails } from "./table-details"
import { cn } from "@/lib/utils"

interface SchemaBrowserProps {
  onInsertSql: (sql: string) => void
  isOpen: boolean
  onToggle: () => void
}

const FAVORITES_KEY = "philotes-schema-favorites"
const MIN_WIDTH = 200
const MAX_WIDTH = 500
const DEFAULT_WIDTH = 300

export function SchemaBrowser({ onInsertSql, isOpen, onToggle }: SchemaBrowserProps) {
  const queryClient = useQueryClient()
  const [searchQuery, setSearchQuery] = useState("")
  const [showFavoritesOnly, setShowFavoritesOnly] = useState(false)
  const [selectedTable, setSelectedTable] = useState<{ catalog: string; schema: string; table: string } | null>(null)
  const [sidebarWidth, setSidebarWidth] = useState(DEFAULT_WIDTH)
  const isResizing = useRef(false)
  const sidebarRef = useRef<HTMLDivElement>(null)

  // Favorites persistence
  const [favorites, setFavorites] = useState<Set<string>>(() => {
    if (typeof window === "undefined") return new Set()
    try {
      const stored = localStorage.getItem(FAVORITES_KEY)
      return stored ? new Set(JSON.parse(stored) as string[]) : new Set()
    } catch {
      return new Set()
    }
  })

  useEffect(() => {
    localStorage.setItem(FAVORITES_KEY, JSON.stringify([...favorites]))
  }, [favorites])

  const handleToggleFavorite = useCallback((qualifiedName: string) => {
    setFavorites((prev) => {
      const next = new Set(prev)
      if (next.has(qualifiedName)) {
        next.delete(qualifiedName)
      } else {
        next.add(qualifiedName)
      }
      return next
    })
  }, [])

  const handleSelectTable = useCallback((catalog: string, schema: string, table: string) => {
    setSelectedTable((prev) =>
      prev?.catalog === catalog && prev?.schema === schema && prev?.table === table
        ? prev
        : { catalog, schema, table }
    )
  }, [])

  const handleInsertColumn = useCallback((columnName: string) => {
    onInsertSql(columnName)
  }, [onInsertSql])

  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["query"] })
  }, [queryClient])

  // Resize handling
  const handleMouseDown = useCallback(() => {
    isResizing.current = true
    document.body.style.cursor = "col-resize"
    document.body.style.userSelect = "none"
  }, [])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing.current || !sidebarRef.current) return
      const rect = sidebarRef.current.getBoundingClientRect()
      const newWidth = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, e.clientX - rect.left))
      setSidebarWidth(newWidth)
    }
    const handleMouseUp = () => {
      if (isResizing.current) {
        isResizing.current = false
        document.body.style.cursor = ""
        document.body.style.userSelect = ""
      }
    }

    document.addEventListener("mousemove", handleMouseMove)
    document.addEventListener("mouseup", handleMouseUp)
    return () => {
      document.removeEventListener("mousemove", handleMouseMove)
      document.removeEventListener("mouseup", handleMouseUp)
      document.body.style.cursor = ""
      document.body.style.userSelect = ""
    }
  }, [])

  if (!isOpen) return null

  return (
    <div
      ref={sidebarRef}
      className="relative flex flex-col border-r bg-background shrink-0"
      style={{ width: sidebarWidth }}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-2 py-1.5 border-b">
        <div className="flex items-center gap-1.5">
          <Database className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm font-medium">Schema</span>
        </div>
        <div className="flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon"
            className={cn("h-6 w-6", showFavoritesOnly && "text-yellow-500")}
            onClick={() => setShowFavoritesOnly(!showFavoritesOnly)}
            title={showFavoritesOnly ? "Show all" : "Show favorites only"}
          >
            <Star className={cn("h-3.5 w-3.5", showFavoritesOnly && "fill-yellow-400")} />
          </Button>
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleRefresh} title="Refresh schema">
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onToggle} title="Close sidebar">
            <PanelLeftClose className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {/* Search */}
      <div className="px-2 py-1.5 border-b">
        <SchemaSearch value={searchQuery} onChange={setSearchQuery} />
      </div>

      {/* Tree */}
      <ScrollArea className={cn("flex-1", selectedTable && "max-h-[60%]")}>
        <div className="p-1">
          <SchemaTree
            searchQuery={searchQuery}
            showFavoritesOnly={showFavoritesOnly}
            favorites={favorites}
            onToggleFavorite={handleToggleFavorite}
            onSelectTable={handleSelectTable}
            onInsertSql={onInsertSql}
            onInsertColumn={handleInsertColumn}
            selectedTable={selectedTable}
          />
        </div>
      </ScrollArea>

      {/* Table Details */}
      {selectedTable && (
        <div className="min-h-[200px] max-h-[40%]">
          <TableDetails
            catalog={selectedTable.catalog}
            schema={selectedTable.schema}
            table={selectedTable.table}
            onInsertSql={onInsertSql}
            onInsertColumn={handleInsertColumn}
            onClose={() => setSelectedTable(null)}
          />
        </div>
      )}

      {/* Resize Handle */}
      <div
        className="absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-primary/20 active:bg-primary/30 transition-colors"
        onMouseDown={handleMouseDown}
      />
    </div>
  )
}
