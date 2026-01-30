"use client"

import { useState } from "react"
import { Link2, Check, Copy } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

interface ShareButtonProps {
  deploymentId: string
  variant?: "default" | "outline" | "ghost"
  size?: "default" | "sm" | "lg" | "icon"
  className?: string
}

export function ShareButton({
  deploymentId,
  variant = "outline",
  size = "default",
  className,
}: ShareButtonProps) {
  const [copied, setCopied] = useState(false)
  const [open, setOpen] = useState(false)

  const shareUrl = typeof window !== "undefined"
    ? `${window.location.origin}/install/deploy/${deploymentId}`
    : `/install/deploy/${deploymentId}`

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error("Failed to copy:", err)
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant={variant} size={size} className={className}>
          <Link2 className="h-4 w-4 mr-2" />
          Share
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80" align="end">
        <div className="space-y-3">
          <div>
            <h4 className="font-medium text-sm">Share Deployment Status</h4>
            <p className="text-xs text-muted-foreground mt-1">
              Anyone with this link can view the deployment progress.
            </p>
          </div>
          <div className="flex gap-2">
            <Input
              value={shareUrl}
              readOnly
              className="text-xs"
              onClick={(e) => e.currentTarget.select()}
            />
            <Button
              variant="outline"
              size="icon"
              onClick={copyToClipboard}
              className={cn(
                "flex-shrink-0 transition-colors",
                copied && "text-green-500 border-green-500"
              )}
            >
              {copied ? (
                <Check className="h-4 w-4" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
