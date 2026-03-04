import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
        ws: true,
        configure: (proxy) => {
          proxy.on("error", () => {})
          proxy.on("proxyReqWs", (_proxyReq, _req, socket) => {
            socket.on("error", () => {})
          })
          proxy.on("proxyRes", (_proxyRes, _req, res) => {
            res.on("error", () => {})
          })
          proxy.on("open", (socket) => {
            socket.on("error", () => {})
          })
        },
      },
    },
  },
})
