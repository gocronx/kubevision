import { lazy, Suspense } from "react"
import { Routes, Route, Navigate } from "react-router-dom"
import { LoaderCircle } from "lucide-react"
import { AppLayout } from "@/components/layout/app-layout"
import { ProtectedRoute } from "@/components/shared/protected-route"

const LoginPage = lazy(() => import("@/pages/auth/login").then((module) => ({ default: module.LoginPage })))
const OverviewPage = lazy(() => import("@/pages/overview/index").then((module) => ({ default: module.OverviewPage })))
const ResourceListPage = lazy(() => import("@/pages/resources/resource-list").then((module) => ({ default: module.ResourceListPage })))
const ResourceDetailPage = lazy(() => import("@/pages/resources/resource-detail").then((module) => ({ default: module.ResourceDetailPage })))
const QuotaPage = lazy(() => import("@/pages/quota/index").then((module) => ({ default: module.QuotaPage })))
const TopologyPage = lazy(() => import("@/pages/topology/index").then((module) => ({ default: module.TopologyPage })))
const SecuritySettingsPage = lazy(() => import("@/pages/settings/security").then((module) => ({ default: module.SecuritySettingsPage })))
const AISettingsPage = lazy(() => import("@/pages/settings/ai").then((module) => ({ default: module.AISettingsPage })))
const AdminPage = lazy(() => import("@/pages/admin/index").then((module) => ({ default: module.AdminPage })))
const WebhooksPage = lazy(() => import("@/pages/admin/webhooks").then((module) => ({ default: module.WebhooksPage })))
const TerminalSessionsPage = lazy(() => import("@/pages/admin/terminal-sessions").then((module) => ({ default: module.TerminalSessionsPage })))
const UsersPage = lazy(() => import("@/pages/admin/users").then((module) => ({ default: module.UsersPage })))
const ComparePage = lazy(() => import("@/pages/compare/index").then((module) => ({ default: module.ComparePage })))
const PluginsPage = lazy(() => import("@/pages/admin/plugins").then((module) => ({ default: module.PluginsPage })))
const OAuthCallbackPage = lazy(() => import("@/pages/auth/oauth-callback").then((module) => ({ default: module.OAuthCallbackPage })))
const CRDListPage = lazy(() => import("@/pages/crds/index").then((module) => ({ default: module.CRDListPage })))
const NotFoundPage = lazy(() => import("@/pages/not-found").then((module) => ({ default: module.NotFoundPage })))
const DirectorySettingsPage = lazy(() => import("@/pages/admin/directory").then((module) => ({ default: module.DirectorySettingsPage })))
const PackageReleasesPage = lazy(() => import("@/pages/packages/index").then((module) => ({ default: module.PackageReleasesPage })))
const PackageReleaseDetailPage = lazy(() => import("@/pages/packages/detail").then((module) => ({ default: module.PackageReleaseDetailPage })))
const OperationsPage = lazy(() => import("@/pages/operations/index").then((module) => ({ default: module.OperationsPage })))

function RouteFallback() {
  return (
    <div className="flex min-h-40 items-center justify-center" role="status">
      <LoaderCircle className="size-5 animate-spin text-muted-foreground" />
    </div>
  )
}

export function AppRoutes() {
  return (
    <Suspense fallback={<RouteFallback />}>
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
        <Route path="operations" element={<OperationsPage />} />
        <Route path=":resource" element={<ResourceListPage />} />
        <Route path=":resource/:name" element={<ResourceDetailPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </Suspense>
  )
}
