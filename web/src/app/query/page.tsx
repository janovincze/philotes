"use client"

import { useState, useCallback } from "react"
import { Play, AlertCircle, Clock, RotateCcw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { SqlEditor } from "@/components/query/sql-editor"
import { ResultsTable } from "@/components/query/results-table"
import { QueryTemplates } from "@/components/query/query-templates"
import { useQueryExecute } from "@/lib/hooks/use-query"
import type { QueryColumn } from "@/lib/api/types"

const DEFAULT_QUERY = `-- Write your SQL query here
-- Press Ctrl+Enter to execute
SELECT * FROM iceberg.public.customers LIMIT 10`

export default function QueryPage() {
  const [sql, setSql] = useState(DEFAULT_QUERY)
  const [columns, setColumns] = useState<QueryColumn[]>([])
  const [rows, setRows] = useState<Record<string, unknown>[]>([])
  const [queryTime, setQueryTime] = useState<number | null>(null)
  const [truncated, setTruncated] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const queryMutation = useQueryExecute()

  const handleExecute = useCallback(() => {
    if (!sql.trim()) return

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
          } else {
            setColumns(data.columns || [])
            setRows(data.rows || [])
            setQueryTime(data.query_time_ms)
            setTruncated(data.truncated)
          }
        },
        onError: (err) => {
          setError(err instanceof Error ? err.message : "Query execution failed")
        },
      }
    )
  }, [sql, queryMutation])

  const handleTemplateSelect = useCallback((templateSql: string) => {
    setSql(templateSql)
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
