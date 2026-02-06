"use client"

import { useState, type FormEvent } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { ArrowLeft, Database, Loader2 } from "lucide-react"
import { toast } from "sonner"
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
import type { CreateSourceInput } from "@/lib/api/types"

const SSL_MODE_OPTIONS = [
  { value: "disable", label: "Disable" },
  { value: "prefer", label: "Prefer (default)" },
  { value: "require", label: "Require" },
]

interface FormState {
  name: string
  host: string
  port: number
  database_name: string
  username: string
  password: string
  ssl_mode: string
}

const initialFormState: FormState = {
  name: "",
  host: "",
  port: 5432,
  database_name: "",
  username: "",
  password: "",
  ssl_mode: "prefer",
}

export default function SourceCreatePage() {
  const router = useRouter()
  const createSource = useCreateSource()
  const [formData, setFormData] = useState<FormState>(initialFormState)

  const updateField = <K extends keyof FormState>(
    field: K,
    value: FormState[K]
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
  }

  const isFormValid =
    formData.name.trim() !== "" &&
    formData.host.trim() !== "" &&
    formData.database_name.trim() !== "" &&
    formData.username.trim() !== "" &&
    formData.password.trim() !== "" &&
    formData.port > 0 &&
    formData.port <= 65535

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()

    if (!isFormValid) return

    const input: CreateSourceInput = {
      name: formData.name.trim(),
      type: "postgresql",
      host: formData.host.trim(),
      port: formData.port,
      database_name: formData.database_name.trim(),
      username: formData.username.trim(),
      password: formData.password,
      ssl_mode: formData.ssl_mode,
    }

    try {
      const newSource = await createSource.mutateAsync(input)
      toast.success("Source created", {
        description: `"${newSource.name}" has been added successfully.`,
      })
      router.push(`/sources/${newSource.id}`)
    } catch (err) {
      toast.error("Failed to create source", {
        description:
          err instanceof Error ? err.message : "An unexpected error occurred",
      })
    }
  }

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Button variant="outline" size="sm" asChild>
          <Link href="/sources">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Link>
        </Button>
        <div>
          <h1 className="text-3xl font-bold">Add Data Source</h1>
          <p className="text-muted-foreground">
            Connect a new PostgreSQL database
          </p>
        </div>
      </div>

      {/* Form card */}
      <Card>
        <CardHeader>
          <div className="flex items-start gap-4">
            <div className="rounded-lg bg-primary/10 p-2">
              <Database className="h-6 w-6 text-primary" />
            </div>
            <div>
              <CardTitle className="text-lg">Connection Details</CardTitle>
              <CardDescription>
                Enter your PostgreSQL database credentials. You can test the
                connection after creating the source.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid gap-4 sm:grid-cols-2">
              {/* Source Name */}
              <div className="sm:col-span-2">
                <Label htmlFor="name">Source Name</Label>
                <Input
                  id="name"
                  placeholder="e.g., Production Database"
                  value={formData.name}
                  onChange={(e) => updateField("name", e.target.value)}
                  className="mt-1"
                />
                <p className="text-xs text-muted-foreground mt-1">
                  A friendly name to identify this database connection
                </p>
              </div>

              {/* Host */}
              <div>
                <Label htmlFor="host">Host</Label>
                <Input
                  id="host"
                  placeholder="localhost or db.example.com"
                  value={formData.host}
                  onChange={(e) => updateField("host", e.target.value)}
                  className="mt-1"
                />
              </div>

              {/* Port */}
              <div>
                <Label htmlFor="port">Port</Label>
                <Input
                  id="port"
                  type="number"
                  placeholder="5432"
                  value={formData.port}
                  onChange={(e) => {
                    const parsed = parseInt(e.target.value, 10)
                    updateField("port", Number.isNaN(parsed) ? 5432 : parsed)
                  }}
                  className="mt-1"
                  min={1}
                  max={65535}
                />
              </div>

              {/* Database Name */}
              <div>
                <Label htmlFor="database_name">Database Name</Label>
                <Input
                  id="database_name"
                  placeholder="myapp"
                  value={formData.database_name}
                  onChange={(e) =>
                    updateField("database_name", e.target.value)
                  }
                  className="mt-1"
                />
              </div>

              {/* SSL Mode */}
              <div>
                <Label htmlFor="ssl_mode">SSL Mode</Label>
                <Select
                  value={formData.ssl_mode}
                  onValueChange={(value) => updateField("ssl_mode", value)}
                >
                  <SelectTrigger className="mt-1">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SSL_MODE_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Username */}
              <div>
                <Label htmlFor="username">Username</Label>
                <Input
                  id="username"
                  placeholder="postgres"
                  value={formData.username}
                  onChange={(e) => updateField("username", e.target.value)}
                  className="mt-1"
                />
              </div>

              {/* Password */}
              <div>
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="********"
                  value={formData.password}
                  onChange={(e) => updateField("password", e.target.value)}
                  className="mt-1"
                />
              </div>
            </div>

            {/* Form actions */}
            <div className="flex items-center justify-between border-t pt-6">
              <Button variant="outline" type="button" asChild>
                <Link href="/sources">Cancel</Link>
              </Button>
              <Button
                type="submit"
                disabled={!isFormValid || createSource.isPending}
              >
                {createSource.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    <Database className="mr-2 h-4 w-4" />
                    Create Source
                  </>
                )}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
