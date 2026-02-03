"use client"

import { useState } from "react"
import { History, Clock, CheckCircle, XCircle, ChevronDown, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export interface QueryHistoryEntry {
  id: string
  sql: string
  executedAt: Date
  durationMs?: number
  rowCount?: number
  error?: string
}

interface QueryHistoryProps {
  history: QueryHistoryEntry[]
  onSelect: (sql: string) => void
  onClear: () => void
}

export function QueryHistory({ history, onSelect, onClear }: QueryHistoryProps) {
  const [isOpen, setIsOpen] = useState(false)

  if (history.length === 0) {
    return null
  }

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
  }

  const truncateSql = (sql: string, maxLength = 50) => {
    // Remove comments and newlines for display
    const cleaned = sql.replace(/--.*$/gm, "").replace(/\n/g, " ").trim()
    if (cleaned.length <= maxLength) return cleaned
    return cleaned.substring(0, maxLength) + "..."
  }

  return (
    <DropdownMenu open={isOpen} onOpenChange={setIsOpen}>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <History className="h-4 w-4 mr-2" />
          History
          <span className="ml-1 text-muted-foreground">({history.length})</span>
          <ChevronDown className="h-4 w-4 ml-2" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-80">
        <div className="flex items-center justify-between px-2 py-1.5">
          <span className="text-sm font-medium">Recent Queries</span>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-muted-foreground hover:text-destructive"
            onClick={(e) => {
              e.preventDefault()
              onClear()
              setIsOpen(false)
            }}
          >
            <Trash2 className="h-3 w-3 mr-1" />
            Clear
          </Button>
        </div>
        <DropdownMenuSeparator />
        <div className="max-h-64 overflow-y-auto">
          {history.map((entry) => (
            <DropdownMenuItem
              key={entry.id}
              className="flex flex-col items-start gap-1 cursor-pointer py-2"
              onClick={() => {
                onSelect(entry.sql)
                setIsOpen(false)
              }}
            >
              <div className="flex items-center gap-2 w-full">
                {entry.error ? (
                  <XCircle className="h-3 w-3 text-destructive flex-shrink-0" />
                ) : (
                  <CheckCircle className="h-3 w-3 text-green-500 flex-shrink-0" />
                )}
                <code className="text-xs font-mono flex-1 truncate">
                  {truncateSql(entry.sql)}
                </code>
              </div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground ml-5">
                <Clock className="h-3 w-3" />
                <span>{formatTime(entry.executedAt)}</span>
                {entry.durationMs !== undefined && (
                  <>
                    <span>|</span>
                    <span>{entry.durationMs}ms</span>
                  </>
                )}
                {entry.rowCount !== undefined && (
                  <>
                    <span>|</span>
                    <span>{entry.rowCount} rows</span>
                  </>
                )}
              </div>
            </DropdownMenuItem>
          ))}
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
