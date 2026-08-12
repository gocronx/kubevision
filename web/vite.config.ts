import path from "path"
import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? "http://127.0.0.1:18082"

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
        target: apiProxyTarget,
        changeOrigin: true,
        ws: true,
        configure: (proxy) => {
          proxy.on("error", () => {})
          proxy.on("proxyReq", (_proxyReq, _req, res) => {
            res.on("error", () => {})
          })
          proxy.on("proxyReqWs", (_proxyReq, _req, socket) => {
            socket.on("error", () => {})
          })
          proxy.on("proxyRes", (_proxyRes, req, res) => {
            res.on("error", () => {})
            req.on("error", () => {})
          })
          proxy.on("open", (socket) => {
            socket.on("error", () => {})
          })
          proxy.on("close", (_res, socket) => {
            socket.on("error", () => {})
          })
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: "./src/test/setup.ts",
    restoreMocks: true,
    exclude: ["e2e/**", "node_modules/**", "dist/**"],
  },
})
