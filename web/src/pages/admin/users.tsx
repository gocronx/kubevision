import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Plus, Pencil, Trash2, KeyRound, Users } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuth } from "@/stores/auth-store"
import { canManageUsers } from "@/lib/permissions"
import {
  useUsers,
  useCreateUser,
  useUpdateUser,
  useDeleteUser,
  useResetPassword,
  type User,
  type CreateUserPayload,
  type UpdateUserPayload,
} from "@/hooks/use-users"

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const ROLES = ["admin", "editor", "viewer", "custom"] as const

function roleBadgeVariant(role: string): "default" | "secondary" | "outline" | "destructive" {
  switch (role) {
    case "super-admin":
      return "destructive"
    case "admin":
      return "default"
    case "editor":
      return "secondary"
    default:
      return "outline"
  }
}

// ---------------------------------------------------------------------------
// Create User Dialog
// ---------------------------------------------------------------------------

interface CreateUserDialogProps {
  open: boolean
  onClose: () => void
}

const DEFAULT_CREATE_FORM: CreateUserPayload = {
  username: "",
  password: "",
  role: "viewer",
}

function CreateUserDialog({ open, onClose }: CreateUserDialogProps) {
  const { t } = useTranslation()
  const createMutation = useCreateUser()
  const [form, setForm] = useState<CreateUserPayload>(DEFAULT_CREATE_FORM)

  function handleOpenChange(o: boolean) {
    if (!o) {
      setForm(DEFAULT_CREATE_FORM)
      onClose()
    }
  }

  async function handleSubmit() {
    if (!form.username.trim() || !form.password.trim()) return
    try {
      await createMutation.mutateAsync(form)
      toast.success(t("users.createdToast"))
      handleOpenChange(false)
    } catch {
      // toasted by api interceptor
    }
  }

  const isValid = form.username.trim().length > 0 && form.password.trim().length > 0

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("users.create")}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cu-username">{t("common.username")}</Label>
            <Input
              id="cu-username"
              value={form.username}
              onChange={(e) => setForm((p) => ({ ...p, username: e.target.value }))}
              placeholder="alice"
              autoFocus
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cu-password">{t("common.password")}</Label>
            <Input
              id="cu-password"
              type="password"
              value={form.password}
              onChange={(e) => setForm((p) => ({ ...p, password: e.target.value }))}
              placeholder="••••••••"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cu-role">{t("users.role")}</Label>
            <select
              id="cu-role"
              value={form.role}
              onChange={(e) => setForm((p) => ({ ...p, role: e.target.value }))}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-ring"
            >
              {ROLES.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!isValid || createMutation.isPending}
          >
            {t("common.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Edit User Dialog
// ---------------------------------------------------------------------------

interface EditUserDialogProps {
  open: boolean
  onClose: () => void
  user: User | null
}

function EditUserDialog({ open, onClose, user }: EditUserDialogProps) {
  const { t } = useTranslation()
  const updateMutation = useUpdateUser()
  const [form, setForm] = useState<UpdateUserPayload>({
    role: user?.role ?? "viewer",
    isActive: user?.isActive ?? true,
  })

  // Sync form when user changes.
  function handleOpenChange(o: boolean) {
    if (o && user) {
      setForm({ role: user.role, isActive: user.isActive })
    }
    if (!o) {
      onClose()
    }
  }

  async function handleSubmit() {
    if (!user) return
    try {
      await updateMutation.mutateAsync({ id: user.id, payload: form })
      toast.success(t("users.updatedToast"))
      onClose()
    } catch {
      // toasted by api interceptor
    }
  }

  // Prevent editing super-admin role.
  const isSuperAdmin = user?.role === "super-admin"

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("users.edit")} — {user?.username}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="eu-role">{t("users.role")}</Label>
            <select
              id="eu-role"
              value={form.role}
              onChange={(e) => setForm((p) => ({ ...p, role: e.target.value }))}
              disabled={isSuperAdmin}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            >
              {ROLES.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
            {isSuperAdmin && (
              <p className="text-xs text-muted-foreground">{t("users.superAdminNote")}</p>
            )}
          </div>

          <div className="flex items-center gap-2">
            <input
              id="eu-active"
              type="checkbox"
              checked={form.isActive}
              onChange={(e) => setForm((p) => ({ ...p, isActive: e.target.checked }))}
              className="size-4 accent-primary"
            />
            <Label htmlFor="eu-active">{t("users.active")}</Label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={updateMutation.isPending}>
            {t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Reset Password Dialog
// ---------------------------------------------------------------------------

interface ResetPasswordDialogProps {
  open: boolean
  onClose: () => void
  user: User | null
}

function ResetPasswordDialog({ open, onClose, user }: ResetPasswordDialogProps) {
  const { t } = useTranslation()
  const resetMutation = useResetPassword()
  const [newPassword, setNewPassword] = useState("")

  function handleOpenChange(o: boolean) {
    if (!o) {
      setNewPassword("")
      onClose()
    }
  }

  async function handleSubmit() {
    if (!user || !newPassword.trim()) return
    try {
      await resetMutation.mutateAsync({ id: user.id, payload: { newPassword } })
      toast.success(t("users.passwordResetToast"))
      handleOpenChange(false)
    } catch {
      // toasted by api interceptor
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("users.resetPassword")} — {user?.username}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="rp-password">{t("users.newPassword")}</Label>
          <Input
            id="rp-password"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder="••••••••"
            autoFocus
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!newPassword.trim() || resetMutation.isPending}
          >
            {t("users.resetPassword")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Delete Confirmation Dialog
// ---------------------------------------------------------------------------

interface DeleteUserDialogProps {
  open: boolean
  onClose: () => void
  user: User | null
}

function DeleteUserDialog({ open, onClose, user }: DeleteUserDialogProps) {
  const { t } = useTranslation()
  const deleteMutation = useDeleteUser()

  async function handleDelete() {
    if (!user) return
    try {
      await deleteMutation.mutateAsync(user.id)
      toast.success(t("users.deletedToast"))
      onClose()
    } catch {
      // toasted by api interceptor
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("users.deleteTitle")}</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">
          {t("users.deleteConfirm", { username: user?.username ?? "" })}
        </p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
          >
            {t("common.delete")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ---------------------------------------------------------------------------
// Users Page
// ---------------------------------------------------------------------------

export function UsersPage() {
  const { t } = useTranslation()
  const { user: currentUser } = useAuth()
  const { data: users = [], isLoading } = useUsers()
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<User | null>(null)
  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)

  // Hard gate: only super-admin can manage users.
  if (!canManageUsers(currentUser?.role ?? "")) {
    return (
      <div className="flex items-center justify-center h-full py-16 text-muted-foreground">
        {t("common.forbidden", "Access denied")}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("users.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("users.description")}</p>
        </div>
        <Button size="sm" onClick={() => setCreateOpen(true)} className="self-start gap-1.5">
          <Plus className="size-4" />
          {t("users.create")}
        </Button>
      </div>

      <div className="max-w-full overflow-x-auto rounded-md border">
        <table className="w-full min-w-[44rem] text-sm">
          <thead className="bg-muted/50">
            <tr className="border-b">
              <th className="px-3 py-2 text-left font-medium">ID</th>
              <th className="px-3 py-2 text-left font-medium">{t("common.username")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("users.role")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("common.status")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("users.createdAt")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <tr key={i} className="border-b">
                  {Array.from({ length: 6 }).map((_, j) => (
                    <td key={j} className="px-3 py-2">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : users.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-3 py-12 text-center text-muted-foreground">
                  <div className="flex flex-col items-center gap-2">
                    <Users className="size-8 opacity-30" />
                    <span>{t("users.empty")}</span>
                  </div>
                </td>
              </tr>
            ) : (
              users.map((user) => (
                <tr key={user.id} className="border-b hover:bg-muted/30 transition-colors">
                  <td className="px-3 py-2 text-muted-foreground">{user.id}</td>
                  <td className="px-3 py-2 font-medium">{user.username}</td>
                  <td className="px-3 py-2">
                    <Badge variant={roleBadgeVariant(user.role)} className="text-xs">
                      {user.role}
                    </Badge>
                  </td>
                  <td className="px-3 py-2">
                    <Badge variant={user.isActive ? "default" : "secondary"} className="text-xs">
                      {user.isActive ? t("users.active") : t("users.inactive")}
                    </Badge>
                  </td>
                  <td className="px-3 py-2 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(user.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-1">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 gap-1 text-xs"
                        onClick={() => setEditTarget(user)}
                      >
                        <Pencil className="size-3" />
                        {t("common.edit")}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 gap-1 text-xs"
                        onClick={() => setResetTarget(user)}
                      >
                        <KeyRound className="size-3" />
                        {t("users.resetPassword")}
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 gap-1 text-xs text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget(user)}
                      >
                        <Trash2 className="size-3" />
                        {t("common.delete")}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <CreateUserDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
      />

      <EditUserDialog
        open={editTarget !== null}
        onClose={() => setEditTarget(null)}
        user={editTarget}
      />

      <ResetPasswordDialog
        open={resetTarget !== null}
        onClose={() => setResetTarget(null)}
        user={resetTarget}
      />

      <DeleteUserDialog
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        user={deleteTarget}
      />
    </div>
  )
}
