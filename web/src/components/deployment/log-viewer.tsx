"use client"

import { useState, useRef, useEffect } from "react"
import { ChevronDown, ChevronRight, Terminal, Download } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { cn } from "@/lib/utils"
import type { DeploymentLogMessage } from "@/lib/api/types"

interface LogViewerProps {
  logs: DeploymentLogMessage[]
  groupByStep?: boolean
  autoScroll?: boolean
  maxHeight?: string
  className?: string
}

interface LogEntryProps {
  log: DeploymentLogMessage
}

function LogEntry({ log }: LogEntryProps) {
  return (
    <div
      className={cn(
        "py-0.5 font-mono text-xs",
        log.level === "error" && "text-red-500",
        log.level === "warn" && "text-yellow-500",
        log.level === "debug" && "text-muted-foreground"
      )}
    >
      <span className="text-muted-foreground">
        [{new Date(log.timestamp).toLocaleTimeString()}]
      </span>{" "}
      {log.step && (
        <span className="text-primary">[{log.step}]</span>
      )}{" "}
      {log.message}
    </div>
  )
}

interface LogGroupProps {
  stepId: string
  logs: DeploymentLogMessage[]
  defaultOpen?: boolean
}

function LogGroup({ stepId, logs, defaultOpen = false }: LogGroupProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger className="flex items-center gap-2 w-full py-1 px-2 hover:bg-muted/50 rounded text-sm">
        {isOpen ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
        <span className="font-medium capitalize">{stepId}</span>
        <span className="text-muted-foreground text-xs">({logs.length} entries)</span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="pl-5 border-l-2 border-muted ml-1.5 mt-1">
          {logs.map((log, index) => (
            <LogEntry key={index} log={log} />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

export function LogViewer({
  logs,
  groupByStep = false,
  autoScroll = true,
  maxHeight = "400px",
  className,
}: LogViewerProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  const downloadLogs = () => {
    const content = logs
      .map((log) => {
        const time = new Date(log.timestamp).toISOString()
        const level = log.level?.toUpperCase() || "INFO"
        const step = log.step ? `[${log.step}]` : ""
        return `${time} ${level} ${step} ${log.message}`
      })
      .join("\n")

    const blob = new Blob([content], { type: "text/plain" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `deployment-logs-${new Date().toISOString().split("T")[0]}.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  // Group logs by step if requested
  const groupedLogs = groupByStep
    ? logs.reduce<Record<string, DeploymentLogMessage[]>>((acc, log) => {
        const step = log.step || "general"
        if (!acc[step]) acc[step] = []
        acc[step].push(log)
        return acc
      }, {})
    : null

  return (
    <div className={cn("rounded-lg border bg-muted/30", className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/50">
        <div className="flex items-center gap-2 text-sm font-medium">
          <Terminal className="h-4 w-4" />
          <span>Deployment Logs</span>
          <span className="text-muted-foreground text-xs">({logs.length} entries)</span>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={downloadLogs}
          disabled={logs.length === 0}
        >
          <Download className="h-4 w-4 mr-1" />
          Download
        </Button>
      </div>

      {/* Log content */}
      <ScrollArea
        ref={scrollRef}
        className="p-3"
        style={{ maxHeight }}
      >
        {logs.length === 0 ? (
          <div className="text-center text-muted-foreground py-8 text-sm">
            Waiting for logs...
          </div>
        ) : groupByStep && groupedLogs ? (
          <div className="space-y-1">
            {Object.entries(groupedLogs).map(([stepId, stepLogs]) => (
              <LogGroup
                key={stepId}
                stepId={stepId}
                logs={stepLogs}
                defaultOpen={stepId === Object.keys(groupedLogs).pop()}
              />
            ))}
          </div>
        ) : (
          <div className="space-y-0">
            {logs.map((log, index) => (
              <LogEntry key={index} log={log} />
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  )
}
