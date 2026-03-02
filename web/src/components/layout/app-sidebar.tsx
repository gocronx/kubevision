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
  FileText,
  KeyRound,
  HardDrive,
  Database,
  Boxes,
  Monitor,
  FolderOpen,
  Activity,
  BarChart3,
  ChevronsUpDown,
  Check,
  Settings,
  ShieldCheck,
  Webhook,
  TerminalSquare,
  GitCompareArrows,
  Users,
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
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { FavoritesPanel } from "@/components/specialized/favorites-panel"
import { useCluster } from "@/hooks/use-cluster"

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
      { titleKey: "nav.topology", icon: Activity, to: "/topology" },
      { titleKey: "nav.compare", icon: GitCompareArrows, to: "/compare" },
    ],
  },
  {
    labelKey: "nav.policy",
    items: [
      { titleKey: "nav.quota", icon: BarChart3, to: "/quota" },
      { titleKey: "nav.resourcequotas", icon: BarChart3, to: "/resourcequotas" },
      { titleKey: "nav.limitranges", icon: BarChart3, to: "/limitranges" },
    ],
  },
  {
    labelKey: "nav.settings",
    items: [
      { titleKey: "nav.settingsSecurity", icon: Settings, to: "/settings/security" },
    ],
  },
]

export function AppSidebar() {
  const { t } = useTranslation()
  const { currentCluster, clusters, setCurrentCluster } = useCluster()
  const { user } = useAuth()
  const role = user?.role ?? ""
  const showAdminSection = canAccessAdmin(role)
  const showUserManagement = canManageUsers(role)
  const currentClusterName = clusters.find((c) => c.id === currentCluster)?.name ?? t("cluster.select")

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
        {clusters.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent group-data-[collapsible=icon]:hidden">
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
            </DropdownMenuContent>
          </DropdownMenu>
        )}
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
        <div className="px-2 py-1 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
          KubeVision v0.1.0
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
