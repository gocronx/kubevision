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
  Share2,
  BarChart3,
  Gauge,
  Settings,
  GitCompareArrows,
  Puzzle,
  Bot,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"

export interface NavItem {
  titleKey: string
  icon: LucideIcon
  to: string
  iconClass?: string
}

export interface NavGroup {
  labelKey: string
  items: NavItem[]
}

// ---------------------------------------------------------------------------
// Icon color classes — sidebar uses a single accent color; the per-category
// palette is exported for use inside page content (list tables, detail cards).
// ---------------------------------------------------------------------------
const NAV_ICON = "text-blue-500"

/** Per-category icon colors for page content (tables, detail cards, etc.). */
export const categoryColor = {
  workloads: "text-green-500",
  network: "text-sky-500",
  config: "text-amber-500",
  storage: "text-purple-500",
  cluster: "text-indigo-500",
  crd: "text-pink-500",
  policy: "text-orange-500",
  admin: "text-rose-500",
} as const

export const navGroups: NavGroup[] = [
  {
    labelKey: "nav.overview",
    items: [
      { titleKey: "nav.overview", icon: LayoutDashboard, to: "/overview", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.workloads",
    items: [
      { titleKey: "nav.pods", icon: Box, to: "/pods", iconClass: NAV_ICON },
      { titleKey: "nav.deployments", icon: Server, to: "/deployments", iconClass: NAV_ICON },
      { titleKey: "nav.statefulsets", icon: Layers, to: "/statefulsets", iconClass: NAV_ICON },
      { titleKey: "nav.daemonsets", icon: Radio, to: "/daemonsets", iconClass: NAV_ICON },
      { titleKey: "nav.jobs", icon: Play, to: "/jobs", iconClass: NAV_ICON },
      { titleKey: "nav.cronjobs", icon: Clock, to: "/cronjobs", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.network",
    items: [
      { titleKey: "nav.services", icon: Network, to: "/services", iconClass: NAV_ICON },
      { titleKey: "nav.ingresses", icon: Globe, to: "/ingresses", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.config",
    items: [
      { titleKey: "nav.configmaps", icon: FileText, to: "/configmaps", iconClass: NAV_ICON },
      { titleKey: "nav.secrets", icon: KeyRound, to: "/secrets", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.storage",
    items: [
      { titleKey: "nav.persistentvolumes", icon: HardDrive, to: "/persistentvolumes", iconClass: NAV_ICON },
      { titleKey: "nav.persistentvolumeclaims", icon: Database, to: "/persistentvolumeclaims", iconClass: NAV_ICON },
      { titleKey: "nav.storageclasses", icon: Boxes, to: "/storageclasses", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.cluster",
    items: [
      { titleKey: "nav.nodes", icon: Monitor, to: "/nodes", iconClass: NAV_ICON },
      { titleKey: "nav.namespaces", icon: FolderOpen, to: "/namespaces", iconClass: NAV_ICON },
      { titleKey: "nav.events", icon: Activity, to: "/events", iconClass: NAV_ICON },
      { titleKey: "nav.topology", icon: Share2, to: "/topology", iconClass: NAV_ICON },
      { titleKey: "nav.compare", icon: GitCompareArrows, to: "/compare", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.customResources",
    items: [
      { titleKey: "nav.crds", icon: Puzzle, to: "/crds", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.policy",
    items: [
      { titleKey: "nav.quota", icon: BarChart3, to: "/quota", iconClass: NAV_ICON },
      { titleKey: "nav.limitranges", icon: Gauge, to: "/limitranges", iconClass: NAV_ICON },
    ],
  },
  {
    labelKey: "nav.settings",
    items: [
      { titleKey: "nav.settingsSecurity", icon: Settings, to: "/settings/security", iconClass: NAV_ICON },
      { titleKey: "nav.settingsAI", icon: Bot, to: "/settings/ai", iconClass: NAV_ICON },
    ],
  },
]

/** Resource type → icon color class for page content (tables, favorites, etc.) */
export const resourceIconClass: Record<string, string> = {
  pods: categoryColor.workloads,
  deployments: categoryColor.workloads,
  statefulsets: categoryColor.workloads,
  daemonsets: categoryColor.workloads,
  jobs: categoryColor.workloads,
  cronjobs: categoryColor.workloads,
  services: categoryColor.network,
  ingresses: categoryColor.network,
  configmaps: categoryColor.config,
  secrets: categoryColor.config,
  persistentvolumes: categoryColor.storage,
  persistentvolumeclaims: categoryColor.storage,
  storageclasses: categoryColor.storage,
  nodes: categoryColor.cluster,
  namespaces: categoryColor.cluster,
  events: categoryColor.cluster,
}

/** Quick lookup: route → NavItem (for pinned items rendering) */
const itemByRoute = new Map<string, NavItem>()
for (const g of navGroups) {
  for (const item of g.items) {
    itemByRoute.set(item.to, item)
  }
}

export function getNavItemByRoute(route: string): NavItem | undefined {
  return itemByRoute.get(route)
}

/** Quick lookup: labelKey → NavGroup */
const groupByKey = new Map<string, NavGroup>()
for (const g of navGroups) {
  groupByKey.set(g.labelKey, g)
}

export function getNavGroupByKey(key: string): NavGroup | undefined {
  return groupByKey.get(key)
}
