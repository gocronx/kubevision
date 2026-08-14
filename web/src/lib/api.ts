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

const transport = axios.create({
  baseURL: "/api/v1",
  timeout: 15000,
})

transport.interceptors.request.use((config) => {
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
type RefreshWaiter = {
  resolve: (token: string) => void
  reject: (error: Error) => void
}

export class TokenRefreshQueue {
  private waiters: RefreshWaiter[] = []

  wait(): Promise<string> {
    return new Promise((resolve, reject) => {
      this.waiters.push({ resolve, reject })
    })
  }

  resolve(token: string) {
    const waiters = this.takeWaiters()
    waiters.forEach((waiter) => waiter.resolve(token))
  }

  reject(error: Error) {
    const waiters = this.takeWaiters()
    waiters.forEach((waiter) => waiter.reject(error))
  }

  get size() {
    return this.waiters.length
  }

  private takeWaiters() {
    const waiters = this.waiters
    this.waiters = []
    return waiters
  }
}

const refreshQueue = new TokenRefreshQueue()

async function tryRefreshToken(): Promise<string | null> {
  const refreshToken = localStorage.getItem("refreshToken")
  if (!refreshToken) return null

  try {
    // Use a plain axios call to avoid triggering our own interceptors.
    const res = await axios.post(
      "/api/v1/auth/refresh",
      { refreshToken },
      { timeout: 15_000 },
    )
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

transport.interceptors.response.use(
  (res) => {
    const body = res.data as ApiResponse
    if (body.code === 0) {
      // When __preserveMeta flag is set, return data+meta together
      if ((res.config as unknown as Record<string, unknown>).__preserveMeta) {
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

      const refreshedToken = refreshQueue.wait()

      if (!isRefreshing) {
        isRefreshing = true
        tryRefreshToken().then((newToken) => {
          isRefreshing = false
          if (newToken) {
            refreshQueue.resolve(newToken)
          } else {
            refreshQueue.reject(new Error("Token refresh failed"))
            forceLogout()
          }
        })
      }

      // Queue this request to be retried after the refresh completes.
      return refreshedToken.then((newToken) => {
          originalRequest.headers = {
            ...originalRequest.headers,
            Authorization: `Bearer ${newToken}`,
          }
          // Retry the original request with the new token.
          return transport.request(originalRequest)
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
export interface ApiClient {
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T>
  put<T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T>
  patch<T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T>
  request<T = unknown>(config: AxiosRequestConfig): Promise<T>
}

const api: ApiClient = {
  get: <T,>(url: string, config?: AxiosRequestConfig) => transport.get<T, T>(url, config),
  delete: <T,>(url: string, config?: AxiosRequestConfig) => transport.delete<T, T>(url, config),
  post: <T, D>(url: string, data?: D, config?: AxiosRequestConfig<D>) => transport.post<T, T, D>(url, data, config),
  put: <T, D>(url: string, data?: D, config?: AxiosRequestConfig<D>) => transport.put<T, T, D>(url, data, config),
  patch: <T, D>(url: string, data?: D, config?: AxiosRequestConfig<D>) => transport.patch<T, T, D>(url, data, config),
  request: <T,>(config: AxiosRequestConfig) => transport.request<T, T>(config),
}

export async function getWithMeta<T = unknown>(
  url: string,
  config?: AxiosRequestConfig,
): Promise<{ data: T; meta?: ApiMeta }> {
  return api.get<{ data: T; meta?: ApiMeta }>(url, {
    ...config,
    __preserveMeta: true,
  } as AxiosRequestConfig)
}

export default api
