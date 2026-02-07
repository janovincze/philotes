"use client"

import { useRef, useCallback, useEffect } from "react"
import Editor, { type Monaco, type OnMount } from "@monaco-editor/react"
import { useTheme } from "next-themes"
import type * as monacoEditor from "monaco-editor"
import type { AutoCompleteMetadata } from "@/lib/hooks/use-query"

interface SqlEditorProps {
  value: string
  onChange: (value: string) => void
  onExecute?: () => void
  onExecuteSelection?: (sql: string) => void
  onFormat?: () => void
  disabled?: boolean
  height?: string
  metadata?: AutoCompleteMetadata
  errorMarker?: { line: number; column: number; message: string } | null
}

// Store metadata globally for the completion provider
let globalMetadata: AutoCompleteMetadata | null = null
let completionProviderDisposable: monacoEditor.IDisposable | null = null

export function SqlEditor({
  value,
  onChange,
  onExecute,
  onExecuteSelection,
  onFormat,
  disabled = false,
  height = "300px",
  metadata,
  errorMarker,
}: SqlEditorProps) {
  const { resolvedTheme } = useTheme()
  const editorRef = useRef<monacoEditor.editor.IStandaloneCodeEditor | null>(null)
  const monacoRef = useRef<Monaco | null>(null)
  const onExecuteRef = useRef(onExecute)
  const onExecuteSelectionRef = useRef(onExecuteSelection)
  const onFormatRef = useRef(onFormat)

  // Keep refs in sync with latest callbacks
  useEffect(() => { onExecuteRef.current = onExecute }, [onExecute])
  useEffect(() => { onExecuteSelectionRef.current = onExecuteSelection }, [onExecuteSelection])
  useEffect(() => { onFormatRef.current = onFormat }, [onFormat])

  // Update global metadata when it changes
  useEffect(() => {
    globalMetadata = metadata || null
  }, [metadata])

  // Update error markers when they change
  useEffect(() => {
    if (!editorRef.current || !monacoRef.current) return

    const model = editorRef.current.getModel()
    if (!model) return

    if (errorMarker) {
      monacoRef.current.editor.setModelMarkers(model, "sql-error", [
        {
          severity: monacoRef.current.MarkerSeverity.Error,
          message: errorMarker.message,
          startLineNumber: errorMarker.line,
          startColumn: errorMarker.column,
          endLineNumber: errorMarker.line,
          endColumn: model.getLineMaxColumn(errorMarker.line),
        },
      ])
    } else {
      monacoRef.current.editor.setModelMarkers(model, "sql-error", [])
    }
  }, [errorMarker])

  const handleEditorMount: OnMount = (editor, monaco) => {
    editorRef.current = editor as monacoEditor.editor.IStandaloneCodeEditor
    monacoRef.current = monaco

    // Ctrl/Cmd+Enter — Execute full query
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
      onExecuteRef.current?.()
    })

    // Ctrl/Cmd+Shift+Enter — Execute selected text
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.Enter,
      () => {
        const selection = editor.getSelection()
        const model = editor.getModel()
        if (selection && model && !selection.isEmpty()) {
          const selectedText = model.getValueInRange(selection).trim()
          if (selectedText) {
            onExecuteSelectionRef.current?.(selectedText)
            return
          }
        }
        // Fallback to full query execution
        onExecuteRef.current?.()
      }
    )

    // Ctrl/Cmd+Shift+F — Format SQL
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyF,
      () => {
        onFormatRef.current?.()
      }
    )

    // Configure SQL language features
    configureSqlLanguage(monaco)
  }

  const handleChange = useCallback(
    (value: string | undefined) => {
      onChange(value || "")
    },
    [onChange]
  )

  return (
    <div className="flex flex-col gap-1">
      <div className="rounded-md border overflow-hidden">
        <Editor
          height={height}
          defaultLanguage="sql"
          value={value}
          onChange={handleChange}
          onMount={handleEditorMount}
          theme={resolvedTheme === "dark" ? "vs-dark" : "light"}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: "on",
            scrollBeyondLastLine: false,
            wordWrap: "on",
            automaticLayout: true,
            tabSize: 2,
            readOnly: disabled,
            folding: true,
            foldingStrategy: "indentation",
            scrollbar: {
              vertical: "auto",
              horizontal: "auto",
            },
            suggestOnTriggerCharacters: true,
            quickSuggestions: true,
            formatOnPaste: true,
            formatOnType: true,
          }}
          loading={
            <div className="flex items-center justify-center h-full text-muted-foreground">
              Loading editor...
            </div>
          }
        />
      </div>
      <div className="flex items-center gap-3 text-[10px] text-muted-foreground px-1">
        <span>Ctrl/Cmd+Enter: Run</span>
        <span>Ctrl/Cmd+Shift+Enter: Run Selection</span>
        <span>Ctrl/Cmd+Shift+F: Format</span>
        <span>Ctrl/Cmd+/: Comment</span>
      </div>
    </div>
  )
}

// Configure SQL language features for Monaco
function configureSqlLanguage(monaco: Monaco) {
  // Dispose of previous completion provider to avoid duplicates
  if (completionProviderDisposable) {
    completionProviderDisposable.dispose()
  }

  // SQL keywords for autocomplete
  const sqlKeywords = [
    "SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "IN", "IS", "NULL",
    "JOIN", "INNER", "LEFT", "RIGHT", "OUTER", "FULL", "CROSS", "ON", "AS",
    "ORDER", "BY", "ASC", "DESC", "GROUP", "HAVING", "LIMIT", "OFFSET",
    "UNION", "ALL", "DISTINCT", "COUNT", "SUM", "AVG", "MIN", "MAX",
    "CASE", "WHEN", "THEN", "ELSE", "END", "CAST", "COALESCE", "NULLIF",
    "EXISTS", "BETWEEN", "LIKE", "ILIKE", "WITH", "EXPLAIN", "ANALYZE",
    "SHOW", "DESCRIBE", "CATALOGS", "SCHEMAS", "TABLES", "COLUMNS",
    "PARTITIONS", "TRUE", "FALSE", "CURRENT_DATE", "CURRENT_TIME",
    "CURRENT_TIMESTAMP", "ROW_NUMBER", "RANK", "DENSE_RANK", "OVER",
    "PARTITION", "WINDOW", "ROWS", "RANGE", "UNBOUNDED", "PRECEDING",
    "FOLLOWING", "CURRENT", "ROW"
  ]

  // SQL functions
  const sqlFunctions = [
    "COUNT", "SUM", "AVG", "MIN", "MAX", "ABS", "CEIL", "FLOOR", "ROUND",
    "TRUNC", "POWER", "SQRT", "EXP", "LOG", "LOG10", "LENGTH", "LOWER",
    "UPPER", "TRIM", "LTRIM", "RTRIM", "SUBSTR", "SUBSTRING", "CONCAT",
    "REPLACE", "SPLIT", "REGEXP_LIKE", "REGEXP_REPLACE", "REGEXP_EXTRACT",
    "DATE", "TIME", "TIMESTAMP", "YEAR", "MONTH", "DAY", "HOUR", "MINUTE",
    "SECOND", "DATE_ADD", "DATE_SUB", "DATE_DIFF", "DATE_TRUNC", "DATE_FORMAT",
    "NOW", "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP",
    "COALESCE", "NULLIF", "IF", "IFNULL", "CASE", "CAST", "TRY_CAST",
    "JSON_EXTRACT", "JSON_EXTRACT_SCALAR", "ARRAY", "MAP", "ROW"
  ]

  // Register SQL completion provider
  completionProviderDisposable = monaco.languages.registerCompletionItemProvider("sql", {
    triggerCharacters: [".", " "],
    provideCompletionItems: (
      model: monacoEditor.editor.ITextModel,
      position: monacoEditor.Position
    ) => {
      const word = model.getWordUntilPosition(position)
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      }

      const suggestions: monacoEditor.languages.CompletionItem[] = []

      // Get text before cursor to detect context
      const lineContent = model.getLineContent(position.lineNumber)
      const textBeforeCursor = lineContent.substring(0, position.column - 1)

      // Check if we're after a dot (table.column reference)
      const dotMatch = textBeforeCursor.match(/(\w+)\.(\w*)$/)

      if (dotMatch && globalMetadata) {
        const prefix = dotMatch[1].toLowerCase()
        // Could be catalog.schema, schema.table, or table.column

        // Check if prefix is a catalog
        if (globalMetadata.catalogs.some(c => c.toLowerCase() === prefix)) {
          // Suggest schemas for this catalog
          globalMetadata.schemas
            .filter(s => s.catalog.toLowerCase() === prefix)
            .forEach(schema => {
              suggestions.push({
                label: schema.name,
                kind: monaco.languages.CompletionItemKind.Module,
                insertText: schema.name,
                detail: `Schema in ${schema.catalog}`,
                range,
              })
            })
        }

        // Check if prefix matches a schema
        const matchingSchemas = globalMetadata.schemas.filter(
          s => s.name.toLowerCase() === prefix
        )
        if (matchingSchemas.length > 0) {
          // Suggest tables in this schema
          matchingSchemas.forEach(schema => {
            globalMetadata!.tables
              .filter(t => t.schema.toLowerCase() === schema.name.toLowerCase())
              .forEach(table => {
                suggestions.push({
                  label: table.name,
                  kind: monaco.languages.CompletionItemKind.Class,
                  insertText: table.name,
                  detail: `Table in ${table.catalog}.${table.schema}`,
                  range,
                })
              })
          })
        }

        // Check if prefix matches a table name
        const matchingTables = globalMetadata.tables.filter(
          t => t.name.toLowerCase() === prefix
        )
        if (matchingTables.length > 0) {
          // Suggest columns for this table
          matchingTables.forEach(table => {
            globalMetadata!.columns
              .filter(c => c.table.toLowerCase() === table.name.toLowerCase())
              .forEach(col => {
                suggestions.push({
                  label: col.name,
                  kind: monaco.languages.CompletionItemKind.Field,
                  insertText: col.name,
                  detail: `${col.type} (${col.catalog}.${col.schema}.${col.table})`,
                  range,
                })
              })
          })
        }
      }

      // Add SQL keywords
      sqlKeywords.forEach((keyword) => {
        suggestions.push({
          label: keyword,
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: keyword,
          range,
        })
      })

      // Add SQL functions
      sqlFunctions.forEach((func) => {
        suggestions.push({
          label: func,
          kind: monaco.languages.CompletionItemKind.Function,
          insertText: func + "()",
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          detail: "SQL Function",
          range,
        })
      })

      // Add metadata-based suggestions
      if (globalMetadata) {
        // Add catalogs
        globalMetadata.catalogs.forEach((catalog) => {
          suggestions.push({
            label: catalog,
            kind: monaco.languages.CompletionItemKind.Module,
            insertText: catalog,
            detail: "Catalog",
            range,
          })
        })

        // Add fully qualified table names
        globalMetadata.tables.forEach((table) => {
          suggestions.push({
            label: table.fullName,
            kind: monaco.languages.CompletionItemKind.Class,
            insertText: table.fullName,
            detail: "Table",
            range,
          })
          // Also add just the table name
          suggestions.push({
            label: table.name,
            kind: monaco.languages.CompletionItemKind.Class,
            insertText: table.name,
            detail: `Table (${table.catalog}.${table.schema})`,
            range,
          })
        })

        // Add column names
        globalMetadata.columns.forEach((col) => {
          suggestions.push({
            label: col.name,
            kind: monaco.languages.CompletionItemKind.Field,
            insertText: col.name,
            detail: `${col.type} (${col.table})`,
            range,
          })
        })
      }

      return { suggestions }
    },
  })
}
