import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface User {
  id: number
  username: string
  email: string
  role: string
  isActive: boolean
  createdAt: string
}

export interface CreateUserPayload {
  username: string
  password: string
  role: string
}

export interface UpdateUserPayload {
  role: string
  isActive: boolean
}

export interface ResetPasswordPayload {
  newPassword: string
}

export interface ChangePasswordPayload {
  oldPassword: string
  newPassword: string
}

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

const USERS_QUERY_KEY = ["users"] as const

// ---------------------------------------------------------------------------
// List users
// ---------------------------------------------------------------------------

export function useUsers() {
  return useQuery<User[]>({
    queryKey: USERS_QUERY_KEY,
    queryFn: async () => {
      const res = await api.get("/users")
      return (res as unknown as User[]) ?? []
    },
  })
}

// ---------------------------------------------------------------------------
// Get single user
// ---------------------------------------------------------------------------

export function useUser(id: number) {
  return useQuery<User>({
    queryKey: [...USERS_QUERY_KEY, id],
    queryFn: async () => {
      const res = await api.get(`/users/${id}`)
      return res as unknown as User
    },
    enabled: id > 0,
  })
}

// ---------------------------------------------------------------------------
// Create user
// ---------------------------------------------------------------------------

export function useCreateUser() {
  const queryClient = useQueryClient()

  return useMutation<User, Error, CreateUserPayload>({
    mutationFn: async (payload) => {
      const res = await api.post("/users", payload)
      return res as unknown as User
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USERS_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Update user
// ---------------------------------------------------------------------------

export function useUpdateUser() {
  const queryClient = useQueryClient()

  return useMutation<User, Error, { id: number; payload: UpdateUserPayload }>({
    mutationFn: async ({ id, payload }) => {
      const res = await api.put(`/users/${id}`, payload)
      return res as unknown as User
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USERS_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Delete user
// ---------------------------------------------------------------------------

export function useDeleteUser() {
  const queryClient = useQueryClient()

  return useMutation<void, Error, number>({
    mutationFn: async (id) => {
      await api.delete(`/users/${id}`)
    },
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: USERS_QUERY_KEY })
      const previous = queryClient.getQueryData<User[]>(USERS_QUERY_KEY)
      queryClient.setQueryData<User[]>(USERS_QUERY_KEY, (old) =>
        old ? old.filter((u) => u.id !== id) : []
      )
      return { previous }
    },
    onError: (_err, _id, context) => {
      const ctx = context as { previous?: User[] } | undefined
      if (ctx?.previous) {
        queryClient.setQueryData(USERS_QUERY_KEY, ctx.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: USERS_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Admin reset password
// ---------------------------------------------------------------------------

export function useResetPassword() {
  return useMutation<void, Error, { id: number; payload: ResetPasswordPayload }>({
    mutationFn: async ({ id, payload }) => {
      await api.put(`/users/${id}/reset-password`, payload)
    },
  })
}

// ---------------------------------------------------------------------------
// Change own password
// ---------------------------------------------------------------------------

export function useChangePassword() {
  return useMutation<void, Error, ChangePasswordPayload>({
    mutationFn: async (payload) => {
      await api.put("/users/me/password", payload)
    },
  })
}
