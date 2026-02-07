"use client"

import { useEffect, useState } from "react"
import { Search, X } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

interface SchemaSearchProps {
  value: string
  onChange: (value: string) => void
}

export function SchemaSearch({ value, onChange }: SchemaSearchProps) {
  const [local, setLocal] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => onChange(local), 300)
    return () => clearTimeout(timer)
  }, [local, onChange])

  useEffect(() => {
    setLocal(value)
  }, [value])

  return (
    <div className="relative">
      <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
      <Input
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        placeholder="Search tables..."
        className="h-8 pl-8 pr-8 text-sm"
      />
      {local && (
        <Button
          variant="ghost"
          size="icon"
          aria-label="Clear search"
          className="absolute right-0 top-0 h-8 w-8"
          onClick={() => {
            setLocal("")
            onChange("")
          }}
        >
          <X className="h-3.5 w-3.5" />
        </Button>
      )}
    </div>
  )
}
