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
} from "lucide-react"
import type { LucideIcon } from "lucide-react"

export interface NavItem {
  titleKey: string
  icon: LucideIcon
  to: string
}

export interface NavGroup {
  labelKey: string
  items: NavItem[]
}

export const navGroups: NavGroup[] = [
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
