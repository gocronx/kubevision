import { NavLink } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  Languages,
  ChevronsUpDown,
  Check,
  ShieldCheck,
  Webhook,
  TerminalSquare,
  Users,
  Plug,
  Plus,
  Pin,
  LayoutDashboard,
  Server,
} from "lucide-react"
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
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
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
import { useSidebarConfig } from "@/components/sidebar-config-provider"
import { navGroups, getNavItemByRoute, getNavGroupByKey } from "@/lib/nav-items"
import { useState, useCallback } from "react"
import { useTranslation as useI18nTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"

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
  const { config, isItemHidden, isItemPinned, isGroupCollapsed, toggleGroupCollapsed } = useSidebarConfig()
  const role = user?.role ?? ""
  const showAdminSection = canAccessAdmin(role)
  const showUserManagement = canManageUsers(role)
  const currentClusterName = clusters.find((c) => c.id === currentCluster)?.name ?? t("cluster.select")
  const [showAddCluster, setShowAddCluster] = useState(false)

  // Resolve pinned items from config
  const pinnedItems = config.pinnedItems
    .map((route) => getNavItemByRoute(route))
    .filter((item) => item && !isItemHidden(item.to))

  // Resolve ordered groups
  const orderedGroups = config.groupOrder
    .map((key) => getNavGroupByKey(key))
    .filter(Boolean)
  // Append any groups not in the order (safety net)
  for (const g of navGroups) {
    if (!config.groupOrder.includes(g.labelKey)) {
      orderedGroups.push(g)
    }
  }

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
        {/* Favorites panel */}
        <FavoritesPanel className="group-data-[collapsible=icon]:hidden" />

        {/* Pinned items — quick access section */}
        {pinnedItems.length > 0 && (
          <SidebarGroup>
            <SidebarGroupLabel>
              <Pin className="mr-1 size-3" />
              {t("sidebarConfig.pinned")}
            </SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {pinnedItems.map((item) => {
                  if (!item) return null
                  return (
                    <SidebarMenuItem key={`pin-${item.to}`}>
                      <NavLink to={item.to}>
                        {({ isActive }) => (
                          <SidebarMenuButton isActive={isActive} tooltip={t(item.titleKey)}>
                            <item.icon />
                            <span>{t(item.titleKey)}</span>
                          </SidebarMenuButton>
                        )}
                      </NavLink>
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

        {/* Navigation groups — ordered and filtered */}
        {orderedGroups.map((group) => {
          if (!group) return null
          const visibleItems = group.items.filter((item) => !isItemHidden(item.to))
          if (visibleItems.length === 0) return null
          const collapsed = isGroupCollapsed(group.labelKey)

          return (
            <Collapsible key={group.labelKey} defaultOpen={!collapsed}>
              <SidebarGroup>
                <CollapsibleTrigger asChild>
                  <SidebarGroupLabel className="cursor-pointer">
                    {t(group.labelKey)}
                  </SidebarGroupLabel>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <SidebarGroupContent>
                    <SidebarMenu>
                      {visibleItems.map((item) => (
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
                </CollapsibleContent>
              </SidebarGroup>
            </Collapsible>
          )
        })}
      </SidebarContent>
      <SidebarFooter>
        {/* Admin links */}
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
