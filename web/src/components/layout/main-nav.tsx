"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  Database,
  GitBranch,
  Bell,
  Settings,
  type LucideIcon,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"

interface NavItem {
  title: string
  href: string
  icon: LucideIcon
}

const navItems: NavItem[] = [
  {
    title: "Dashboard",
    href: "/",
    icon: LayoutDashboard,
  },
  {
    title: "Sources",
    href: "/sources",
    icon: Database,
  },
  {
    title: "Pipelines",
    href: "/pipelines",
    icon: GitBranch,
  },
  {
    title: "Alerts",
    href: "/alerts",
    icon: Bell,
  },
  {
    title: "Settings",
    href: "/settings",
    icon: Settings,
  },
]

interface MainNavProps {
  collapsed?: boolean
}

export function MainNav({ collapsed = false }: MainNavProps) {
  const pathname = usePathname()

  return (
    <nav className="flex flex-col gap-1 px-2">
      {navItems.map((item) => {
        // Use startsWith for nested route highlighting (e.g., /sources/new highlights "Sources")
        // Exact match for root path to avoid always being active
        const isActive = pathname === item.href ||
          (item.href !== "/" && pathname.startsWith(item.href))

        return (
          <Button
            key={item.href}
            variant={isActive ? "secondary" : "ghost"}
            className={cn(
              "justify-start",
              collapsed && "justify-center px-2"
            )}
            asChild
          >
            <Link href={item.href}>
              <item.icon className={cn("h-4 w-4", !collapsed && "mr-2")} />
              {!collapsed && <span>{item.title}</span>}
            </Link>
          </Button>
        )
      })}
    </nav>
  )
}
