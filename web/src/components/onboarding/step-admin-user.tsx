"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import * as z from "zod"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { ArrowLeft, ArrowRight, Loader2, CheckCircle2, Copy, Key, AlertCircle } from "lucide-react"
import { useAdminExists, useRegisterAdmin } from "@/lib/hooks/use-onboarding"
import { ApiClientError } from "@/lib/api"
import { toast } from "sonner"

const formSchema = z.object({
  email: z.string().email("Please enter a valid email address"),
  password: z.string().min(8, "Password must be at least 8 characters"),
  confirmPassword: z.string(),
  name: z.string().optional(),
  generateApiKey: z.boolean(),
}).refine((data) => data.password === data.confirmPassword, {
  message: "Passwords do not match",
  path: ["confirmPassword"],
})

type FormValues = z.infer<typeof formSchema>

interface StepAdminUserProps {
  onNext: (data?: Record<string, unknown>) => void
  onBack: () => void
  onAdminCreated: (created: boolean, apiKey?: string) => void
}

export function StepAdminUser({ onNext, onBack, onAdminCreated }: StepAdminUserProps) {
  const { data: adminExistsData, isLoading: isCheckingAdmin } = useAdminExists()
  const registerMutation = useRegisterAdmin()
  const [apiKey, setApiKey] = useState<string | null>(null)
  const [registered, setRegistered] = useState(false)

  const adminExists = adminExistsData?.exists ?? false

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: "",
      password: "",
      confirmPassword: "",
      name: "",
      generateApiKey: true,
    },
  })

  const onSubmit = async (values: FormValues) => {
    try {
      const response = await registerMutation.mutateAsync({
        email: values.email,
        password: values.password,
        confirm_password: values.confirmPassword,
        name: values.name,
        generate_api_key: values.generateApiKey,
      })

      setRegistered(true)
      if (response.api_key) {
        setApiKey(response.api_key)
        onAdminCreated(true, response.api_key)
      } else {
        onAdminCreated(true)
      }

      toast.success("Admin account created", {
        description: "You can now proceed with the setup.",
      })
    } catch (error) {
      if (error instanceof ApiClientError) {
        toast.error("Registration failed", {
          description: error.details?.detail || error.message,
        })
      }
    }
  }

  const copyApiKey = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey)
      toast.success("API key copied", {
        description: "The API key has been copied to your clipboard.",
      })
    }
  }

  const handleContinue = () => {
    onNext({ admin_created: true, api_key_generated: !!apiKey })
  }

  // If admin already exists, show different UI
  if (adminExists && !isCheckingAdmin) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Admin User</h2>
          <p className="text-muted-foreground mt-2">
            Configure the first administrator account for Philotes.
          </p>
        </div>

        <Alert>
          <CheckCircle2 className="h-4 w-4" />
          <AlertTitle>Admin account already exists</AlertTitle>
          <AlertDescription>
            An administrator account has already been created. You can proceed with the setup.
          </AlertDescription>
        </Alert>

        <div className="flex justify-between pt-4">
          <Button variant="outline" onClick={onBack}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <Button onClick={() => onNext({ admin_existed: true })}>
            Continue
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    )
  }

  // If registration complete, show API key
  if (registered) {
    return (
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">Admin User Created</h2>
          <p className="text-muted-foreground mt-2">
            Your administrator account has been created successfully.
          </p>
        </div>

        <Alert className="bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-800">
          <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
          <AlertTitle>Account created successfully</AlertTitle>
          <AlertDescription>
            You can now log in with your email and password.
          </AlertDescription>
        </Alert>

        {apiKey && (
          <Alert>
            <Key className="h-4 w-4" />
            <AlertTitle>Your API Key</AlertTitle>
            <AlertDescription className="mt-2">
              <p className="mb-2 text-sm">
                Save this API key securely. It will only be shown once.
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 p-2 bg-muted rounded text-xs font-mono break-all">
                  {apiKey}
                </code>
                <Button variant="outline" size="icon" onClick={copyApiKey}>
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </AlertDescription>
          </Alert>
        )}

        <div className="flex justify-between pt-4">
          <Button variant="outline" onClick={onBack}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <Button onClick={handleContinue}>
            Continue
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">Create Admin User</h2>
        <p className="text-muted-foreground mt-2">
          Set up the first administrator account for Philotes. This account will have full access
          to all features.
        </p>
      </div>

      {isCheckingAdmin && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      )}

      {!isCheckingAdmin && (
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input placeholder="admin@example.com" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name (optional)</FormLabel>
                  <FormControl>
                    <Input placeholder="Admin User" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Password</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="Minimum 8 characters" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="confirmPassword"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Confirm Password</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="Confirm your password" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="generateApiKey"
              render={({ field }) => (
                <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                  <div className="space-y-1 leading-none">
                    <FormLabel>Generate API Key</FormLabel>
                    <FormDescription>
                      Create an API key for programmatic access. The key will only be shown once.
                    </FormDescription>
                  </div>
                </FormItem>
              )}
            />

            {registerMutation.error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  {registerMutation.error instanceof ApiClientError
                    ? registerMutation.error.details?.detail || registerMutation.error.message
                    : "Failed to create account"}
                </AlertDescription>
              </Alert>
            )}

            <div className="flex justify-between pt-4">
              <Button type="button" variant="outline" onClick={onBack}>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back
              </Button>
              <Button type="submit" disabled={registerMutation.isPending}>
                {registerMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    Create Account
                    <ArrowRight className="ml-2 h-4 w-4" />
                  </>
                )}
              </Button>
            </div>
          </form>
        </Form>
      )}
    </div>
  )
}
