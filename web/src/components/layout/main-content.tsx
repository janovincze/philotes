"use client"

import { useUIStore } from "@/lib/store/ui-store"
import { cn } from "@/lib/utils"

export function MainContent({ children }: { children: React.ReactNode }) {
  const { sidebarCollapsed } = useUIStore()

  return (
    <main
      className={cn(
        "transition-all duration-300 pt-16",
        "md:pl-64",
        sidebarCollapsed && "md:pl-16"
      )}
    >
      <div className="container py-6">{children}</div>
    </main>
  )
}
