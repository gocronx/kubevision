import axios from "axios"
import { toast } from "sonner"

interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
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
      return body.data as never
    }
    if (body.code === 40101) {
      // TODO: refresh token
      localStorage.removeItem("token")
      localStorage.removeItem("user")
      window.location.href = "/login"
      return Promise.reject(new Error("Token expired"))
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

export default api
