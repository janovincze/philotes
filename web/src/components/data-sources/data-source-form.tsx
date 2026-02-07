"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { QueryDataSource, QueryDataSourceType } from "@/lib/api/types"

const dataSourceSchema = z.object({
  name: z.string().min(1, "Name is required").max(255),
  type: z.enum(["postgresql", "mysql"]),
  catalog_name: z
    .string()
    .min(1, "Catalog name is required")
    .regex(
      /^[a-zA-Z_][a-zA-Z0-9_]*$/,
      "Must contain only letters, numbers, and underscores, starting with a letter or underscore"
    ),
  host: z.string().min(1, "Host is required"),
  port: z.number().int().min(1).max(65535),
  database_name: z.string().min(1, "Database name is required"),
  username: z.string().min(1, "Username is required"),
  password: z.string().optional(),
  ssl_mode: z.string().optional(),
})

type DataSourceFormValues = z.infer<typeof dataSourceSchema>

interface DataSourceFormProps {
  onSubmit: (values: DataSourceFormValues) => void
  defaultValues?: Partial<DataSourceFormValues>
  existingSource?: QueryDataSource
  isSubmitting?: boolean
}

export function DataSourceForm({
  onSubmit,
  defaultValues,
  existingSource,
  isSubmitting,
}: DataSourceFormProps) {
  const isEdit = !!existingSource

  const form = useForm<DataSourceFormValues>({
    resolver: zodResolver(dataSourceSchema),
    defaultValues: {
      name: existingSource?.name ?? defaultValues?.name ?? "",
      type: (existingSource?.type as QueryDataSourceType) ?? defaultValues?.type ?? "postgresql",
      catalog_name: existingSource?.catalog_name ?? defaultValues?.catalog_name ?? "",
      host: existingSource?.host ?? defaultValues?.host ?? "",
      port: existingSource?.port ?? defaultValues?.port ?? 5432,
      database_name: existingSource?.database_name ?? defaultValues?.database_name ?? "",
      username: existingSource?.username ?? defaultValues?.username ?? "",
      password: defaultValues?.password ?? "",
      ssl_mode: existingSource?.ssl_mode ?? defaultValues?.ssl_mode ?? "prefer",
    },
  })

  const selectedType = form.watch("type")

  // Update default port when type changes
  const handleTypeChange = (value: string) => {
    form.setValue("type", value as QueryDataSourceType)
    const currentPort = form.getValues("port")
    if (currentPort === 5432 || currentPort === 3306) {
      form.setValue("port", value === "mysql" ? 3306 : 5432)
    }
  }

  const handleFormSubmit = (values: DataSourceFormValues) => {
    if (!isEdit && !values.password) {
      form.setError("password", { message: "Password is required" })
      return
    }
    onSubmit(values)
  }

  return (
    <form onSubmit={form.handleSubmit(handleFormSubmit)} className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            placeholder="production_db"
            {...form.register("name")}
          />
          {form.formState.errors.name && (
            <p className="text-sm text-destructive">{form.formState.errors.name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="type">Type</Label>
          <Select
            value={selectedType}
            onValueChange={handleTypeChange}
            disabled={isEdit}
          >
            <SelectTrigger id="type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="postgresql">PostgreSQL</SelectItem>
              <SelectItem value="mysql">MySQL</SelectItem>
            </SelectContent>
          </Select>
          {form.formState.errors.type && (
            <p className="text-sm text-destructive">{form.formState.errors.type.message}</p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="catalog_name">Catalog Name</Label>
        <Input
          id="catalog_name"
          placeholder="pg_production"
          disabled={isEdit}
          {...form.register("catalog_name")}
        />
        <p className="text-xs text-muted-foreground">
          Used in queries as: <code>{form.watch("catalog_name") || "catalog"}.schema.table</code>
        </p>
        {form.formState.errors.catalog_name && (
          <p className="text-sm text-destructive">{form.formState.errors.catalog_name.message}</p>
        )}
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="col-span-2 space-y-2">
          <Label htmlFor="host">Host</Label>
          <Input
            id="host"
            placeholder="db.example.com"
            {...form.register("host")}
          />
          {form.formState.errors.host && (
            <p className="text-sm text-destructive">{form.formState.errors.host.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="port">Port</Label>
          <Input
            id="port"
            type="number"
            {...form.register("port", { valueAsNumber: true })}
          />
          {form.formState.errors.port && (
            <p className="text-sm text-destructive">{form.formState.errors.port.message}</p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="database_name">Database Name</Label>
        <Input
          id="database_name"
          placeholder="myapp"
          {...form.register("database_name")}
        />
        {form.formState.errors.database_name && (
          <p className="text-sm text-destructive">{form.formState.errors.database_name.message}</p>
        )}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="username">Username</Label>
          <Input
            id="username"
            placeholder="readonly_user"
            {...form.register("username")}
          />
          {form.formState.errors.username && (
            <p className="text-sm text-destructive">{form.formState.errors.username.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            placeholder={isEdit ? "Leave empty to keep current" : ""}
            {...form.register("password")}
          />
          {form.formState.errors.password && (
            <p className="text-sm text-destructive">{form.formState.errors.password.message}</p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="ssl_mode">SSL Mode</Label>
        <Select
          value={form.watch("ssl_mode") || "prefer"}
          onValueChange={(val) => form.setValue("ssl_mode", val)}
        >
          <SelectTrigger id="ssl_mode">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="disable">Disable</SelectItem>
            <SelectItem value="prefer">Prefer</SelectItem>
            <SelectItem value="require">Require</SelectItem>
            <SelectItem value="verify-ca">Verify CA</SelectItem>
            <SelectItem value="verify-full">Verify Full</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="flex justify-end gap-2 pt-4">
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting
            ? isEdit
              ? "Updating..."
              : "Creating..."
            : isEdit
              ? "Update Data Source"
              : "Create Data Source"}
        </Button>
      </div>
    </form>
  )
}
