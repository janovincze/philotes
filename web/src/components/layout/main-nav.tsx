"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  Database,
  GitBranch,
  Scale,
  Bell,
  Settings,
  BarChart3,
  ExternalLink,
  type LucideIcon,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"

interface NavItem {
  title: string
  href: string
  icon: LucideIcon
  external?: boolean
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
    title: "Settings",
    href: "/settings",
    icon: Settings,
  },
]

const externalLinks: NavItem[] = [
  {
    title: "Grafana",
    href: "http://localhost:3001",
    icon: BarChart3,
    external: true,
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

      {/* External links section */}
      {externalLinks.length > 0 && (
        <>
          <Separator className="my-2" />
          {!collapsed && (
            <span className="px-3 py-1 text-xs font-medium text-muted-foreground">
              External
            </span>
          )}
          {externalLinks.map((item) => (
            <Button
              key={item.href}
              variant="ghost"
              className={cn(
                "justify-start",
                collapsed && "justify-center px-2"
              )}
              asChild
            >
              <a
                href={item.href}
                target="_blank"
                rel="noopener noreferrer"
              >
                <item.icon className={cn("h-4 w-4", !collapsed && "mr-2")} />
                {!collapsed && (
                  <>
                    <span>{item.title}</span>
                    <ExternalLink className="ml-auto h-3 w-3 text-muted-foreground" />
                  </>
                )}
              </a>
            </Button>
          ))}
        </>
      )}
    </nav>
  )
}
