import { apiClient } from "./client"
import type {
  DagsterRoleAssignment,
  DagsterRoleAssignmentWithUser,
  DagsterRoleAssignmentsResponse,
  DagsterUserPermissions,
  DagsterPermission,
  DagsterRoleInfo,
  AvailableDagsterRolesResponse,
  AssignDagsterRoleRequest,
  AddDagsterPermissionRequest,
  DagsterRole,
} from "./types"

export const dagsterRbacApi = {
  /**
   * Get all Dagster permissions for a user
   */
  async getUserPermissions(userId: string): Promise<DagsterUserPermissions> {
    return apiClient.get<DagsterUserPermissions>(
      `/api/v1/users/${userId}/dagster-permissions`
    )
  },

  /**
   * Assign a Dagster role to a user
   */
  async assignRole(
    userId: string,
    role: DagsterRole,
    resourcePattern?: string
  ): Promise<DagsterRoleAssignment> {
    const request: AssignDagsterRoleRequest = { role, resource_pattern: resourcePattern }
    return apiClient.post<DagsterRoleAssignment>(
      `/api/v1/users/${userId}/dagster-roles`,
      request
    )
  },

  /**
   * Remove a Dagster role assignment
   */
  async removeRole(roleId: string): Promise<void> {
    await apiClient.delete(`/api/v1/dagster-roles/${roleId}`)
  },

  /**
   * Add a custom Dagster permission to a user
   */
  async addPermission(
    userId: string,
    permission: string
  ): Promise<DagsterPermission> {
    const request: AddDagsterPermissionRequest = { permission }
    return apiClient.post<DagsterPermission>(
      `/api/v1/users/${userId}/dagster-permissions`,
      request
    )
  },

  /**
   * Remove a custom Dagster permission
   */
  async removePermission(permissionId: string): Promise<void> {
    await apiClient.delete(`/api/v1/dagster-permissions/${permissionId}`)
  },

  /**
   * List all Dagster role assignments (admin only)
   */
  async listAllRoleAssignments(): Promise<{
    assignments: DagsterRoleAssignmentWithUser[]
    total: number
  }> {
    const response = await apiClient.get<DagsterRoleAssignmentsResponse>(
      "/api/v1/dagster-roles"
    )
    return {
      assignments: response.role_assignments,
      total: response.total_count,
    }
  },

  /**
   * Get available Dagster roles information
   */
  async getAvailableRoles(): Promise<DagsterRoleInfo[]> {
    const response = await apiClient.get<AvailableDagsterRolesResponse>(
      "/api/v1/dagster-roles/available"
    )
    return response.roles
  },
}
