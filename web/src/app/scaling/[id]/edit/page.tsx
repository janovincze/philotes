"use client"

import { use } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { ArrowLeft, XCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useScalingPolicy, useUpdateScalingPolicy } from "@/lib/hooks/use-scaling"
import { ScalingPolicyForm } from "@/components/scaling/scaling-policy-form"
import type { CreateScalingPolicyInput } from "@/lib/api/types"

function EditFormSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-64" />
      <Skeleton className="h-48" />
      <Skeleton className="h-48" />
    </div>
  )
}

export default function EditScalingPolicyPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const router = useRouter()

  const { data: policy, isLoading, error } = useScalingPolicy(id)
  const updatePolicy = useUpdateScalingPolicy()

  const handleSubmit = (data: CreateScalingPolicyInput) => {
    updatePolicy.mutate(
      { id, input: data },
      {
        onSuccess: () => {
          router.push(`/scaling/${id}`)
        },
      }
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href={`/scaling/${id}`}>
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <div className="space-y-2">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-32" />
          </div>
        </div>
        <EditFormSkeleton />
      </div>
    )
  }

  if (error || !policy) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/scaling">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
        </div>
        <Card>
          <CardContent className="py-8 text-center">
            <XCircle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-muted-foreground">
              Failed to load scaling policy. Please try again.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link href={`/scaling/${id}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-3xl font-bold">Edit Policy</h1>
          <p className="text-muted-foreground">{policy.name}</p>
        </div>
      </div>

      {/* Form */}
      <ScalingPolicyForm
        policy={policy}
        onSubmit={handleSubmit}
        isSubmitting={updatePolicy.isPending}
      />

      {/* Error display */}
      {updatePolicy.error && (
        <div className="rounded-md bg-destructive/10 p-4 text-destructive">
          Failed to update policy: {updatePolicy.error.message}
        </div>
      )}
    </div>
  )
}
