"use client"

import { useEffect, useState } from "react"
import { Settings, Shield, Users, Trash2, Plus, Info, AlertTriangle } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { useDagsterRbac } from "@/lib/hooks/use-dagster-rbac"
import type { DagsterRole, DagsterRoleInfo, DagsterRoleAssignmentWithUser } from "@/lib/api/types"

// UUID validation regex
const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

function DagsterRbacSettings() {
  const {
    loading,
    error,
    allRoleAssignments,
    availableRoles,
    totalAssignments,
    fetchAllRoleAssignments,
    fetchAvailableRoles,
    assignRole,
    removeRole,
  } = useDagsterRbac()

  const [assignDialogOpen, setAssignDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [roleToDelete, setRoleToDelete] = useState<DagsterRoleAssignmentWithUser | null>(null)
  const [newUserId, setNewUserId] = useState("")
  const [newRole, setNewRole] = useState<DagsterRole>("dagster-viewer")
  const [newResourcePattern, setNewResourcePattern] = useState("")
  const [assignError, setAssignError] = useState<string | null>(null)
  const [isAssigning, setIsAssigning] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [initialLoading, setInitialLoading] = useState(true)

  useEffect(() => {
    const loadData = async () => {
      try {
        await Promise.all([fetchAllRoleAssignments(), fetchAvailableRoles()])
      } finally {
        setInitialLoading(false)
      }
    }
    loadData()
  }, [fetchAllRoleAssignments, fetchAvailableRoles])

  const validateUserId = (id: string): string | null => {
    if (!id.trim()) {
      return "User ID is required"
    }
    if (!UUID_REGEX.test(id.trim())) {
      return "Invalid UUID format"
    }
    return null
  }

  const handleAssignRole = async () => {
    const validationError = validateUserId(newUserId)
    if (validationError) {
      setAssignError(validationError)
      return
    }

    setIsAssigning(true)
    setAssignError(null)

    try {
      await assignRole(
        newUserId.trim(),
        newRole,
        newResourcePattern.trim() || undefined
      )
      setAssignDialogOpen(false)
      setNewUserId("")
      setNewRole("dagster-viewer")
      setNewResourcePattern("")
      // Refresh the list after successful assignment
      await fetchAllRoleAssignments()
    } catch (err) {
      setAssignError(err instanceof Error ? err.message : "Failed to assign role")
    } finally {
      setIsAssigning(false)
    }
  }

  const handleDeleteClick = (assignment: DagsterRoleAssignmentWithUser) => {
    setRoleToDelete(assignment)
    setDeleteDialogOpen(true)
  }

  const handleConfirmDelete = async () => {
    if (!roleToDelete) return

    setIsDeleting(true)
    try {
      await removeRole(roleToDelete.id)
      setDeleteDialogOpen(false)
      setRoleToDelete(null)
    } catch (err) {
      console.error("Failed to remove role:", err)
    } finally {
      setIsDeleting(false)
    }
  }

  const getRoleBadgeVariant = (role: string) => {
    switch (role) {
      case "dagster-admin":
        return "destructive"
      case "dagster-operator":
        return "default"
      case "dagster-viewer":
        return "secondary"
      default:
        return "outline"
    }
  }

  const getRoleDescription = (role: string): string => {
    const roleInfo = availableRoles.find((r) => r.role === role)
    return roleInfo?.description || role
  }

  if (initialLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Role Info Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        {availableRoles.map((roleInfo: DagsterRoleInfo) => (
          <Card key={roleInfo.role}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Badge variant={getRoleBadgeVariant(roleInfo.role)}>
                  {roleInfo.role.replace("dagster-", "")}
                </Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground mb-2">
                {roleInfo.description}
              </p>
              <div className="text-xs">
                <strong>Permissions:</strong>
                <ul className="mt-1 space-y-0.5">
                  {roleInfo.permissions.slice(0, 3).map((perm) => (
                    <li key={perm} className="text-muted-foreground font-mono">
                      {perm}
                    </li>
                  ))}
                  {roleInfo.permissions.length > 3 && (
                    <li className="text-muted-foreground">
                      +{roleInfo.permissions.length - 3} more
                    </li>
                  )}
                </ul>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Role Assignments Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Dagster Role Assignments</CardTitle>
              <CardDescription>
                Manage user roles for Dagster access control ({totalAssignments} total)
              </CardDescription>
            </div>
            <Dialog open={assignDialogOpen} onOpenChange={setAssignDialogOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  Assign Role
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Assign Dagster Role</DialogTitle>
                  <DialogDescription>
                    Assign a Dagster role to a user. Roles control what actions users can perform in Dagster.
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  {assignError && (
                    <Alert variant="destructive">
                      <AlertDescription>{assignError}</AlertDescription>
                    </Alert>
                  )}
                  <div className="space-y-2">
                    <Label htmlFor="userId">User ID</Label>
                    <Input
                      id="userId"
                      placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                      value={newUserId}
                      onChange={(e) => setNewUserId(e.target.value)}
                    />
                    <p className="text-xs text-muted-foreground">
                      Enter the UUID of the user to assign the role to
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="role">Role</Label>
                    <Select value={newRole} onValueChange={(v) => setNewRole(v as DagsterRole)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {availableRoles.map((roleInfo) => (
                          <SelectItem key={roleInfo.role} value={roleInfo.role}>
                            <div className="flex items-center gap-2">
                              <Badge variant={getRoleBadgeVariant(roleInfo.role)}>
                                {roleInfo.role.replace("dagster-", "")}
                              </Badge>
                              <span className="text-muted-foreground text-xs truncate max-w-[150px]" title={roleInfo.description}>
                                {roleInfo.description}
                              </span>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="resourcePattern">Resource Pattern (optional)</Label>
                    <Input
                      id="resourcePattern"
                      placeholder="e.g., etl_* or warehouse/*"
                      value={newResourcePattern}
                      onChange={(e) => setNewResourcePattern(e.target.value)}
                    />
                    <p className="text-xs text-muted-foreground">
                      Restrict this role to specific resources using wildcards
                    </p>
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setAssignDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button onClick={handleAssignRole} disabled={isAssigning}>
                    {isAssigning ? "Assigning..." : "Assign Role"}
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        </CardHeader>
        <CardContent>
          {error && (
            <Alert variant="destructive" className="mb-4">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {allRoleAssignments.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Shield className="mx-auto h-12 w-12 mb-4" />
              <p>No role assignments yet</p>
              <p className="text-sm">Assign roles to users to control their Dagster access</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>User</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Resource Pattern</TableHead>
                  <TableHead>Assigned</TableHead>
                  <TableHead className="w-[70px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {allRoleAssignments.map((assignment) => (
                  <TableRow key={assignment.id}>
                    <TableCell>
                      <div>
                        <div className="font-medium">{assignment.user_email}</div>
                        {assignment.user_name && (
                          <div className="text-sm text-muted-foreground">
                            {assignment.user_name}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={getRoleBadgeVariant(assignment.role)}
                        title={getRoleDescription(assignment.role)}
                      >
                        {assignment.role.replace("dagster-", "")}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {assignment.resource_pattern ? (
                        <code className="text-xs bg-muted px-1 py-0.5 rounded">
                          {assignment.resource_pattern}
                        </code>
                      ) : (
                        <span className="text-muted-foreground">All resources</span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {new Date(assignment.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteClick(assignment)}
                        disabled={loading || isDeleting}
                        aria-label={`Remove ${assignment.role} role from ${assignment.user_email}`}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Remove Role Assignment
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove the{" "}
              <Badge variant={getRoleBadgeVariant(roleToDelete?.role || "")}>
                {roleToDelete?.role.replace("dagster-", "")}
              </Badge>{" "}
              role from <strong>{roleToDelete?.user_email}</strong>?
              <br /><br />
              This will revoke their Dagster permissions immediately.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? "Removing..." : "Remove Role"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Permission Format Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Info className="h-4 w-4" />
            Permission Format
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-2">
            Dagster permissions follow the format:{" "}
            <code className="bg-muted px-1 py-0.5 rounded">
              dagster:{"{type}"}:{"{name}"}:{"{action}"}
            </code>
          </p>
          <div className="grid gap-2 text-sm">
            <div>
              <strong>Types:</strong> jobs, assets, schedules, sensors, runs
            </div>
            <div>
              <strong>Actions:</strong> view, execute, materialize, control, terminate
            </div>
            <div>
              <strong>Wildcards:</strong> Use * for any match (e.g., dagster:jobs:etl_*:execute)
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      {/* Page header */}
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="text-muted-foreground">
          Configure system settings and access control
        </p>
      </div>

      {/* Settings Tabs */}
      <Tabs defaultValue="dagster-rbac" className="space-y-4">
        <TabsList>
          <TabsTrigger value="dagster-rbac" className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            Dagster RBAC
          </TabsTrigger>
          <TabsTrigger value="users" className="flex items-center gap-2" disabled>
            <Users className="h-4 w-4" />
            Users
          </TabsTrigger>
          <TabsTrigger value="general" className="flex items-center gap-2" disabled>
            <Settings className="h-4 w-4" />
            General
          </TabsTrigger>
        </TabsList>

        <TabsContent value="dagster-rbac">
          <DagsterRbacSettings />
        </TabsContent>

        <TabsContent value="users">
          <Card>
            <CardContent className="py-12 text-center">
              <Users className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">User management coming soon</h3>
              <p className="mt-2 text-muted-foreground">
                Manage users and their system roles.
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="general">
          <Card>
            <CardContent className="py-12 text-center">
              <Settings className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">General settings coming soon</h3>
              <p className="mt-2 text-muted-foreground">
                System configuration options will be available in a future release.
              </p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
