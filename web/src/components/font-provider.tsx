import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react"

export const FONTS = ["system", "inter", "jetbrains-mono"] as const
export type AppFont = (typeof FONTS)[number]

const STORAGE_KEY = "kubevision-font"

const FONT_STACKS: Record<AppFont, string> = {
  system: "ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
  inter: "'Inter', ui-sans-serif, system-ui, sans-serif",
  "jetbrains-mono": "'JetBrains Mono Variable', 'JetBrains Mono', ui-monospace, monospace",
}

interface FontContextValue {
  font: AppFont
  setFont: (font: AppFont) => void
}

const FontContext = createContext<FontContextValue | undefined>(undefined)

function getStoredFont(): AppFont {
  if (typeof window === "undefined") return "system"
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored && FONTS.includes(stored as AppFont)) return stored as AppFont
  return "jetbrains-mono"
}

export function FontProvider({ children }: { children: ReactNode }) {
  const [font, setFontState] = useState<AppFont>(getStoredFont)

  const setFont = useCallback((f: AppFont) => {
    setFontState(f)
    localStorage.setItem(STORAGE_KEY, f)
  }, [])

  useEffect(() => {
    document.documentElement.style.setProperty("--app-font-sans", FONT_STACKS[font])
  }, [font])

  return (
    <FontContext.Provider value={{ font, setFont }}>
      {children}
    </FontContext.Provider>
  )
}

export function useFont() {
  const ctx = useContext(FontContext)
  if (!ctx) throw new Error("useFont must be used within FontProvider")
  return ctx
}
