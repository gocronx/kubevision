import { render, screen } from "@testing-library/react"
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom"
import { AuthProvider } from "@/stores/auth-store"
import { ProtectedRoute } from "./protected-route"

function LoginProbe() {
  const location = useLocation()
  const from = location.state?.from?.pathname
  return <p>login destination: {from}</p>
}

describe("ProtectedRoute", () => {
  it("redirects anonymous visitors and preserves the requested destination", () => {
    render(
      <MemoryRouter initialEntries={["/admin/users"]}>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<LoginProbe />} />
            <Route
              path="/admin/users"
              element={<ProtectedRoute><p>private content</p></ProtectedRoute>}
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    )

    expect(screen.getByText("login destination: /admin/users")).toBeInTheDocument()
    expect(screen.queryByText("private content")).not.toBeInTheDocument()
  })

  it("renders private content for a stored authenticated session", () => {
    localStorage.setItem("token", "local-test-token")
    localStorage.setItem("user", JSON.stringify({ id: 7, username: "tester", role: "viewer" }))

    render(
      <MemoryRouter initialEntries={["/overview"]}>
        <AuthProvider>
          <ProtectedRoute><p>private content</p></ProtectedRoute>
        </AuthProvider>
      </MemoryRouter>
    )

    expect(screen.getByText("private content")).toBeInTheDocument()
  })
})
