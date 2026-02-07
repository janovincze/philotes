"use client"

import { useState, useMemo } from "react"
import { ChevronDown, ChevronUp, Download, FileJson, Copy } from "lucide-react"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area"
import { toast } from "sonner"
import type { QueryColumn } from "@/lib/api/types"

interface ResultsTableProps {
  columns: QueryColumn[]
  rows: Record<string, unknown>[]
  isLoading?: boolean
}

type SortDirection = "asc" | "desc" | null

export function ResultsTable({ columns, rows, isLoading }: ResultsTableProps) {
  const [sortColumn, setSortColumn] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<SortDirection>(null)

  // Sort rows
  const sortedRows = useMemo(() => {
    if (!sortColumn || !sortDirection) return rows

    return [...rows].sort((a, b) => {
      const aVal = a[sortColumn]
      const bVal = b[sortColumn]

      if (aVal === null || aVal === undefined) return sortDirection === "asc" ? 1 : -1
      if (bVal === null || bVal === undefined) return sortDirection === "asc" ? -1 : 1

      if (typeof aVal === "number" && typeof bVal === "number") {
        return sortDirection === "asc" ? aVal - bVal : bVal - aVal
      }

      const aStr = String(aVal)
      const bStr = String(bVal)
      return sortDirection === "asc"
        ? aStr.localeCompare(bStr)
        : bStr.localeCompare(aStr)
    })
  }, [rows, sortColumn, sortDirection])

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      if (sortDirection === "asc") {
        setSortDirection("desc")
      } else if (sortDirection === "desc") {
        setSortColumn(null)
        setSortDirection(null)
      }
    } else {
      setSortColumn(column)
      setSortDirection("asc")
    }
  }

  const exportToCsv = () => {
    if (columns.length === 0 || sortedRows.length === 0) return

    const headers = columns.map((c) => `"${c.name.replace(/"/g, '""')}"`).join(",")
    const dataRows = sortedRows.map((row) =>
      columns
        .map((col) => {
          const value = row[col.name]
          if (value === null || value === undefined) return ""
          const str = String(value).replace(/"/g, '""')
          return `"${str}"`
        })
        .join(",")
    )
    const csv = [headers, ...dataRows].join("\n")

    downloadFile(csv, `query-results-${Date.now()}.csv`, "text/csv;charset=utf-8;")
  }

  const exportToJson = () => {
    if (columns.length === 0 || sortedRows.length === 0) return

    const json = JSON.stringify(sortedRows, null, 2)
    downloadFile(json, `query-results-${Date.now()}.json`, "application/json;charset=utf-8;")
  }

  const handleCellCopy = (value: unknown) => {
    const text = formatCellText(value)
    if (!navigator.clipboard) {
      toast.error("Clipboard not available")
      return
    }
    navigator.clipboard.writeText(text).then(
      () => toast.success("Copied to clipboard"),
      () => toast.error("Failed to copy")
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground">
        Executing query...
      </div>
    )
  }

  if (columns.length === 0) {
    return (
      <div className="flex items-center justify-center h-48 text-muted-foreground">
        No results to display. Run a query to see data.
      </div>
    )
  }

  return (
    <div className="space-y-2" data-testid="results-table">
      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground" data-testid="row-count">
          {rows.length} row{rows.length !== 1 ? "s" : ""}
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              const allText = sortedRows
                .map((row) => columns.map((col) => formatCellText(row[col.name])).join("\t"))
                .join("\n")
              if (!navigator.clipboard) {
                toast.error("Clipboard not available")
                return
              }
              navigator.clipboard.writeText(allText).then(
                () => toast.success("All rows copied"),
                () => toast.error("Failed to copy")
              )
            }}
          >
            <Copy className="h-4 w-4 mr-2" />
            Copy All
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" data-testid="export-csv-button">
                <Download className="h-4 w-4 mr-2" />
                Export
                <ChevronDown className="h-3 w-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={exportToCsv}>
                <Download className="h-4 w-4 mr-2" />
                Export as CSV
              </DropdownMenuItem>
              <DropdownMenuItem onClick={exportToJson}>
                <FileJson className="h-4 w-4 mr-2" />
                Export as JSON
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <ScrollArea className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              {columns.map((column) => (
                <TableHead
                  key={column.name}
                  className="cursor-pointer hover:bg-muted/50 whitespace-nowrap"
                  onClick={() => handleSort(column.name)}
                >
                  <div className="flex items-center gap-1">
                    <span>{column.name}</span>
                    <span className="text-xs text-muted-foreground">
                      ({column.type})
                    </span>
                    {sortColumn === column.name && (
                      sortDirection === "asc" ? (
                        <ChevronUp className="h-3 w-3" />
                      ) : (
                        <ChevronDown className="h-3 w-3" />
                      )
                    )}
                  </div>
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedRows.map((row, rowIndex) => (
              <TableRow key={rowIndex}>
                {columns.map((column) => {
                  const value = row[column.name]
                  const isNull = value === null || value === undefined
                  return (
                    <TableCell
                      key={column.name}
                      className="whitespace-nowrap cursor-pointer hover:bg-muted/30"
                      onClick={() => handleCellCopy(value)}
                      title="Click to copy"
                    >
                      {isNull ? (
                        <span className="italic text-muted-foreground">NULL</span>
                      ) : (
                        formatCellText(value)
                      )}
                    </TableCell>
                  )
                })}
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
    </div>
  )
}

function formatCellText(value: unknown): string {
  if (value === null || value === undefined) {
    return "NULL"
  }
  if (typeof value === "boolean") {
    return value ? "true" : "false"
  }
  if (typeof value === "object") {
    return JSON.stringify(value)
  }
  return String(value)
}

function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement("a")
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
