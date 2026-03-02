import axios from "axios"
import { toast } from "sonner"

interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
  meta?: ApiMeta
}

export interface ApiMeta {
  total?: number
  stale?: boolean
  source?: string
  requestId?: string
}

const api = axios.create({
  baseURL: "/api/v1",
  timeout: 15000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token")
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => {
    const body = res.data as ApiResponse
    if (body.code === 0) {
      // When __preserveMeta flag is set, return data+meta together
      if ((res.config as Record<string, unknown>).__preserveMeta) {
        return { data: body.data, meta: body.meta } as never
      }
      return body.data as never
    }
    if (body.code === 40101) {
      // TODO: refresh token
      localStorage.removeItem("token")
      localStorage.removeItem("user")
      window.location.href = "/login"
      return Promise.reject(new Error("Token expired"))
    }
    // 40102 = 2FA required — return the full body so callers can handle it.
    if (body.code === 40102) {
      return Promise.reject({ is2FARequired: true, data: body.data })
    }
    // 40300 = permission denied.
    if (body.code === 40300) {
      toast.error("Permission denied")
      return Promise.reject(new Error(body.message || "Permission denied"))
    }
    if (body.code >= 40000) {
      toast.error(body.message || "Request failed")
      return Promise.reject(new Error(body.message))
    }
    return body.data as never
  },
  (error: unknown) => {
    if (axios.isAxiosError(error)) {
      const message = (error.response?.data as ApiResponse | undefined)?.message
        ?? error.message
      toast.error(message)
    } else if (error instanceof Error) {
      toast.error(error.message)
    }
    return Promise.reject(error)
  }
)

/**
 * GET request that preserves the `meta` field from paginated API responses.
 * Use this for endpoints that return meta (total, stale, etc.).
 */
export async function getWithMeta<T = unknown>(
  url: string,
  config?: Parameters<typeof api.get>[1]
): Promise<{ data: T; meta?: ApiMeta }> {
  const res = await api.get(url, {
    ...config,
    __preserveMeta: true,
  } as Parameters<typeof api.get>[1])
  return res as unknown as { data: T; meta?: ApiMeta }
}

export default api
