"use client"

import { useState, useEffect, useMemo } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { ArrowLeft, ArrowRight, Table, Search, Columns, Key } from "lucide-react"
import { useDiscoverTables } from "@/lib/hooks/use-sources"
import type { TableInfo } from "@/lib/api/types"

interface StepTablesProps {
  sourceId: string
  availableTables: TableInfo[]
  onTablesLoaded: (tables: TableInfo[]) => void
  selectedTables: string[]
  onSelectedTablesChange: (tables: string[]) => void
  onNext: () => void
  onBack: () => void
}

export function StepTables({
  sourceId,
  availableTables,
  onTablesLoaded,
  selectedTables,
  onSelectedTablesChange,
  onNext,
  onBack,
}: StepTablesProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const { data, isLoading, error } = useDiscoverTables(sourceId)

  // Update available tables when data loads
  useEffect(() => {
    if (data?.tables && data.tables.length > 0) {
      onTablesLoaded(data.tables)
    }
  }, [data, onTablesLoaded])

  // Memoize tables to prevent unnecessary re-renders
  const tables = useMemo(() => {
    return availableTables.length > 0 ? availableTables : data?.tables ?? []
  }, [availableTables, data?.tables])

  // Filter tables based on search query
  const filteredTables = useMemo(() => {
    if (!searchQuery.trim()) return tables
    const query = searchQuery.toLowerCase()
    return tables.filter(
      (table) =>
        table.name.toLowerCase().includes(query) ||
        table.schema.toLowerCase().includes(query)
    )
  }, [tables, searchQuery])

  const handleToggleTable = (table: TableInfo) => {
    const fullName = `${table.schema}.${table.name}`
    if (selectedTables.includes(fullName)) {
      onSelectedTablesChange(selectedTables.filter((t) => t !== fullName))
    } else {
      onSelectedTablesChange([...selectedTables, fullName])
    }
  }

  const handleSelectAll = () => {
    const allTableNames = filteredTables.map((t) => `${t.schema}.${t.name}`)
    onSelectedTablesChange(allTableNames)
  }

  const handleDeselectAll = () => {
    onSelectedTablesChange([])
  }

  const isTableSelected = (table: TableInfo) => {
    return selectedTables.includes(`${table.schema}.${table.name}`)
  }

  const getPrimaryKeyColumns = (table: TableInfo) => {
    return table.columns.filter((col) => col.primary_key).map((col) => col.name)
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="space-y-2">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-72" />
        </div>
        <Skeleton className="h-10 w-full" />
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="text-center py-8">
          <p className="text-destructive">Failed to discover tables: {error.message}</p>
          <Button variant="outline" onClick={onBack} className="mt-4">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Go Back
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Table className="h-5 w-5 text-primary" />
          <h2 className="text-xl font-semibold">Select Tables to Replicate</h2>
        </div>
        <p className="text-sm text-muted-foreground">
          Choose which tables you want to replicate to your data lake.
          {tables.length > 0 && ` Found ${tables.length} tables.`}
        </p>
      </div>

      {/* Search and actions */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search tables..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={handleSelectAll}>
            Select All
          </Button>
          <Button variant="outline" size="sm" onClick={handleDeselectAll}>
            Deselect All
          </Button>
        </div>
      </div>

      {/* Selected count */}
      {selectedTables.length > 0 && (
        <div className="text-sm text-muted-foreground">
          {selectedTables.length} table{selectedTables.length !== 1 ? "s" : ""} selected
        </div>
      )}

      {/* Table list */}
      <div className="space-y-2 max-h-[400px] overflow-y-auto">
        {filteredTables.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            {searchQuery ? "No tables match your search" : "No tables found"}
          </div>
        ) : (
          filteredTables.map((table) => {
            const primaryKeys = getPrimaryKeyColumns(table)
            const isSelected = isTableSelected(table)

            return (
              <div
                key={`${table.schema}.${table.name}`}
                className={`flex items-start gap-3 p-4 rounded-lg border cursor-pointer transition-colors ${
                  isSelected
                    ? "bg-primary/5 border-primary/50"
                    : "bg-card hover:bg-muted/50"
                }`}
                onClick={() => handleToggleTable(table)}
              >
                <Checkbox
                  checked={isSelected}
                  onCheckedChange={() => handleToggleTable(table)}
                  className="mt-1"
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <Label className="font-medium cursor-pointer">
                      {table.schema}.{table.name}
                    </Label>
                    {primaryKeys.length > 0 && (
                      <Badge variant="outline" className="text-xs">
                        <Key className="mr-1 h-3 w-3" />
                        {primaryKeys.join(", ")}
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-center gap-1 mt-1 text-xs text-muted-foreground">
                    <Columns className="h-3 w-3" />
                    <span>{table.columns.length} columns</span>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center justify-between pt-4">
        <Button variant="outline" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Button onClick={onNext} disabled={selectedTables.length === 0}>
          Continue
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
