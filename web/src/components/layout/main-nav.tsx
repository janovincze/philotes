"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  Database,
  GitBranch,
  Scale,
  Bell,
  BarChart3,
  Settings,
  Terminal,
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
    title: "Query",
    href: "/query",
    icon: Terminal,
  },
  {
    title: "Scaling",
    href: "/scaling",
    icon: Scale,
  },
  {
    title: "Alerts",
    href: "/alerts",
    icon: Bell,
  },
  {
    title: "Monitoring",
    href: "http://localhost:3000",
    icon: BarChart3,
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
        const isExternal = item.href.startsWith("http://") || item.href.startsWith("https://")
        // Use startsWith for nested route highlighting (e.g., /sources/new highlights "Sources")
        // Exact match for root path to avoid always being active
        const isActive = !isExternal && (pathname === item.href ||
          (item.href !== "/" && pathname.startsWith(item.href)))

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
            {isExternal ? (
              <a href={item.href} target="_blank" rel="noopener noreferrer">
                <item.icon className={cn("h-4 w-4", !collapsed && "mr-2")} />
                {!collapsed && <span>{item.title}</span>}
              </a>
            ) : (
              <Link href={item.href}>
                <item.icon className={cn("h-4 w-4", !collapsed && "mr-2")} />
                {!collapsed && <span>{item.title}</span>}
              </Link>
            )}
          </Button>
        )
      })}
    </nav>
  )
}
