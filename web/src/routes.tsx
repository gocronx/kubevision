import { Routes, Route, Navigate } from "react-router-dom"
import { AppLayout } from "@/components/layout/app-layout"
import { ProtectedRoute } from "@/components/shared/protected-route"
import { LoginPage } from "@/pages/auth/login"
import { OverviewPage } from "@/pages/overview/index"
import { ResourceListPage } from "@/pages/resources/resource-list"
import { ResourceDetailPage } from "@/pages/resources/resource-detail"
import { QuotaPage } from "@/pages/quota/index"
import { TopologyPage } from "@/pages/topology/index"
import { SecuritySettingsPage } from "@/pages/settings/security"
import { AISettingsPage } from "@/pages/settings/ai"
import { AdminPage } from "@/pages/admin/index"
import { WebhooksPage } from "@/pages/admin/webhooks"
import { TerminalSessionsPage } from "@/pages/admin/terminal-sessions"
import { UsersPage } from "@/pages/admin/users"
import { ComparePage } from "@/pages/compare/index"
import { PluginsPage } from "@/pages/admin/plugins"
import { OAuthCallbackPage } from "@/pages/auth/oauth-callback"
import { CRDListPage } from "@/pages/crds/index"
import { NotFoundPage } from "@/pages/not-found"
import { DirectorySettingsPage } from "@/pages/admin/directory"
import { PackageReleasesPage } from "@/pages/packages/index"
import { PackageReleaseDetailPage } from "@/pages/packages/detail"

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/oauth/:provider/callback" element={<OAuthCallbackPage />} />
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
        <Route path="quota" element={<QuotaPage />} />
        <Route path="topology" element={<TopologyPage />} />
        <Route path="settings/security" element={<SecuritySettingsPage />} />
        <Route path="settings/ai" element={<AISettingsPage />} />
        <Route path="admin" element={<AdminPage />} />
        <Route path="admin/webhooks" element={<WebhooksPage />} />
        <Route path="admin/terminal-sessions" element={<TerminalSessionsPage />} />
        <Route path="admin/users" element={<UsersPage />} />
        <Route path="admin/plugins" element={<PluginsPage />} />
        <Route path="admin/directory" element={<DirectorySettingsPage />} />
        <Route path="compare" element={<ComparePage />} />
        <Route path="crds" element={<CRDListPage />} />
        <Route path="package-releases" element={<PackageReleasesPage />} />
        <Route path="package-releases/:namespace/:name" element={<PackageReleaseDetailPage />} />
        <Route path=":resource" element={<ResourceListPage />} />
        <Route path=":resource/:name" element={<ResourceDetailPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
