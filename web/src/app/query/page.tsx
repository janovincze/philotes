"use client"

import { useState, useCallback, useEffect, useMemo } from "react"
import { useSearchParams } from "next/navigation"
import { Play, AlertCircle, Clock, RotateCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { SqlEditor } from "@/components/query/sql-editor"
import { ResultsTable } from "@/components/query/results-table"
import { QueryTemplates } from "@/components/query/query-templates"
import { QueryHistory, type QueryHistoryEntry } from "@/components/query/query-history"
import { useQueryExecute, useAutoCompleteMetadata } from "@/lib/hooks/use-query"
import type { QueryColumn } from "@/lib/api/types"

const DEFAULT_QUERY = `-- Write your SQL query here
-- Press Ctrl+Enter to execute
SELECT * FROM iceberg.public.customers LIMIT 10`

// Parse error messages to extract line/column information
function parseErrorLocation(error: string): { line: number; column: number; message: string } | null {
  // Common Trino error patterns:
  // "line 1:15: Column 'foo' cannot be resolved"
  // "Query failed (#20240101_123456_00001_xxxxx): line 2:10: ..."
  const lineColMatch = error.match(/line\s+(\d+):(\d+):\s*(.+)/i)
  if (lineColMatch) {
    return {
      line: parseInt(lineColMatch[1], 10),
      column: parseInt(lineColMatch[2], 10),
      message: lineColMatch[3],
    }
  }
  return null
}

export default function QueryPage() {
  const searchParams = useSearchParams()
  const rawTableParam = searchParams.get("table")
  const tableParam =
    rawTableParam && /^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$/.test(rawTableParam.trim())
      ? rawTableParam.trim()
      : null
  const [sql, setSql] = useState(
    tableParam ? `SELECT * FROM ${tableParam} LIMIT 10` : DEFAULT_QUERY
  )

  useEffect(() => {
    if (tableParam) {
      setSql(`SELECT * FROM ${tableParam} LIMIT 10`)
    }
  }, [tableParam])
  const [columns, setColumns] = useState<QueryColumn[]>([])
  const [rows, setRows] = useState<Record<string, unknown>[]>([])
  const [queryTime, setQueryTime] = useState<number | null>(null)
  const [truncated, setTruncated] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [queryHistory, setQueryHistory] = useState<QueryHistoryEntry[]>([])

  const queryMutation = useQueryExecute()
  const metadata = useAutoCompleteMetadata()

  // Parse error for editor highlighting
  const errorMarker = useMemo(() => {
    if (!error) return null
    return parseErrorLocation(error)
  }, [error])

  const addToHistory = useCallback((entry: Omit<QueryHistoryEntry, "id">) => {
    setQueryHistory((prev) => {
      const newEntry: QueryHistoryEntry = {
        ...entry,
        id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      }
      // Keep only last 20 queries
      return [newEntry, ...prev].slice(0, 20)
    })
  }, [])

  const handleExecute = useCallback(() => {
    if (!sql.trim()) return

    // Clear any comment-only lines for validation
    const cleanedSql = sql.replace(/--.*$/gm, "").trim()
    if (!cleanedSql) return

    setError(null)
    setColumns([])
    setRows([])
    setQueryTime(null)
    setTruncated(false)

    queryMutation.mutate(
      { sql, limit: 100 },
      {
        onSuccess: (data) => {
          if (data.error) {
            setError(data.error)
            addToHistory({
              sql,
              executedAt: new Date(),
              error: data.error,
            })
          } else {
            setColumns(data.columns || [])
            setRows(data.rows || [])
            setQueryTime(data.query_time_ms)
            setTruncated(data.truncated)
            addToHistory({
              sql,
              executedAt: new Date(),
              durationMs: data.query_time_ms,
              rowCount: data.row_count,
            })
          }
        },
        onError: (err) => {
          const errorMsg = err instanceof Error ? err.message : "Query execution failed"
          setError(errorMsg)
          addToHistory({
            sql,
            executedAt: new Date(),
            error: errorMsg,
          })
        },
      }
    )
  }, [sql, queryMutation, addToHistory])

  const handleTemplateSelect = useCallback((templateSql: string) => {
    setSql(templateSql)
    setError(null)
  }, [])

  const handleHistorySelect = useCallback((historySql: string) => {
    setSql(historySql)
    setError(null)
  }, [])

  const handleClearHistory = useCallback(() => {
    setQueryHistory([])
  }, [])

  const handleClear = useCallback(() => {
    setSql("")
    setColumns([])
    setRows([])
    setQueryTime(null)
    setTruncated(false)
    setError(null)
  }, [])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Query</h1>
          <p className="text-muted-foreground">
            Execute SQL queries against your Iceberg tables
          </p>
        </div>
      </div>

      <div className="flex-1 grid grid-rows-[auto_1fr] gap-4 min-h-0">
        {/* Editor Section */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-lg">SQL Editor</CardTitle>
                <CardDescription>
                  Write and execute SQL queries. Press Ctrl+Enter to run.
                </CardDescription>
              </div>
              <div className="flex items-center gap-2">
                <QueryHistory
                  history={queryHistory}
                  onSelect={handleHistorySelect}
                  onClear={handleClearHistory}
                />
                <QueryTemplates onSelect={handleTemplateSelect} />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleClear}
                  disabled={queryMutation.isPending}
                >
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Clear
                </Button>
                <Button
                  size="sm"
                  onClick={handleExecute}
                  disabled={queryMutation.isPending || !sql.trim()}
                  data-testid="run-query-button"
                >
                  <Play className="h-4 w-4 mr-2" />
                  {queryMutation.isPending ? "Running..." : "Run Query"}
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <SqlEditor
              value={sql}
              onChange={setSql}
              onExecute={handleExecute}
              disabled={queryMutation.isPending}
              height="200px"
              metadata={metadata}
              errorMarker={errorMarker}
            />
          </CardContent>
        </Card>

        {/* Results Section */}
        <Card className="min-h-0 flex flex-col">
          <CardHeader className="pb-3 flex-shrink-0">
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="text-lg">Results</CardTitle>
                {queryTime !== null && (
                  <CardDescription className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    Query completed in {queryTime}ms
                    {truncated && " (results truncated to 100 rows)"}
                  </CardDescription>
                )}
              </div>
            </div>
          </CardHeader>
          <CardContent className="flex-1 min-h-0 overflow-auto">
            {error && (
              <Alert variant="destructive" className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <ResultsTable
              columns={columns}
              rows={rows}
              isLoading={queryMutation.isPending}
            />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
