import { Routes, Route, Navigate } from "react-router-dom"
import { AppLayout } from "@/components/layout/app-layout"
import { ProtectedRoute } from "@/components/shared/protected-route"
import { LoginPage } from "@/pages/auth/login"
import { OverviewPage } from "@/pages/overview/index"
import { ResourceListPage } from "@/pages/resources/resource-list"
import { ResourceDetailPage } from "@/pages/resources/resource-detail"
import { NotFoundPage } from "@/pages/not-found"

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AppLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/overview" replace />} />
        <Route path="overview" element={<OverviewPage />} />
        <Route path=":resource" element={<ResourceListPage />} />
        <Route path=":resource/:name" element={<ResourceDetailPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
