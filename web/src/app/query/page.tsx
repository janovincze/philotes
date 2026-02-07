"use client"

import { useState, useCallback, useEffect, useMemo } from "react"
import { useSearchParams } from "next/navigation"
import { Play, AlertCircle, Clock, RotateCcw, PanelLeftOpen, WandSparkles } from "lucide-react"
import { format as formatSql } from "sql-formatter"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { SqlEditor } from "@/components/query/sql-editor"
import { ResultsTable } from "@/components/query/results-table"
import { QueryTemplates } from "@/components/query/query-templates"
import { QueryHistory, type QueryHistoryEntry } from "@/components/query/query-history"
import { QueryTabBar, type QueryTab } from "@/components/query/query-tabs"
import { SchemaBrowser } from "@/components/schema-browser/schema-browser"
import { useQueryExecute, useAutoCompleteMetadata } from "@/lib/hooks/use-query"
import type { QueryColumn } from "@/lib/api/types"

const DEFAULT_QUERY = `-- Write your SQL query here
-- Press Ctrl+Enter to execute
SELECT * FROM iceberg.public.customers LIMIT 10`

const SIDEBAR_STORAGE_KEY = "philotes-schema-browser-open"
const TABS_STORAGE_KEY = "philotes-query-tabs"

// --- Tab runtime state (not persisted) ---
interface TabRuntime {
  columns: QueryColumn[]
  rows: Record<string, unknown>[]
  queryTime: number | null
  truncated: boolean
  error: string | null
}

// --- Persistence helpers ---
interface PersistedTabState {
  activeTabId: string
  tabs: QueryTab[]
}

function generateTabId(): string {
  return `tab-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
}

function loadTabState(): { tabs: QueryTab[]; activeTabId: string } {
  if (typeof window === "undefined") {
    return { tabs: [{ id: "tab-1", name: "Query 1", sql: DEFAULT_QUERY }], activeTabId: "tab-1" }
  }
  try {
    const stored = localStorage.getItem(TABS_STORAGE_KEY)
    if (stored) {
      const parsed: PersistedTabState = JSON.parse(stored)
      if (parsed.tabs?.length > 0) {
        return { tabs: parsed.tabs, activeTabId: parsed.activeTabId || parsed.tabs[0].id }
      }
    }
  } catch {
    // ignore parse errors
  }
  return { tabs: [{ id: "tab-1", name: "Query 1", sql: DEFAULT_QUERY }], activeTabId: "tab-1" }
}

function saveTabState(tabs: QueryTab[], activeTabId: string) {
  try {
    localStorage.setItem(TABS_STORAGE_KEY, JSON.stringify({ activeTabId, tabs }))
  } catch {
    // ignore quota errors
  }
}

// Parse error messages to extract line/column information
function parseErrorLocation(error: string): { line: number; column: number; message: string } | null {
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

  // --- Tab state (single loadTabState call to avoid divergence) ---
  const [initialState] = useState(loadTabState)
  const [tabs, setTabs] = useState<QueryTab[]>(initialState.tabs)
  const [activeTabId, setActiveTabId] = useState<string>(initialState.activeTabId)
  const [runtimeState, setRuntimeState] = useState<Map<string, TabRuntime>>(() => new Map())

  // Persist tabs on change
  useEffect(() => {
    saveTabState(tabs, activeTabId)
  }, [tabs, activeTabId])

  // Handle URL table parameter — update active tab SQL
  useEffect(() => {
    if (tableParam) {
      setTabs((prev) =>
        prev.map((t) =>
          t.id === activeTabId ? { ...t, sql: `SELECT * FROM ${tableParam} LIMIT 10` } : t
        )
      )
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tableParam])

  const activeTab = tabs.find((t) => t.id === activeTabId) || tabs[0]
  const activeRuntime = runtimeState.get(activeTabId) || {
    columns: [],
    rows: [],
    queryTime: null,
    truncated: false,
    error: null,
  }

  const updateActiveTab = useCallback(
    (updates: Partial<QueryTab>) => {
      setTabs((prev) =>
        prev.map((t) => (t.id === activeTabId ? { ...t, ...updates } : t))
      )
    },
    [activeTabId]
  )

  const updateActiveRuntime = useCallback(
    (updates: Partial<TabRuntime>) => {
      setRuntimeState((prev) => {
        const next = new Map(prev)
        const current = next.get(activeTabId) || {
          columns: [],
          rows: [],
          queryTime: null,
          truncated: false,
          error: null,
        }
        next.set(activeTabId, { ...current, ...updates })
        return next
      })
    },
    [activeTabId]
  )

  // --- Tab operations ---
  const handleAddTab = useCallback(() => {
    const newId = generateTabId()
    const tabNumber = tabs.length + 1
    setTabs((prev) => [...prev, { id: newId, name: `Query ${tabNumber}`, sql: "" }])
    setActiveTabId(newId)
  }, [tabs.length])

  const handleCloseTab = useCallback(
    (id: string) => {
      setTabs((prev) => {
        if (prev.length <= 1) return prev // never close last tab
        const idx = prev.findIndex((t) => t.id === id)
        const filtered = prev.filter((t) => t.id !== id)
        // If closing the active tab, switch to the adjacent tab
        if (id === activeTabId) {
          const newActive = filtered[Math.min(idx, filtered.length - 1)] || filtered[0]
          setActiveTabId(newActive.id)
        }
        return filtered
      })
      // Clean up runtime state
      setRuntimeState((prev) => {
        const next = new Map(prev)
        next.delete(id)
        return next
      })
    },
    [activeTabId]
  )

  const handleRenameTab = useCallback((id: string, name: string) => {
    setTabs((prev) => prev.map((t) => (t.id === id ? { ...t, name } : t)))
  }, [])

  // --- Query history (shared across tabs) ---
  const [queryHistory, setQueryHistory] = useState<QueryHistoryEntry[]>([])

  // Schema browser sidebar state
  const [schemaBrowserOpen, setSchemaBrowserOpen] = useState(() => {
    if (typeof window === "undefined") return true
    return localStorage.getItem(SIDEBAR_STORAGE_KEY) !== "false"
  })

  useEffect(() => {
    localStorage.setItem(SIDEBAR_STORAGE_KEY, String(schemaBrowserOpen))
  }, [schemaBrowserOpen])

  const queryMutation = useQueryExecute()
  const metadata = useAutoCompleteMetadata()

  // Parse error for editor highlighting
  const errorMarker = useMemo(() => {
    if (!activeRuntime.error) return null
    return parseErrorLocation(activeRuntime.error)
  }, [activeRuntime.error])

  const addToHistory = useCallback((entry: Omit<QueryHistoryEntry, "id">) => {
    setQueryHistory((prev) => {
      const newEntry: QueryHistoryEntry = {
        ...entry,
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`,
      }
      return [newEntry, ...prev].slice(0, 20)
    })
  }, [])

  const executeQuery = useCallback(
    (sqlToExecute: string) => {
      if (!sqlToExecute.trim()) return

      const cleanedSql = sqlToExecute.replace(/--.*$/gm, "").trim()
      if (!cleanedSql) return

      updateActiveRuntime({ error: null, columns: [], rows: [], queryTime: null, truncated: false })

      queryMutation.mutate(
        { sql: sqlToExecute, limit: 100 },
        {
          onSuccess: (data) => {
            if (data.error) {
              updateActiveRuntime({ error: data.error })
              addToHistory({ sql: sqlToExecute, executedAt: new Date(), error: data.error })
            } else {
              updateActiveRuntime({
                columns: data.columns || [],
                rows: data.rows || [],
                queryTime: data.query_time_ms,
                truncated: data.truncated,
              })
              addToHistory({
                sql: sqlToExecute,
                executedAt: new Date(),
                durationMs: data.query_time_ms,
                rowCount: data.row_count,
              })
            }
          },
          onError: (err) => {
            const errorMsg = err instanceof Error ? err.message : "Query execution failed"
            updateActiveRuntime({ error: errorMsg })
            addToHistory({ sql: sqlToExecute, executedAt: new Date(), error: errorMsg })
          },
        }
      )
    },
    [queryMutation, addToHistory, updateActiveRuntime]
  )

  const handleExecute = useCallback(() => {
    executeQuery(activeTab.sql)
  }, [executeQuery, activeTab.sql])

  const handleExecuteSelection = useCallback(
    (selectedSql: string) => {
      executeQuery(selectedSql)
    },
    [executeQuery]
  )

  const handleFormat = useCallback(() => {
    try {
      const formatted = formatSql(activeTab.sql, {
        language: "trino",
        keywordCase: "upper",
        tabWidth: 2,
      })
      updateActiveTab({ sql: formatted })
    } catch {
      // If formatting fails (e.g., invalid SQL), keep original
    }
  }, [activeTab.sql, updateActiveTab])

  const handleInsertSql = useCallback(
    (sqlToInsert: string) => {
      updateActiveTab({
        sql: (() => {
          const stripped = activeTab.sql.replace(/--.*$/gm, "").trim()
          if (stripped) {
            return activeTab.sql + "\n" + sqlToInsert
          }
          return sqlToInsert
        })(),
      })
    },
    [activeTab.sql, updateActiveTab]
  )

  const handleTemplateSelect = useCallback(
    (templateSql: string) => {
      updateActiveTab({ sql: templateSql })
      updateActiveRuntime({ error: null })
    },
    [updateActiveTab, updateActiveRuntime]
  )

  const handleHistorySelect = useCallback(
    (historySql: string) => {
      updateActiveTab({ sql: historySql })
      updateActiveRuntime({ error: null })
    },
    [updateActiveTab, updateActiveRuntime]
  )

  const handleClearHistory = useCallback(() => {
    setQueryHistory([])
  }, [])

  const handleClear = useCallback(() => {
    updateActiveTab({ sql: "" })
    updateActiveRuntime({ columns: [], rows: [], queryTime: null, truncated: false, error: null })
  }, [updateActiveTab, updateActiveRuntime])

  return (
    <div className="flex h-full">
      {/* Schema Browser Sidebar */}
      <SchemaBrowser
        onInsertSql={handleInsertSql}
        isOpen={schemaBrowserOpen}
        onToggle={() => setSchemaBrowserOpen(false)}
      />

      {/* Main Content */}
      <div className="flex flex-col flex-1 min-w-0">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            {!schemaBrowserOpen && (
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => setSchemaBrowserOpen(true)}
                title="Open schema browser"
              >
                <PanelLeftOpen className="h-4 w-4" />
              </Button>
            )}
            <div>
              <h1 className="text-2xl font-bold">Query</h1>
              <p className="text-muted-foreground">
                Execute SQL queries against your Iceberg tables
              </p>
            </div>
          </div>
        </div>

        <div className="flex-1 grid grid-rows-[auto_1fr] gap-4 min-h-0">
          {/* Editor Section */}
          <Card>
            {/* Tab Bar */}
            <QueryTabBar
              tabs={tabs}
              activeTabId={activeTabId}
              onSelectTab={setActiveTabId}
              onAddTab={handleAddTab}
              onCloseTab={handleCloseTab}
              onRenameTab={handleRenameTab}
            />
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
                    onClick={handleFormat}
                    disabled={queryMutation.isPending || !activeTab.sql.trim()}
                    title="Format SQL (Ctrl+Shift+F)"
                  >
                    <WandSparkles className="h-4 w-4 mr-2" />
                    Format
                  </Button>
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
                    disabled={queryMutation.isPending || !activeTab.sql.trim()}
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
                value={activeTab.sql}
                onChange={(newSql) => updateActiveTab({ sql: newSql })}
                onExecute={handleExecute}
                onExecuteSelection={handleExecuteSelection}
                onFormat={handleFormat}
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
                  {activeRuntime.queryTime !== null && (
                    <CardDescription className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      Query completed in {activeRuntime.queryTime}ms
                      {activeRuntime.truncated && " (results truncated to 100 rows)"}
                    </CardDescription>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="flex-1 min-h-0 overflow-auto">
              {activeRuntime.error && (
                <Alert variant="destructive" className="mb-4">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>{activeRuntime.error}</AlertDescription>
                </Alert>
              )}
              <ResultsTable
                columns={activeRuntime.columns}
                rows={activeRuntime.rows}
                isLoading={queryMutation.isPending}
              />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
