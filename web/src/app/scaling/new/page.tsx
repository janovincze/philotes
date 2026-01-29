"use client"

import { useRouter } from "next/navigation"
import Link from "next/link"
import { ArrowLeft } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useCreateScalingPolicy } from "@/lib/hooks/use-scaling"
import { ScalingPolicyForm } from "@/components/scaling/scaling-policy-form"
import type { CreateScalingPolicyInput } from "@/lib/api/types"

export default function NewScalingPolicyPage() {
  const router = useRouter()
  const createPolicy = useCreateScalingPolicy()

  const handleSubmit = (data: CreateScalingPolicyInput) => {
    createPolicy.mutate(data, {
      onSuccess: (policy) => {
        router.push(`/scaling/${policy.id}`)
      },
    })
  }

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/scaling">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-3xl font-bold">New Scaling Policy</h1>
          <p className="text-muted-foreground">
            Create a new auto-scaling policy for your infrastructure
          </p>
        </div>
      </div>

      {/* Form */}
      <ScalingPolicyForm onSubmit={handleSubmit} isSubmitting={createPolicy.isPending} />

      {/* Error display */}
      {createPolicy.error && (
        <div className="rounded-md bg-destructive/10 p-4 text-destructive">
          Failed to create policy: {createPolicy.error.message}
        </div>
      )}
    </div>
  )
}
