import { useLocation } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Sun, Moon, LogOut } from "lucide-react"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { useAuth } from "@/stores/auth-store"
import { useTheme } from "@/hooks/use-theme"
import { GlobalSearch } from "@/components/shared/global-search"

export function AppHeader() {
  const { t } = useTranslation()
  const location = useLocation()
  const { user, logout } = useAuth()
  const { setTheme, resolvedTheme } = useTheme()

  const segments = location.pathname.split("/").filter(Boolean)
  const breadcrumb = segments.length > 0
    ? segments.map((s) => s.charAt(0).toUpperCase() + s.slice(1)).join(" / ")
    : "Overview"

  const isDark = resolvedTheme === "dark"

  function toggleTheme() {
    setTheme(isDark ? "light" : "dark")
  }

  const initials = user?.username
    ? user.username.slice(0, 2).toUpperCase()
    : "U"

  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
      <div className="flex flex-1 items-center gap-2">
        <SidebarTrigger />
        <Separator orientation="vertical" className="mr-2 h-4" />
        <span className="text-sm text-muted-foreground">{breadcrumb}</span>
      </div>
      <div className="flex items-center gap-2">
        {/* Global search — always visible; Cmd+K / Ctrl+K also opens it */}
        <GlobalSearch />
        <Button variant="ghost" size="icon" onClick={toggleTheme}>
          {isDark ? <Sun className="size-4" /> : <Moon className="size-4" />}
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="rounded-full">
              <Avatar size="sm">
                <AvatarFallback>{initials}</AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>{user?.username ?? "User"}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={logout}>
              <LogOut className="size-4" />
              {t("common.logout")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
