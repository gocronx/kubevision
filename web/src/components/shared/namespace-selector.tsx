import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { ChevronsUpDown, Check, FolderOpen } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useResourceList } from "@/hooks/use-resource"
import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/lib/utils"

interface NamespaceSelectorProps {
  clusterID: string
  value: string
  onChange: (namespace: string) => void
  className?: string
}

export function NamespaceSelector({
  clusterID,
  value,
  onChange,
  className,
}: NamespaceSelectorProps) {
  const { t } = useTranslation()
  const { data } = useResourceList(clusterID, "namespaces", {
    enabled: !!clusterID,
  })

  const namespaces = useMemo(() => {
    if (!data?.items) return []
    return data.items
      .map((item) => {
        const meta = item.metadata as { name?: string } | undefined
        return meta?.name ?? ""
      })
      .filter(Boolean)
      .sort()
  }, [data])

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className={cn("gap-2", className)}>
          <FolderOpen className="size-4" />
          <span className="max-w-[150px] truncate">
            {value ? value : t("common.allNamespaces")}
          </span>
          <ChevronsUpDown className="size-3 opacity-50" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[220px]">
        <ScrollArea className="max-h-[300px]">
          <DropdownMenuItem
            onClick={() => onChange("")}
          >
            <Check
              className={cn(
                "mr-2 size-4",
                value === "" ? "opacity-100" : "opacity-0"
              )}
            />
            {t("common.allNamespaces")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          {namespaces.map((ns) => (
            <DropdownMenuItem
              key={ns}
              onClick={() => onChange(ns)}
            >
              <Check
                className={cn(
                  "mr-2 size-4",
                  value === ns ? "opacity-100" : "opacity-0"
                )}
              />
              {ns}
            </DropdownMenuItem>
          ))}
        </ScrollArea>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
