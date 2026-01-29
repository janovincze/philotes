"use client"

import { useState } from "react"
import Link from "next/link"
import { Scale, Plus, XCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  useScalingPolicies,
  useEnableScalingPolicy,
  useDisableScalingPolicy,
  useDeleteScalingPolicy,
} from "@/lib/hooks/use-scaling"
import { ScalingPolicyCard, ScalingPolicyCardSkeleton } from "@/components/scaling/scaling-policy-card"

function PoliciesListSkeleton() {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {[1, 2, 3, 4].map((i) => (
        <ScalingPolicyCardSkeleton key={i} />
      ))}
    </div>
  )
}

export default function ScalingPage() {
  const [mutatingId, setMutatingId] = useState<string | null>(null)
  const { data: policies, isLoading, error } = useScalingPolicies()
  const enablePolicy = useEnableScalingPolicy()
  const disablePolicy = useDisableScalingPolicy()
  const deletePolicy = useDeleteScalingPolicy()

  const handleEnable = (id: string) => {
    setMutatingId(id)
    enablePolicy.mutate(id, {
      onSettled: () => setMutatingId(null),
    })
  }

  const handleDisable = (id: string) => {
    setMutatingId(id)
    disablePolicy.mutate(id, {
      onSettled: () => setMutatingId(null),
    })
  }

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this policy?")) {
      setMutatingId(id)
      deletePolicy.mutate(id, {
        onSettled: () => setMutatingId(null),
      })
    }
  }

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Scaling Policies</h1>
          <p className="text-muted-foreground">
            Configure auto-scaling rules for your infrastructure
          </p>
        </div>
        <Button asChild>
          <Link href="/scaling/new">
            <Plus className="mr-2 h-4 w-4" />
            New Policy
          </Link>
        </Button>
      </div>

      {/* Policies list */}
      {isLoading ? (
        <PoliciesListSkeleton />
      ) : error ? (
        <Card>
          <CardContent className="py-8 text-center">
            <XCircle className="mx-auto h-8 w-8 text-destructive" />
            <p className="mt-2 text-muted-foreground">
              Failed to load scaling policies. Please try again.
            </p>
          </CardContent>
        </Card>
      ) : policies && policies.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2">
          {policies.map((policy) => (
            <ScalingPolicyCard
              key={policy.id}
              policy={policy}
              onEnable={handleEnable}
              onDisable={handleDisable}
              onDelete={handleDelete}
              mutatingId={mutatingId}
            />
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center">
            <Scale className="mx-auto h-12 w-12 text-muted-foreground" />
            <h3 className="mt-4 text-lg font-medium">No scaling policies</h3>
            <p className="mt-2 text-muted-foreground">
              Create your first scaling policy to automatically manage resource capacity.
            </p>
            <Button className="mt-4" asChild>
              <Link href="/scaling/new">
                <Plus className="mr-2 h-4 w-4" />
                New Policy
              </Link>
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
