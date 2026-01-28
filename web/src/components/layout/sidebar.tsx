"use client"

import { ChevronLeft, ChevronRight } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { MainNav } from "./main-nav"
import { useUIStore } from "@/lib/store/ui-store"

export function Sidebar() {
  const { sidebarCollapsed, toggleSidebar } = useUIStore()

  return (
    <aside
      className={cn(
        "fixed left-0 top-0 z-40 h-screen border-r bg-background transition-all duration-300",
        "hidden md:block", // Hide on mobile, show on medium screens and up
        sidebarCollapsed ? "w-16" : "w-64"
      )}
    >
      <div className="flex h-full flex-col">
        {/* Logo */}
        <div className={cn(
          "flex h-16 items-center border-b px-4",
          sidebarCollapsed && "justify-center px-2"
        )}>
          {sidebarCollapsed ? (
            <span className="text-xl font-bold">P</span>
          ) : (
            <span className="text-xl font-bold">Philotes</span>
          )}
        </div>

        {/* Navigation */}
        <div className="flex-1 overflow-y-auto py-4">
          <MainNav collapsed={sidebarCollapsed} />
        </div>

        {/* Collapse toggle */}
        <div className="border-t p-2">
          <Button
            variant="ghost"
            size="icon"
            className="w-full"
            onClick={toggleSidebar}
          >
            {sidebarCollapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    </aside>
  )
}
