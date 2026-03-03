import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { App } from "./App"
import "@fontsource/inter/400.css"
import "@fontsource/inter/500.css"
import "@fontsource/inter/600.css"
import "@fontsource/inter/700.css"
// @ts-expect-error -- @fontsource-variable packages lack type declarations
import "@fontsource-variable/jetbrains-mono"
import "./index.css"
import "./i18n"

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
)
