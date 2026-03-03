import { useTranslation } from "react-i18next"
import {
  Eye,
  EyeOff,
  Pin,
  PinOff,
  ChevronUp,
  ChevronDown,
  RotateCcw,
  SlidersHorizontal,
  ChevronRight,
} from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { useSidebarConfig } from "@/components/sidebar-config-provider"
import { navGroups, getNavItemByRoute } from "@/lib/nav-items"
import { cn } from "@/lib/utils"

export function SidebarCustomizer({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("sidebarConfig.title")}</DialogTitle>
          <DialogDescription>{t("sidebarConfig.description")}</DialogDescription>
        </DialogHeader>
        <CustomizerBody />
        <DialogFooter>
          <ResetButton />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ResetButton() {
  const { t } = useTranslation()
  const { resetToDefault } = useSidebarConfig()

  return (
    <Button variant="outline" size="sm" onClick={resetToDefault}>
      <RotateCcw className="size-3.5" />
      {t("sidebarConfig.reset")}
    </Button>
  )
}

function CustomizerBody() {
  const { t } = useTranslation()
  const {
    config,
    isItemHidden,
    isItemPinned,
    toggleItemVisibility,
    toggleItemPinned,
    isGroupCollapsed,
    toggleGroupCollapsed,
    moveGroup,
  } = useSidebarConfig()

  const pinnedItems = config.pinnedItems
    .map((route) => getNavItemByRoute(route))
    .filter(Boolean)

  return (
    <div className="max-h-[60vh] space-y-4 overflow-y-auto pr-1">
      {/* Pinned items section */}
      {pinnedItems.length > 0 && (
        <div>
          <h4 className="mb-2 text-sm font-medium text-muted-foreground">
            {t("sidebarConfig.pinned")}
          </h4>
          <div className="space-y-1">
            {pinnedItems.map((item) => {
              if (!item) return null
              const Icon = item.icon
              return (
                <div
                  key={item.to}
                  className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm"
                >
                  <Icon className="size-4 text-muted-foreground" />
                  <span className="flex-1">{t(item.titleKey)}</span>
                  <button
                    onClick={() => toggleItemPinned(item.to)}
                    className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                    title={t("sidebarConfig.unpin")}
                  >
                    <PinOff className="size-3.5" />
                  </button>
                </div>
              )
            })}
          </div>
          <Separator className="mt-3" />
        </div>
      )}

      {/* Groups */}
      {config.groupOrder.map((groupKey, idx) => {
        const group = navGroups.find((g) => g.labelKey === groupKey)
        if (!group) return null

        const visibleCount = group.items.filter((i) => !isItemHidden(i.to)).length
        const collapsed = isGroupCollapsed(groupKey)

        return (
          <div key={groupKey}>
            <div className="flex items-center gap-1">
              {/* Collapse toggle */}
              <button
                onClick={() => toggleGroupCollapsed(groupKey)}
                className="rounded p-0.5 text-muted-foreground hover:text-foreground"
              >
                <ChevronRight
                  className={cn("size-3.5 transition-transform", !collapsed && "rotate-90")}
                />
              </button>
              <h4 className="flex-1 text-sm font-medium">
                {t(group.labelKey)}
              </h4>
              <span className="text-xs text-muted-foreground">
                {visibleCount}/{group.items.length}
              </span>
              {/* Reorder buttons */}
              <button
                onClick={() => moveGroup(groupKey, "up")}
                disabled={idx === 0}
                className="rounded p-0.5 text-muted-foreground hover:text-foreground disabled:opacity-30"
              >
                <ChevronUp className="size-3.5" />
              </button>
              <button
                onClick={() => moveGroup(groupKey, "down")}
                disabled={idx === config.groupOrder.length - 1}
                className="rounded p-0.5 text-muted-foreground hover:text-foreground disabled:opacity-30"
              >
                <ChevronDown className="size-3.5" />
              </button>
            </div>

            {/* Items — shown when group is expanded */}
            {!collapsed && (
              <div className="mt-1 space-y-0.5 pl-5">
                {group.items.map((item) => {
                  const hidden = isItemHidden(item.to)
                  const pinned = isItemPinned(item.to)
                  const Icon = item.icon
                  return (
                    <div
                      key={item.to}
                      className={cn(
                        "flex items-center gap-2 rounded-md px-2 py-1 text-sm",
                        hidden && "opacity-40",
                      )}
                    >
                      <Icon className="size-4 text-muted-foreground" />
                      <span className="flex-1">{t(item.titleKey)}</span>
                      {/* Pin toggle — only for visible items */}
                      {!hidden && (
                        <button
                          onClick={() => toggleItemPinned(item.to)}
                          className={cn(
                            "rounded p-0.5 hover:text-foreground",
                            pinned ? "text-primary" : "text-muted-foreground",
                          )}
                          title={pinned ? t("sidebarConfig.unpin") : t("sidebarConfig.pin")}
                        >
                          {pinned ? <Pin className="size-3.5" /> : <PinOff className="size-3.5" />}
                        </button>
                      )}
                      {/* Visibility toggle */}
                      <button
                        onClick={() => toggleItemVisibility(item.to)}
                        className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                        title={hidden ? t("sidebarConfig.show") : t("sidebarConfig.hide")}
                      >
                        {hidden ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                      </button>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
