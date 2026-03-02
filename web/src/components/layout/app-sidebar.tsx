import { NavLink } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  LayoutDashboard,
  Box,
  Server,
  Layers,
  Radio,
  Play,
  Clock,
  Network,
  Globe,
  Languages,
  FileText,
  KeyRound,
  HardDrive,
  Database,
  Boxes,
  Monitor,
  FolderOpen,
  Activity,
  Share2,
  BarChart3,
  Gauge,
  ChevronsUpDown,
  Check,
  Settings,
  ShieldCheck,
  Webhook,
  TerminalSquare,
  GitCompareArrows,
  Users,
  Puzzle,
  Plug,
  Plus,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin, canManageUsers } from "@/lib/permissions"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarHeader,
  SidebarFooter,
} from "@/components/ui/sidebar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { FavoritesPanel } from "@/components/specialized/favorites-panel"
import { AddClusterDialog } from "@/components/shared/add-cluster-dialog"
import { useCluster } from "@/hooks/use-cluster"
import { useState, useCallback } from "react"
import { useTranslation as useI18nTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"

interface NavItem {
  titleKey: string
  icon: LucideIcon
  to: string
}

interface NavGroup {
  labelKey: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    labelKey: "nav.overview",
    items: [
      { titleKey: "nav.overview", icon: LayoutDashboard, to: "/overview" },
    ],
  },
  {
    labelKey: "nav.workloads",
    items: [
      { titleKey: "nav.pods", icon: Box, to: "/pods" },
      { titleKey: "nav.deployments", icon: Server, to: "/deployments" },
      { titleKey: "nav.statefulsets", icon: Layers, to: "/statefulsets" },
      { titleKey: "nav.daemonsets", icon: Radio, to: "/daemonsets" },
      { titleKey: "nav.jobs", icon: Play, to: "/jobs" },
      { titleKey: "nav.cronjobs", icon: Clock, to: "/cronjobs" },
    ],
  },
  {
    labelKey: "nav.network",
    items: [
      { titleKey: "nav.services", icon: Network, to: "/services" },
      { titleKey: "nav.ingresses", icon: Globe, to: "/ingresses" },
    ],
  },
  {
    labelKey: "nav.config",
    items: [
      { titleKey: "nav.configmaps", icon: FileText, to: "/configmaps" },
      { titleKey: "nav.secrets", icon: KeyRound, to: "/secrets" },
    ],
  },
  {
    labelKey: "nav.storage",
    items: [
      { titleKey: "nav.persistentvolumes", icon: HardDrive, to: "/persistentvolumes" },
      { titleKey: "nav.persistentvolumeclaims", icon: Database, to: "/persistentvolumeclaims" },
      { titleKey: "nav.storageclasses", icon: Boxes, to: "/storageclasses" },
    ],
  },
  {
    labelKey: "nav.cluster",
    items: [
      { titleKey: "nav.nodes", icon: Monitor, to: "/nodes" },
      { titleKey: "nav.namespaces", icon: FolderOpen, to: "/namespaces" },
      { titleKey: "nav.events", icon: Activity, to: "/events" },
      { titleKey: "nav.topology", icon: Share2, to: "/topology" },
      { titleKey: "nav.compare", icon: GitCompareArrows, to: "/compare" },
    ],
  },
  {
    labelKey: "nav.customResources",
    items: [
      { titleKey: "nav.crds", icon: Puzzle, to: "/crds" },
    ],
  },
  {
    labelKey: "nav.policy",
    items: [
      { titleKey: "nav.quota", icon: BarChart3, to: "/quota" },
      { titleKey: "nav.limitranges", icon: Gauge, to: "/limitranges" },
    ],
  },
  {
    labelKey: "nav.settings",
    items: [
      { titleKey: "nav.settingsSecurity", icon: Settings, to: "/settings/security" },
    ],
  },
]

function LanguageToggle() {
  const { i18n } = useI18nTranslation()
  const isZh = i18n.language === "zh"

  const toggle = useCallback(() => {
    const next = isZh ? "en" : "zh"
    i18n.changeLanguage(next)
    localStorage.setItem("language", next)
  }, [i18n, isZh])

  return (
    <Button
      variant="ghost"
      size="sm"
      className="h-6 w-auto px-1.5 text-xs text-muted-foreground hover:text-foreground"
      onClick={toggle}
      title={isZh ? "Switch to English" : "切换为中文"}
    >
      <Languages className="size-3.5" />
      <span className="ml-1 group-data-[collapsible=icon]:hidden">{isZh ? "EN" : "中文"}</span>
    </Button>
  )
}

export function AppSidebar() {
  const { t } = useTranslation()
  const { currentCluster, clusters, setCurrentCluster } = useCluster()
  const { user } = useAuth()
  const role = user?.role ?? ""
  const showAdminSection = canAccessAdmin(role)
  const showUserManagement = canManageUsers(role)
  const currentClusterName = clusters.find((c) => c.id === currentCluster)?.name ?? t("cluster.select")
  const [showAddCluster, setShowAddCluster] = useState(false)

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2 py-1">
          <div className="flex size-6 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <LayoutDashboard className="size-4" />
          </div>
          <span className="truncate text-sm font-semibold group-data-[collapsible=icon]:hidden">
            KubeVision
          </span>
        </div>
        {clusters.length > 0 ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring group-data-[collapsible=icon]:hidden">
                <Server className="size-4 shrink-0 text-muted-foreground" />
                <span className="flex-1 truncate text-left">{currentClusterName}</span>
                <ChevronsUpDown className="size-3.5 shrink-0 text-muted-foreground" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-56">
              {clusters.map((cluster) => (
                <DropdownMenuItem
                  key={cluster.id}
                  onClick={() => setCurrentCluster(cluster.id)}
                >
                  <Check className={`size-4 ${cluster.id === currentCluster ? "opacity-100" : "opacity-0"}`} />
                  {cluster.name}
                </DropdownMenuItem>
              ))}
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => setShowAddCluster(true)}>
                <Plus className="size-4" />
                {t("cluster.add_title")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <Button
            variant="outline"
            size="sm"
            className="w-full group-data-[collapsible=icon]:hidden"
            onClick={() => setShowAddCluster(true)}
          >
            <Plus className="mr-1.5 size-3.5" />
            {t("cluster.add_submit")}
          </Button>
        )}
        <AddClusterDialog open={showAddCluster} onOpenChange={setShowAddCluster} />
      </SidebarHeader>
      <SidebarContent>
        {/* Favorites panel — shown at the top for quick access */}
        <FavoritesPanel className="group-data-[collapsible=icon]:hidden" />

        {navGroups.map((group) => (
          <SidebarGroup key={group.labelKey}>
            <SidebarGroupLabel>{t(group.labelKey)}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => (
                  <SidebarMenuItem key={item.to}>
                    <NavLink to={item.to}>
                      {({ isActive }) => (
                        <SidebarMenuButton isActive={isActive} tooltip={t(item.titleKey)}>
                          <item.icon />
                          <span>{t(item.titleKey)}</span>
                        </SidebarMenuButton>
                      )}
                    </NavLink>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        {/* Admin links — shown only to super-admin and admin roles */}
        {showAdminSection && (
          <SidebarMenu>
            <SidebarMenuItem>
              <NavLink to="/admin">
                {({ isActive }) => (
                  <SidebarMenuButton isActive={isActive} tooltip={t("admin.title")}>
                    <ShieldCheck />
                    <span>{t("admin.title")}</span>
                  </SidebarMenuButton>
                )}
              </NavLink>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <NavLink to="/admin/webhooks">
                {({ isActive }) => (
                  <SidebarMenuButton isActive={isActive} tooltip={t("webhook.title")}>
                    <Webhook />
                    <span>{t("webhook.title")}</span>
                  </SidebarMenuButton>
                )}
              </NavLink>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <NavLink to="/admin/terminal-sessions">
                {({ isActive }) => (
                  <SidebarMenuButton isActive={isActive} tooltip={t("terminalSession.title")}>
                    <TerminalSquare />
                    <span>{t("terminalSession.title")}</span>
                  </SidebarMenuButton>
                )}
              </NavLink>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <NavLink to="/admin/plugins">
                {({ isActive }) => (
                  <SidebarMenuButton isActive={isActive} tooltip={t("plugin.title")}>
                    <Plug />
                    <span>{t("plugin.title")}</span>
                  </SidebarMenuButton>
                )}
              </NavLink>
            </SidebarMenuItem>
            {/* User management — super-admin only */}
            {showUserManagement && (
              <SidebarMenuItem>
                <NavLink to="/admin/users">
                  {({ isActive }) => (
                    <SidebarMenuButton isActive={isActive} tooltip={t("users.title")}>
                      <Users />
                      <span>{t("users.title")}</span>
                    </SidebarMenuButton>
                  )}
                </NavLink>
              </SidebarMenuItem>
            )}
          </SidebarMenu>
        )}
        <div className="flex items-center justify-between px-2 py-1 group-data-[collapsible=icon]:justify-center">
          <span className="text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
            KubeVision v0.1.0
          </span>
          <LanguageToggle />
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
