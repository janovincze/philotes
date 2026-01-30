"use client"

import { useState } from "react"
import { AlertTriangle, Loader2, Server, Network, HardDrive } from "lucide-react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import type { CreatedResource } from "@/lib/api/types"

interface CancelDialogProps {
  resources?: CreatedResource[]
  onCancel: () => Promise<void>
  isLoading?: boolean
  disabled?: boolean
  children?: React.ReactNode
}

function ResourceIcon({ type }: { type: string }) {
  const typeLower = type.toLowerCase()
  if (typeLower.includes("server") || typeLower.includes("instance") || typeLower.includes("compute")) {
    return <Server className="h-4 w-4" />
  }
  if (typeLower.includes("network") || typeLower.includes("subnet") || typeLower.includes("firewall")) {
    return <Network className="h-4 w-4" />
  }
  if (typeLower.includes("volume") || typeLower.includes("storage")) {
    return <HardDrive className="h-4 w-4" />
  }
  return <Server className="h-4 w-4" />
}

export function CancelDialog({
  resources = [],
  onCancel,
  isLoading = false,
  disabled = false,
  children,
}: CancelDialogProps) {
  const [open, setOpen] = useState(false)
  const [isCanceling, setIsCanceling] = useState(false)

  const handleCancel = async () => {
    setIsCanceling(true)
    try {
      await onCancel()
      setOpen(false)
    } finally {
      setIsCanceling(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>
        {children || (
          <Button variant="destructive" disabled={disabled || isLoading}>
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Canceling...
              </>
            ) : (
              "Cancel Deployment"
            )}
          </Button>
        )}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-yellow-500" />
            Cancel Deployment?
          </AlertDialogTitle>
          <AlertDialogDescription>
            This will stop the deployment process. Any resources that have already been
            created will be destroyed.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {resources.length > 0 && (
          <div className="py-4">
            <h4 className="text-sm font-medium mb-2">
              The following resources will be destroyed:
            </h4>
            <ScrollArea className="h-[150px] rounded-md border p-3">
              <ul className="space-y-2">
                {resources.map((resource, index) => (
                  <li
                    key={index}
                    className="flex items-center gap-2 text-sm text-muted-foreground"
                  >
                    <ResourceIcon type={resource.type} />
                    <span className="font-medium text-foreground">{resource.name}</span>
                    <span className="text-xs">({resource.type})</span>
                    {resource.region && (
                      <span className="text-xs text-muted-foreground">
                        in {resource.region}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </ScrollArea>
          </div>
        )}

        {resources.length === 0 && (
          <div className="py-4">
            <p className="text-sm text-muted-foreground">
              No resources have been created yet. The deployment can be safely canceled.
            </p>
          </div>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel disabled={isCanceling}>Keep Running</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleCancel}
            disabled={isCanceling}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {isCanceling ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Canceling...
              </>
            ) : (
              "Cancel Deployment"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
