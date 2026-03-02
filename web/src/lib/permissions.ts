/**
 * Centralised frontend permission helpers.
 *
 * These functions mirror the backend role definitions in
 * internal/repository/db.go and are the single source of truth for
 * deciding which UI elements each role can see or interact with.
 *
 * Role hierarchy (highest → lowest):
 *   super-admin > admin > editor > viewer > custom
 *
 * Keep this file in sync with the backend initSystemRoles seed whenever
 * role permissions are updated.
 */

export type UserRole = "super-admin" | "admin" | "editor" | "viewer" | "custom" | string

/**
 * Returns true when the role is allowed to manage users (create, edit, delete,
 * reset passwords). Only super-admin has this capability.
 */
export function canManageUsers(role: UserRole): boolean {
  return role === "super-admin"
}

/**
 * Returns true when the role is allowed to access the admin panel
 * (audit logs, API keys, webhooks, terminal session recordings).
 * Both super-admin and admin can access these pages.
 */
export function canAccessAdmin(role: UserRole): boolean {
  return role === "super-admin" || role === "admin"
}

/**
 * Returns true when the role is allowed to create, update, or delete
 * Kubernetes resources (resources:create / resources:update / resources:delete).
 * Viewer and custom roles are read-only and must not see mutating buttons.
 */
export function canMutateResources(role: UserRole): boolean {
  return role === "super-admin" || role === "admin" || role === "editor"
}

/**
 * Returns true when the role is allowed to exec into pods or stream logs
 * via the terminal.
 */
export function canExec(role: UserRole): boolean {
  return role === "super-admin" || role === "admin" || role === "editor"
}

/**
 * Returns true when the role has read-only access only (no create/update/delete
 * on K8s resources). Convenience inverse of canMutateResources.
 */
export function isReadOnly(role: UserRole): boolean {
  return !canMutateResources(role)
}
