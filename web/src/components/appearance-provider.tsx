import type { ReactNode } from "react"
import { ThemeProvider } from "next-themes"
import { ColorThemeProvider, useColorTheme } from "./color-theme-provider"
import { FontProvider, useFont } from "./font-provider"
import { useTheme } from "@/hooks/use-theme"

export function AppearanceProvider({ children }: { children: ReactNode }) {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
      <ColorThemeProvider>
        <FontProvider>
          {children}
        </FontProvider>
      </ColorThemeProvider>
    </ThemeProvider>
  )
}

/** Unified hook aggregating theme mode, color theme, and font */
export function useAppearance() {
  const { theme, setTheme, resolvedTheme } = useTheme()
  const { colorTheme, setColorTheme } = useColorTheme()
  const { font, setFont } = useFont()

  return {
    theme,
    setTheme,
    resolvedTheme,
    colorTheme,
    setColorTheme,
    font,
    setFont,
  } as const
}
