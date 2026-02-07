"use client"

import { useState, useRef, useEffect } from "react"
import { Plus, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export interface QueryTab {
  id: string
  name: string
  sql: string
}

interface QueryTabBarProps {
  tabs: QueryTab[]
  activeTabId: string
  onSelectTab: (id: string) => void
  onAddTab: () => void
  onCloseTab: (id: string) => void
  onRenameTab: (id: string, name: string) => void
}

export function QueryTabBar({
  tabs,
  activeTabId,
  onSelectTab,
  onAddTab,
  onCloseTab,
  onRenameTab,
}: QueryTabBarProps) {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (editingId && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [editingId])

  const handleDoubleClick = (tab: QueryTab) => {
    setEditingId(tab.id)
    setEditValue(tab.name)
  }

  const handleRenameCommit = () => {
    if (editingId && editValue.trim()) {
      onRenameTab(editingId, editValue.trim())
    }
    setEditingId(null)
  }

  return (
    <div role="tablist" className="flex items-center gap-0.5 border-b px-1 bg-muted/30 overflow-x-auto">
      {tabs.map((tab) => (
        <div
          key={tab.id}
          role="tab"
          id={`query-tab-${tab.id}`}
          aria-selected={tab.id === activeTabId}
          aria-controls={`query-tabpanel-${tab.id}`}
          tabIndex={tab.id === activeTabId ? 0 : -1}
          className={cn(
            "group flex items-center gap-1 px-3 py-1.5 text-sm cursor-pointer border-b-2 transition-colors shrink-0 outline-none focus-visible:ring-2 focus-visible:ring-ring",
            tab.id === activeTabId
              ? "border-primary bg-background text-foreground"
              : "border-transparent text-muted-foreground hover:text-foreground hover:bg-muted/50"
          )}
          onClick={() => onSelectTab(tab.id)}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault()
              onSelectTab(tab.id)
            }
          }}
        >
          {editingId === tab.id ? (
            <input
              ref={inputRef}
              value={editValue}
              onChange={(e) => setEditValue(e.target.value)}
              onBlur={handleRenameCommit}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRenameCommit()
                if (e.key === "Escape") setEditingId(null)
              }}
              className="w-24 bg-transparent border-b border-primary outline-none text-sm"
              onClick={(e) => e.stopPropagation()}
            />
          ) : (
            <span
              className="truncate max-w-[120px]"
              onDoubleClick={() => handleDoubleClick(tab)}
              title={`Double-click to rename: ${tab.name}`}
            >
              {tab.name}
            </span>
          )}
          {tabs.length > 1 && (
            <button
              type="button"
              className={cn(
                "ml-1 p-0.5 rounded hover:bg-accent transition-opacity",
                tab.id === activeTabId ? "opacity-60 hover:opacity-100" : "opacity-0 group-hover:opacity-60 hover:!opacity-100"
              )}
              onClick={(e) => {
                e.stopPropagation()
                onCloseTab(tab.id)
              }}
              aria-label={`Close ${tab.name}`}
            >
              <X className="h-3 w-3" />
            </button>
          )}
        </div>
      ))}
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 shrink-0"
        onClick={onAddTab}
        aria-label="New query tab"
        title="New query tab"
      >
        <Plus className="h-3.5 w-3.5" />
      </Button>
    </div>
  )
}
