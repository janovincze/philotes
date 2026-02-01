"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { ArrowLeft, Database, Loader2, Eye, EyeOff } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useCreateSource } from "@/lib/hooks/use-sources"

interface FormData {
  name: string
  host: string
  port: number
  database_name: string
  username: string
  password: string
  ssl_mode: string
}

const initialFormData: FormData = {
  name: "",
  host: "",
  port: 5432,
  database_name: "",
  username: "",
  password: "",
  ssl_mode: "prefer",
}

export default function NewSourcePage() {
  const router = useRouter()
  const createSource = useCreateSource()
  const [formData, setFormData] = useState<FormData>(initialFormData)
  const [showPassword, setShowPassword] = useState(false)
  const [errors, setErrors] = useState<Partial<Record<keyof FormData, string>>>(
    {}
  )

  const handleChange = (field: keyof FormData, value: string | number) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
    // Clear error when field is edited
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }))
    }
  }

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof FormData, string>> = {}

    if (!formData.name.trim()) {
      newErrors.name = "Name is required"
    }
    if (!formData.host.trim()) {
      newErrors.host = "Host is required"
    }
    if (!formData.port || formData.port < 1 || formData.port > 65535) {
      newErrors.port = "Port must be between 1 and 65535"
    }
    if (!formData.database_name.trim()) {
      newErrors.database_name = "Database name is required"
    }
    if (!formData.username.trim()) {
      newErrors.username = "Username is required"
    }
    if (!formData.password) {
      newErrors.password = "Password is required"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validate()) {
      return
    }

    try {
      const result = await createSource.mutateAsync({
        name: formData.name,
        type: "postgresql",
        host: formData.host,
        port: formData.port,
        database_name: formData.database_name,
        username: formData.username,
        password: formData.password,
        ssl_mode: formData.ssl_mode,
      })
      router.push(`/sources/${result.id}`)
    } catch (error) {
      // Error is handled by the mutation
      console.error("Failed to create source:", error)
    }
  }

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Link
        href="/sources"
        className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Sources
      </Link>

      {/* Header */}
      <div className="flex items-center gap-4">
        <div className="rounded-lg bg-primary/10 p-3">
          <Database className="h-8 w-8 text-primary" />
        </div>
        <div>
          <h1 className="text-2xl font-bold">Add Data Source</h1>
          <p className="text-muted-foreground">
            Connect a PostgreSQL database for CDC replication
          </p>
        </div>
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit}>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Connection Details</CardTitle>
            <CardDescription>
              Enter the connection details for your PostgreSQL database
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Name */}
            <div className="space-y-2">
              <Label htmlFor="name">Source Name</Label>
              <Input
                id="name"
                placeholder="e.g., Production Database"
                value={formData.name}
                onChange={(e) => handleChange("name", e.target.value)}
                className={errors.name ? "border-destructive" : ""}
              />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name}</p>
              )}
              <p className="text-sm text-muted-foreground">
                A friendly name to identify this data source
              </p>
            </div>

            {/* Host and Port */}
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="sm:col-span-2 space-y-2">
                <Label htmlFor="host">Host</Label>
                <Input
                  id="host"
                  placeholder="e.g., localhost or db.example.com"
                  value={formData.host}
                  onChange={(e) => handleChange("host", e.target.value)}
                  className={errors.host ? "border-destructive" : ""}
                />
                {errors.host && (
                  <p className="text-sm text-destructive">{errors.host}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="port">Port</Label>
                <Input
                  id="port"
                  type="number"
                  placeholder="5432"
                  value={formData.port}
                  onChange={(e) =>
                    handleChange("port", parseInt(e.target.value) || 0)
                  }
                  className={errors.port ? "border-destructive" : ""}
                />
                {errors.port && (
                  <p className="text-sm text-destructive">{errors.port}</p>
                )}
              </div>
            </div>

            {/* Database Name */}
            <div className="space-y-2">
              <Label htmlFor="database_name">Database Name</Label>
              <Input
                id="database_name"
                placeholder="e.g., myapp_production"
                value={formData.database_name}
                onChange={(e) => handleChange("database_name", e.target.value)}
                className={errors.database_name ? "border-destructive" : ""}
              />
              {errors.database_name && (
                <p className="text-sm text-destructive">
                  {errors.database_name}
                </p>
              )}
            </div>

            {/* Username */}
            <div className="space-y-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                placeholder="e.g., postgres"
                value={formData.username}
                onChange={(e) => handleChange("username", e.target.value)}
                className={errors.username ? "border-destructive" : ""}
              />
              {errors.username && (
                <p className="text-sm text-destructive">{errors.username}</p>
              )}
            </div>

            {/* Password */}
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  placeholder="Enter password"
                  value={formData.password}
                  onChange={(e) => handleChange("password", e.target.value)}
                  className={errors.password ? "border-destructive pr-10" : "pr-10"}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                  onClick={() => setShowPassword(!showPassword)}
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4 text-muted-foreground" />
                  ) : (
                    <Eye className="h-4 w-4 text-muted-foreground" />
                  )}
                </Button>
              </div>
              {errors.password && (
                <p className="text-sm text-destructive">{errors.password}</p>
              )}
            </div>

            {/* SSL Mode */}
            <div className="space-y-2">
              <Label htmlFor="ssl_mode">SSL Mode</Label>
              <Select
                value={formData.ssl_mode}
                onValueChange={(value) => handleChange("ssl_mode", value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select SSL mode" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="disable">Disable</SelectItem>
                  <SelectItem value="prefer">Prefer</SelectItem>
                  <SelectItem value="require">Require</SelectItem>
                  <SelectItem value="verify-ca">Verify CA</SelectItem>
                  <SelectItem value="verify-full">Verify Full</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-sm text-muted-foreground">
                SSL/TLS encryption mode for the connection
              </p>
            </div>

            {/* Error message */}
            {createSource.isError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                Failed to create source. Please check your connection details
                and try again.
              </div>
            )}

            {/* Submit buttons */}
            <div className="flex justify-end gap-3 pt-4">
              <Button type="button" variant="outline" asChild>
                <Link href="/sources">Cancel</Link>
              </Button>
              <Button type="submit" disabled={createSource.isPending}>
                {createSource.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  "Create Source"
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      </form>
    </div>
  )
}
