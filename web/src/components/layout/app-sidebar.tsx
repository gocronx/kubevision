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
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
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
import { FavoritesPanel } from "@/components/specialized/favorites-panel"

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
]

export function AppSidebar() {
  const { t } = useTranslation()

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
        <div className="px-2 py-1 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
          KubeVision v0.1.0
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
