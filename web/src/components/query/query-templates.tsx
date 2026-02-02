"use client"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { FileCode2 } from "lucide-react"

interface QueryTemplate {
  name: string
  description: string
  sql: string
}

const defaultTemplates: QueryTemplate[] = [
  {
    name: "Show Catalogs",
    description: "List all available catalogs",
    sql: "SHOW CATALOGS",
  },
  {
    name: "Show Schemas",
    description: "List schemas in iceberg catalog",
    sql: "SHOW SCHEMAS FROM iceberg",
  },
  {
    name: "Show Tables",
    description: "List tables in public schema",
    sql: "SHOW TABLES FROM iceberg.public",
  },
  {
    name: "Sample Customers",
    description: "View first 10 customers",
    sql: `SELECT *
FROM iceberg.public.customers
LIMIT 10`,
  },
  {
    name: "Sample Orders",
    description: "View recent orders",
    sql: `SELECT *
FROM iceberg.public.orders
ORDER BY created_at DESC
LIMIT 10`,
  },
  {
    name: "Customer Order Count",
    description: "Count orders per customer",
    sql: `SELECT
  c.first_name,
  c.last_name,
  COUNT(o.id) as order_count
FROM iceberg.public.customers c
LEFT JOIN iceberg.public.orders o ON c.id = o.customer_id
GROUP BY c.id, c.first_name, c.last_name
ORDER BY order_count DESC
LIMIT 10`,
  },
  {
    name: "Table Row Counts",
    description: "Count rows in each table",
    sql: `-- Run these queries one at a time:
-- SELECT COUNT(*) as customer_count FROM iceberg.public.customers
-- SELECT COUNT(*) as order_count FROM iceberg.public.orders
-- SELECT COUNT(*) as product_count FROM iceberg.public.products
SHOW TABLES FROM iceberg.public`,
  },
]

interface QueryTemplatesProps {
  onSelect: (sql: string) => void
  customTemplates?: QueryTemplate[]
}

export function QueryTemplates({ onSelect, customTemplates }: QueryTemplatesProps) {
  const templates = customTemplates || defaultTemplates

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <FileCode2 className="h-4 w-4 mr-2" />
          Templates
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-64">
        <DropdownMenuLabel>Query Templates</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {templates.map((template, index) => (
          <DropdownMenuItem
            key={index}
            onClick={() => onSelect(template.sql)}
            className="flex flex-col items-start"
          >
            <span className="font-medium">{template.name}</span>
            <span className="text-xs text-muted-foreground">
              {template.description}
            </span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
