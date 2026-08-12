import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react"

const STORAGE_KEY = "kubevision-sidebar-config"

export interface SidebarConfig {
  /** Item routes that the user has hidden */
  hiddenItems: string[]
  /** Item routes that the user has pinned to the top */
  pinnedItems: string[]
  /** Ordered array of group labelKeys — controls display order */
  groupOrder: string[]
  /** Groups the user has collapsed by default */
  collapsedGroups: string[]
}

/** Default group order matching the hardcoded navGroups */
export const DEFAULT_GROUP_ORDER = [
  "nav.overview",
  "nav.workloads",
  "nav.network",
  "nav.config",
  "nav.storage",
  "nav.cluster",
  "nav.customResources",
  "nav.policy",
  "nav.settings",
]

const DEFAULT_CONFIG: SidebarConfig = {
  hiddenItems: [],
  pinnedItems: [],
  groupOrder: DEFAULT_GROUP_ORDER,
  collapsedGroups: [],
}

interface SidebarConfigContextValue {
  config: SidebarConfig
  isItemHidden: (route: string) => boolean
  isItemPinned: (route: string) => boolean
  isGroupCollapsed: (groupKey: string) => boolean
  setGroupCollapsed: (groupKey: string, collapsed: boolean) => void
  toggleItemVisibility: (route: string) => void
  toggleItemPinned: (route: string) => void
  toggleGroupCollapsed: (groupKey: string) => void
  moveGroup: (groupKey: string, direction: "up" | "down") => void
  resetToDefault: () => void
}

const SidebarConfigContext = createContext<SidebarConfigContextValue | undefined>(undefined)

function loadConfig(): SidebarConfig {
  if (typeof window === "undefined") return DEFAULT_CONFIG
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return DEFAULT_CONFIG
    const parsed = JSON.parse(raw) as Partial<SidebarConfig>
    return {
      hiddenItems: Array.isArray(parsed.hiddenItems) ? parsed.hiddenItems : [],
      pinnedItems: Array.isArray(parsed.pinnedItems) ? parsed.pinnedItems : [],
      groupOrder: Array.isArray(parsed.groupOrder) && parsed.groupOrder.length > 0
        ? parsed.groupOrder
        : DEFAULT_GROUP_ORDER,
      collapsedGroups: Array.isArray(parsed.collapsedGroups) ? parsed.collapsedGroups : [],
    }
  } catch {
    return DEFAULT_CONFIG
  }
}

function saveConfig(config: SidebarConfig) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(config))
}

export function SidebarConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<SidebarConfig>(loadConfig)

  const update = useCallback((fn: (prev: SidebarConfig) => SidebarConfig) => {
    setConfig((prev) => {
      const next = fn(prev)
      saveConfig(next)
      return next
    })
  }, [])

  const isItemHidden = useCallback((route: string) => config.hiddenItems.includes(route), [config.hiddenItems])
  const isItemPinned = useCallback((route: string) => config.pinnedItems.includes(route), [config.pinnedItems])
  const isGroupCollapsed = useCallback((groupKey: string) => config.collapsedGroups.includes(groupKey), [config.collapsedGroups])

  const toggleItemVisibility = useCallback((route: string) => {
    update((prev) => {
      const hidden = prev.hiddenItems.includes(route)
        ? prev.hiddenItems.filter((r) => r !== route)
        : [...prev.hiddenItems, route]
      // If hiding an item, also unpin it
      const pinned = hidden.includes(route)
        ? prev.pinnedItems.filter((r) => r !== route)
        : prev.pinnedItems
      return { ...prev, hiddenItems: hidden, pinnedItems: pinned }
    })
  }, [update])

  const toggleItemPinned = useCallback((route: string) => {
    update((prev) => {
      const pinned = prev.pinnedItems.includes(route)
        ? prev.pinnedItems.filter((r) => r !== route)
        : [...prev.pinnedItems, route]
      return { ...prev, pinnedItems: pinned }
    })
  }, [update])

  const toggleGroupCollapsed = useCallback((groupKey: string) => {
    update((prev) => {
      const collapsed = prev.collapsedGroups.includes(groupKey)
        ? prev.collapsedGroups.filter((k) => k !== groupKey)
        : [...prev.collapsedGroups, groupKey]
      return { ...prev, collapsedGroups: collapsed }
    })
  }, [update])

  const setGroupCollapsed = useCallback((groupKey: string, collapsed: boolean) => {
    update((prev) => {
      const isCollapsed = prev.collapsedGroups.includes(groupKey)
      if (isCollapsed === collapsed) return prev

      return {
        ...prev,
        collapsedGroups: collapsed
          ? [...prev.collapsedGroups, groupKey]
          : prev.collapsedGroups.filter((key) => key !== groupKey),
      }
    })
  }, [update])

  const moveGroup = useCallback((groupKey: string, direction: "up" | "down") => {
    update((prev) => {
      const order = [...prev.groupOrder]
      const idx = order.indexOf(groupKey)
      if (idx < 0) return prev
      const targetIdx = direction === "up" ? idx - 1 : idx + 1
      if (targetIdx < 0 || targetIdx >= order.length) return prev
      ;[order[idx], order[targetIdx]] = [order[targetIdx], order[idx]]
      return { ...prev, groupOrder: order }
    })
  }, [update])

  const resetToDefault = useCallback(() => {
    const fresh = { ...DEFAULT_CONFIG, groupOrder: [...DEFAULT_GROUP_ORDER] }
    setConfig(fresh)
    saveConfig(fresh)
  }, [])

  const value = useMemo<SidebarConfigContextValue>(() => ({
    config,
    isItemHidden,
    isItemPinned,
    isGroupCollapsed,
    setGroupCollapsed,
    toggleItemVisibility,
    toggleItemPinned,
    toggleGroupCollapsed,
    moveGroup,
    resetToDefault,
  }), [config, isItemHidden, isItemPinned, isGroupCollapsed, setGroupCollapsed, toggleItemVisibility, toggleItemPinned, toggleGroupCollapsed, moveGroup, resetToDefault])

  return (
    <SidebarConfigContext.Provider value={value}>
      {children}
    </SidebarConfigContext.Provider>
  )
}

export function useSidebarConfig() {
  const ctx = useContext(SidebarConfigContext)
  if (!ctx) throw new Error("useSidebarConfig must be used within SidebarConfigProvider")
  return ctx
}
