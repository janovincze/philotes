import { useState, useCallback } from "react"
import { dagsterRbacApi } from "@/lib/api/dagster-rbac"
import type {
  DagsterUserPermissions,
  DagsterRoleAssignmentWithUser,
  DagsterRoleInfo,
  DagsterRole,
} from "@/lib/api/types"

export function useDagsterRbac() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [userPermissions, setUserPermissions] =
    useState<DagsterUserPermissions | null>(null)
  const [allRoleAssignments, setAllRoleAssignments] = useState<
    DagsterRoleAssignmentWithUser[]
  >([])
  const [availableRoles, setAvailableRoles] = useState<DagsterRoleInfo[]>([])
  const [totalAssignments, setTotalAssignments] = useState(0)

  const fetchUserPermissions = useCallback(async (userId: string) => {
    setLoading(true)
    setError(null)
    try {
      const permissions = await dagsterRbacApi.getUserPermissions(userId)
      setUserPermissions(permissions)
      return permissions
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to fetch permissions"
      setError(message)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchAllRoleAssignments = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const { assignments, total } =
        await dagsterRbacApi.listAllRoleAssignments()
      setAllRoleAssignments(assignments)
      setTotalAssignments(total)
      return { assignments, total }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to fetch role assignments"
      setError(message)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchAvailableRoles = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const roles = await dagsterRbacApi.getAvailableRoles()
      setAvailableRoles(roles)
      return roles
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to fetch available roles"
      setError(message)
      throw err
    } finally {
      setLoading(false)
    }
  }, [])

  const assignRole = useCallback(
    async (userId: string, role: DagsterRole, resourcePattern?: string) => {
      setLoading(true)
      setError(null)
      try {
        const assignment = await dagsterRbacApi.assignRole(
          userId,
          role,
          resourcePattern
        )
        // Refresh the permissions after assignment
        await fetchUserPermissions(userId)
        return assignment
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to assign role"
        setError(message)
        throw err
      } finally {
        setLoading(false)
      }
    },
    [fetchUserPermissions]
  )

  const removeRole = useCallback(
    async (roleId: string, userId?: string) => {
      setLoading(true)
      setError(null)
      try {
        await dagsterRbacApi.removeRole(roleId)
        // Refresh permissions if userId is provided
        if (userId) {
          await fetchUserPermissions(userId)
        }
        // Refresh all assignments list
        await fetchAllRoleAssignments()
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to remove role"
        setError(message)
        throw err
      } finally {
        setLoading(false)
      }
    },
    [fetchUserPermissions, fetchAllRoleAssignments]
  )

  const addPermission = useCallback(
    async (userId: string, permission: string) => {
      setLoading(true)
      setError(null)
      try {
        const perm = await dagsterRbacApi.addPermission(userId, permission)
        // Refresh the permissions after adding
        await fetchUserPermissions(userId)
        return perm
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to add permission"
        setError(message)
        throw err
      } finally {
        setLoading(false)
      }
    },
    [fetchUserPermissions]
  )

  const removePermission = useCallback(
    async (permissionId: string, userId?: string) => {
      setLoading(true)
      setError(null)
      try {
        await dagsterRbacApi.removePermission(permissionId)
        // Refresh permissions if userId is provided
        if (userId) {
          await fetchUserPermissions(userId)
        }
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to remove permission"
        setError(message)
        throw err
      } finally {
        setLoading(false)
      }
    },
    [fetchUserPermissions]
  )

  return {
    loading,
    error,
    userPermissions,
    allRoleAssignments,
    availableRoles,
    totalAssignments,
    fetchUserPermissions,
    fetchAllRoleAssignments,
    fetchAvailableRoles,
    assignRole,
    removeRole,
    addPermission,
    removePermission,
  }
}
