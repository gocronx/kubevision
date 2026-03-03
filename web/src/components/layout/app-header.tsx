import { useState } from "react"
import { useLocation } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  Sun,
  Moon,
  Monitor,
  LogOut,
  Palette,
  Check,
  Type,
  SlidersHorizontal,
  Languages,
} from "lucide-react"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { useAuth } from "@/stores/auth-store"
import { useAppearance } from "@/components/appearance-provider"
import { GlobalSearch } from "@/components/shared/global-search"
import { SidebarCustomizer } from "@/components/sidebar-customizer"
import { COLOR_THEMES, type ColorTheme } from "@/components/color-theme-provider"
import { FONTS } from "@/components/font-provider"

const THEME_MODE_ICONS = {
  light: Sun,
  dark: Moon,
  system: Monitor,
} as const

const COLOR_DOTS: Record<ColorTheme, string> = {
  default: "bg-neutral-500",
  ocean: "bg-blue-500",
  aurora: "bg-teal-500",
  ember: "bg-amber-500",
  nebula: "bg-violet-500",
}

export function AppHeader() {
  const { t, i18n } = useTranslation()
  const location = useLocation()
  const { user, logout } = useAuth()
  const { theme, setTheme, resolvedTheme, colorTheme, setColorTheme, font, setFont } = useAppearance()
  const [showCustomizer, setShowCustomizer] = useState(false)

  const segments = location.pathname.split("/").filter(Boolean)
  const breadcrumb = segments.length > 0
    ? segments.map((s) => s.charAt(0).toUpperCase() + s.slice(1)).join(" / ")
    : "Overview"

  const isDark = resolvedTheme === "dark"
  const ThemeModeIcon = isDark ? Sun : Moon

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
        <GlobalSearch />

        {/* Appearance dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <ThemeModeIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            {/* Theme mode */}
            <DropdownMenuLabel>{t("appearance.themeMode")}</DropdownMenuLabel>
            {(["light", "dark", "system"] as const).map((mode) => {
              const Icon = THEME_MODE_ICONS[mode]
              return (
                <DropdownMenuItem key={mode} onClick={() => setTheme(mode)}>
                  <Icon className="size-4" />
                  {t(`appearance.mode_${mode}`)}
                  {theme === mode && <Check className="ml-auto size-3.5" />}
                </DropdownMenuItem>
              )
            })}

            <DropdownMenuSeparator />

            {/* Color theme — submenu */}
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Palette className="size-4" />
                {t("appearance.colorTheme")}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                {COLOR_THEMES.map((ct) => (
                  <DropdownMenuItem key={ct} onClick={() => setColorTheme(ct)}>
                    <span className={`size-3 rounded-full ${COLOR_DOTS[ct]}`} />
                    {t(`appearance.color_${ct}`)}
                    {colorTheme === ct && <Check className="ml-auto size-3.5" />}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>

            {/* Font — submenu */}
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Type className="size-4" />
                {t("appearance.font")}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                {FONTS.map((f) => (
                  <DropdownMenuItem key={f} onClick={() => setFont(f)}>
                    {t(`appearance.font_${f}`)}
                    {font === f && <Check className="ml-auto size-3.5" />}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Language toggle */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => {
            const next = i18n.language === "zh" ? "en" : "zh"
            i18n.changeLanguage(next)
            localStorage.setItem("language", next)
          }}
          title={i18n.language === "zh" ? "Switch to English" : "切换为中文"}
        >
          <Languages className="size-4" />
        </Button>

        {/* User menu */}
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
            <DropdownMenuItem onClick={() => setShowCustomizer(true)}>
              <SlidersHorizontal className="size-4" />
              {t("sidebarConfig.title")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={logout}>
              <LogOut className="size-4" />
              {t("common.logout")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <SidebarCustomizer open={showCustomizer} onOpenChange={setShowCustomizer} />
      </div>
    </header>
  )
}
