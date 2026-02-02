"use client"

import { useRef, useCallback } from "react"
import Editor, { type Monaco, type OnMount } from "@monaco-editor/react"
import { useTheme } from "next-themes"
import type * as monacoEditor from "monaco-editor"

interface SqlEditorProps {
  value: string
  onChange: (value: string) => void
  onExecute?: () => void
  disabled?: boolean
  height?: string
}

export function SqlEditor({
  value,
  onChange,
  onExecute,
  disabled = false,
  height = "300px",
}: SqlEditorProps) {
  const { resolvedTheme } = useTheme()
  const editorRef = useRef<monacoEditor.editor.IStandaloneCodeEditor | null>(null)

  const handleEditorMount: OnMount = (editor, monaco) => {
    editorRef.current = editor as monacoEditor.editor.IStandaloneCodeEditor

    // Add Ctrl+Enter keybinding to execute
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => {
      onExecute?.()
    })

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
  )
}

// Configure SQL language features for Monaco
function configureSqlLanguage(monaco: Monaco) {
  // SQL keywords for autocomplete
  const sqlKeywords = [
    "SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "IN", "IS", "NULL",
    "JOIN", "INNER", "LEFT", "RIGHT", "OUTER", "ON", "AS", "ORDER", "BY",
    "ASC", "DESC", "GROUP", "HAVING", "LIMIT", "OFFSET", "UNION", "ALL",
    "DISTINCT", "COUNT", "SUM", "AVG", "MIN", "MAX", "CASE", "WHEN", "THEN",
    "ELSE", "END", "CAST", "COALESCE", "NULLIF", "EXISTS", "BETWEEN", "LIKE",
    "WITH", "EXPLAIN", "ANALYZE", "SHOW", "DESCRIBE", "CATALOGS", "SCHEMAS",
    "TABLES", "COLUMNS", "PARTITIONS"
  ]

  // Register SQL completion provider
  monaco.languages.registerCompletionItemProvider("sql", {
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

      const suggestions = sqlKeywords.map((keyword) => ({
        label: keyword,
        kind: monaco.languages.CompletionItemKind.Keyword,
        insertText: keyword,
        range: range,
      }))

      return { suggestions }
    },
  })
}
