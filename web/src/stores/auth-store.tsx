import { createContext, useContext, useState, useCallback } from "react"
import type { ReactNode } from "react"

interface User {
  id: number
  username: string
  role: string
  totpEnabled?: boolean
}

interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean
}

interface AuthContextValue extends AuthState {
  login: (token: string, user: User, refreshToken?: string) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function getStoredAuth(): AuthState {
  const token = localStorage.getItem("token")
  const userJson = localStorage.getItem("user")
  if (token && userJson) {
    try {
      const user = JSON.parse(userJson) as User
      return { token, user, isAuthenticated: true }
    } catch {
      return { token: null, user: null, isAuthenticated: false }
    }
  }
  return { token: null, user: null, isAuthenticated: false }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [authState, setAuthState] = useState<AuthState>(getStoredAuth)

  const login = useCallback((token: string, user: User, refreshToken?: string) => {
    localStorage.setItem("token", token)
    localStorage.setItem("user", JSON.stringify(user))
    if (refreshToken) {
      localStorage.setItem("refreshToken", refreshToken)
    }
    setAuthState({ token, user, isAuthenticated: true })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem("token")
    localStorage.removeItem("refreshToken")
    localStorage.removeItem("user")
    setAuthState({ token: null, user: null, isAuthenticated: false })
  }, [])

  return (
    <AuthContext.Provider
      value={{
        ...authState,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider")
  }
  return context
}
