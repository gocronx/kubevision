import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar"
import { SidebarConfigProvider } from "@/components/sidebar-config-provider"
import { AppSidebar } from "./app-sidebar"
import { AppHeader } from "./app-header"
import { AIChatWidget } from "@/components/ai-chat/ai-chat-widget"
import { Outlet } from "react-router-dom"
import { readSidebarOpen } from "./sidebar-state-preference"

export function AppLayout() {
  return (
    <SidebarConfigProvider>
      <SidebarProvider defaultOpen={readSidebarOpen()}>
        <AppSidebar />
        <SidebarInset>
          <AppHeader />
          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-auto p-3 sm:p-4">
            <Outlet />
          </div>
        </SidebarInset>
        <AIChatWidget />
      </SidebarProvider>
    </SidebarConfigProvider>
  )
}
