"use client"

import { useState } from "react"
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
import { Badge } from "@/components/ui/badge"
import { ArrowLeft, ArrowRight, Database, CheckCircle2, XCircle, Loader2 } from "lucide-react"
import { useCreateSource, useTestSourceConnection } from "@/lib/hooks/use-sources"
import type { Source } from "@/lib/api/types"
import type { SourceFormData } from "./setup-wizard"

interface StepConnectProps {
  formData: SourceFormData
  onFormDataChange: (data: Partial<SourceFormData>) => void
  source: Source | null
  onSourceCreated: (source: Source) => void
  connectionTested: boolean
  onConnectionTested: (tested: boolean) => void
  onNext: () => void
  onBack: () => void
}

const SSL_MODE_OPTIONS = [
  { value: "disable", label: "Disable" },
  { value: "prefer", label: "Prefer (default)" },
  { value: "require", label: "Require" },
  { value: "verify-ca", label: "Verify CA" },
  { value: "verify-full", label: "Verify Full" },
]

export function StepConnect({
  formData,
  onFormDataChange,
  source,
  onSourceCreated,
  connectionTested,
  onConnectionTested,
  onNext,
  onBack,
}: StepConnectProps) {
  const [testResult, setTestResult] = useState<{
    success: boolean
    message: string
    serverInfo?: string
  } | null>(null)

  const createSource = useCreateSource()
  const testConnection = useTestSourceConnection()

  const isFormValid =
    formData.name.trim() !== "" &&
    formData.host.trim() !== "" &&
    formData.database_name.trim() !== "" &&
    formData.username.trim() !== "" &&
    formData.password.trim() !== "" &&
    formData.port > 0 &&
    formData.port <= 65535

  const handleTestConnection = async () => {
    if (!isFormValid) return

    setTestResult(null)
    onConnectionTested(false)

    try {
      // First create the source if it doesn't exist
      let currentSource = source
      if (!currentSource) {
        currentSource = await createSource.mutateAsync({
          name: formData.name,
          type: "postgresql",
          host: formData.host,
          port: formData.port,
          database_name: formData.database_name,
          username: formData.username,
          password: formData.password,
          ssl_mode: formData.ssl_mode,
        })
        onSourceCreated(currentSource)
      }

      // Then test the connection
      const result = await testConnection.mutateAsync(currentSource.id)
      setTestResult({
        success: result.success,
        message: result.message,
        serverInfo: result.server_info,
      })
      onConnectionTested(result.success)
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : "Connection failed",
      })
      onConnectionTested(false)
    }
  }

  const isLoading = createSource.isPending || testConnection.isPending

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Database className="h-5 w-5 text-primary" />
          <h2 className="text-xl font-semibold">Connect Your Database</h2>
        </div>
        <p className="text-sm text-muted-foreground">
          Enter your PostgreSQL database credentials. We&apos;ll test the connection before proceeding.
        </p>
      </div>

      {/* Form */}
      <div className="grid gap-4 sm:grid-cols-2">
        {/* Source Name */}
        <div className="sm:col-span-2">
          <Label htmlFor="name">Source Name</Label>
          <Input
            id="name"
            placeholder="e.g., Production Database"
            value={formData.name}
            onChange={(e) => onFormDataChange({ name: e.target.value })}
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
            onChange={(e) => onFormDataChange({ host: e.target.value })}
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
            onChange={(e) => onFormDataChange({ port: parseInt(e.target.value) || 5432 })}
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
            onChange={(e) => onFormDataChange({ database_name: e.target.value })}
            className="mt-1"
          />
        </div>

        {/* SSL Mode */}
        <div>
          <Label htmlFor="ssl_mode">SSL Mode</Label>
          <Select
            value={formData.ssl_mode}
            onValueChange={(value) => onFormDataChange({ ssl_mode: value })}
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
            onChange={(e) => onFormDataChange({ username: e.target.value })}
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
            onChange={(e) => onFormDataChange({ password: e.target.value })}
            className="mt-1"
          />
        </div>
      </div>

      {/* Test Connection Result */}
      {testResult && (
        <div
          className={`flex items-start gap-3 p-4 rounded-lg ${
            testResult.success
              ? "bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800"
              : "bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-800"
          }`}
        >
          {testResult.success ? (
            <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400 shrink-0 mt-0.5" />
          ) : (
            <XCircle className="h-5 w-5 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
          )}
          <div className="space-y-1">
            <p
              className={`text-sm font-medium ${
                testResult.success
                  ? "text-green-800 dark:text-green-200"
                  : "text-red-800 dark:text-red-200"
              }`}
            >
              {testResult.message}
            </p>
            {testResult.serverInfo && (
              <p className="text-xs text-muted-foreground">{testResult.serverInfo}</p>
            )}
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-between pt-4">
        <Button variant="outline" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <div className="flex items-center gap-2">
          {connectionTested && (
            <Badge variant="outline" className="text-green-600">
              <CheckCircle2 className="mr-1 h-3 w-3" />
              Connected
            </Badge>
          )}
          <Button
            variant="outline"
            onClick={handleTestConnection}
            disabled={!isFormValid || isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Testing...
              </>
            ) : (
              "Test Connection"
            )}
          </Button>
          <Button onClick={onNext} disabled={!connectionTested}>
            Continue
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}
