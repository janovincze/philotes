import { apiClient } from "./client"
import type {
  ScalingPolicy,
  ScalingPolicyResponse,
  ScalingPoliciesResponse,
  ScalingState,
  ScalingStateResponse,
  ScalingHistory,
  ScalingHistoryResponse,
  ScalingEvaluationResult,
  ScalingEvaluationResponse,
  CreateScalingPolicyInput,
} from "./types"

export const scalingApi = {
  /**
   * List all scaling policies
   */
  async listPolicies(): Promise<ScalingPolicy[]> {
    const response = await apiClient.get<ScalingPoliciesResponse>(
      "/api/v1/scaling/policies"
    )
    return response.policies
  },

  /**
   * Get a single scaling policy by ID
   */
  async getPolicy(id: string): Promise<ScalingPolicy> {
    const response = await apiClient.get<ScalingPolicyResponse>(
      `/api/v1/scaling/policies/${id}`
    )
    return response.policy
  },

  /**
   * Create a new scaling policy
   */
  async createPolicy(input: CreateScalingPolicyInput): Promise<ScalingPolicy> {
    const response = await apiClient.post<ScalingPolicyResponse>(
      "/api/v1/scaling/policies",
      input
    )
    return response.policy
  },

  /**
   * Update an existing scaling policy
   */
  async updatePolicy(
    id: string,
    input: Partial<CreateScalingPolicyInput>
  ): Promise<ScalingPolicy> {
    const response = await apiClient.put<ScalingPolicyResponse>(
      `/api/v1/scaling/policies/${id}`,
      input
    )
    return response.policy
  },

  /**
   * Delete a scaling policy
   */
  async deletePolicy(id: string): Promise<void> {
    await apiClient.delete(`/api/v1/scaling/policies/${id}`)
  },

  /**
   * Enable a scaling policy
   */
  async enablePolicy(id: string): Promise<ScalingPolicy> {
    const response = await apiClient.post<ScalingPolicyResponse>(
      `/api/v1/scaling/policies/${id}/enable`
    )
    return response.policy
  },

  /**
   * Disable a scaling policy
   */
  async disablePolicy(id: string): Promise<ScalingPolicy> {
    const response = await apiClient.post<ScalingPolicyResponse>(
      `/api/v1/scaling/policies/${id}/disable`
    )
    return response.policy
  },

  /**
   * Evaluate a scaling policy (dry-run or actual)
   */
  async evaluatePolicy(
    id: string,
    dryRun: boolean = true
  ): Promise<ScalingEvaluationResult> {
    const response = await apiClient.post<ScalingEvaluationResponse>(
      `/api/v1/scaling/policies/${id}/evaluate`,
      { dry_run: dryRun }
    )
    return response.result
  },

  /**
   * Get the current scaling state for a policy
   */
  async getPolicyState(id: string): Promise<ScalingState> {
    const response = await apiClient.get<ScalingStateResponse>(
      `/api/v1/scaling/policies/${id}/state`
    )
    return response.state
  },

  /**
   * List scaling history, optionally filtered by policy ID
   */
  async listHistory(policyId?: string): Promise<ScalingHistory[]> {
    const path = policyId
      ? `/api/v1/scaling/policies/${policyId}/history`
      : "/api/v1/scaling/history"
    const response = await apiClient.get<ScalingHistoryResponse>(path)
    return response.history
  },
}
