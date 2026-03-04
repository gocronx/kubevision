import axios, { type AxiosRequestConfig } from "axios"
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

// ---------------------------------------------------------------------------
// Token refresh state — ensures only one refresh is in-flight at a time.
// Concurrent 40101 responses queue up and resolve once the single refresh
// completes, then all queued requests are retried with the new token.
// ---------------------------------------------------------------------------

let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

function subscribeTokenRefresh(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach((cb) => cb(newToken))
  refreshSubscribers = []
}

function onRefreshFailed() {
  refreshSubscribers = []
}

async function tryRefreshToken(): Promise<string | null> {
  const refreshToken = localStorage.getItem("refreshToken")
  if (!refreshToken) return null

  try {
    // Use a plain axios call to avoid triggering our own interceptors.
    const res = await axios.post("/api/v1/auth/refresh", { refreshToken })
    const body = res.data as ApiResponse<{
      accessToken: string
      refreshToken: string
      user: unknown
    }>
    if (body.code === 0 && body.data) {
      localStorage.setItem("token", body.data.accessToken)
      localStorage.setItem("refreshToken", body.data.refreshToken)
      if (body.data.user) {
        localStorage.setItem("user", JSON.stringify(body.data.user))
      }
      return body.data.accessToken
    }
  } catch {
    // Refresh failed — token revoked, expired, or network error.
  }
  return null
}

function forceLogout() {
  localStorage.removeItem("token")
  localStorage.removeItem("refreshToken")
  localStorage.removeItem("user")
  window.location.href = "/login"
}

// ---------------------------------------------------------------------------

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
      const originalRequest = res.config as AxiosRequestConfig & { _retry?: boolean }

      // Prevent infinite retry loops.
      if (originalRequest._retry) {
        forceLogout()
        return Promise.reject(new Error("Token expired"))
      }
      originalRequest._retry = true

      if (!isRefreshing) {
        isRefreshing = true
        tryRefreshToken().then((newToken) => {
          isRefreshing = false
          if (newToken) {
            onTokenRefreshed(newToken)
          } else {
            onRefreshFailed()
            forceLogout()
          }
        })
      }

      // Queue this request to be retried after the refresh completes.
      return new Promise((resolve, reject) => {
        subscribeTokenRefresh((newToken: string) => {
          originalRequest.headers = {
            ...originalRequest.headers,
            Authorization: `Bearer ${newToken}`,
          }
          // Retry the original request with the new token.
          api.request(originalRequest).then(resolve, reject)
        })
      })
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
