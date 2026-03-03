import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react"

export const COLOR_THEMES = ["default", "ocean", "aurora", "ember", "nebula"] as const
export type ColorTheme = (typeof COLOR_THEMES)[number]

const STORAGE_KEY = "kubevision-color-theme"

interface ColorThemeContextValue {
  colorTheme: ColorTheme
  setColorTheme: (theme: ColorTheme) => void
}

const ColorThemeContext = createContext<ColorThemeContextValue | undefined>(undefined)

function getStoredTheme(): ColorTheme {
  if (typeof window === "undefined") return "default"
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored && COLOR_THEMES.includes(stored as ColorTheme)) return stored as ColorTheme
  return "default"
}

export function ColorThemeProvider({ children }: { children: ReactNode }) {
  const [colorTheme, setColorThemeState] = useState<ColorTheme>(getStoredTheme)

  const setColorTheme = useCallback((theme: ColorTheme) => {
    setColorThemeState(theme)
    localStorage.setItem(STORAGE_KEY, theme)
  }, [])

  useEffect(() => {
    const root = document.documentElement
    // Remove any existing color-* class
    COLOR_THEMES.forEach((t) => root.classList.remove(`color-${t}`))
    // Apply new class (skip for default — no CSS overrides needed)
    if (colorTheme !== "default") {
      root.classList.add(`color-${colorTheme}`)
    }
  }, [colorTheme])

  return (
    <ColorThemeContext.Provider value={{ colorTheme, setColorTheme }}>
      {children}
    </ColorThemeContext.Provider>
  )
}

export function useColorTheme() {
  const ctx = useContext(ColorThemeContext)
  if (!ctx) throw new Error("useColorTheme must be used within ColorThemeProvider")
  return ctx
}
