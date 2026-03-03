import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar"
import { SidebarConfigProvider } from "@/components/sidebar-config-provider"
import { AppSidebar } from "./app-sidebar"
import { AppHeader } from "./app-header"
import { Outlet } from "react-router-dom"

export function AppLayout() {
  return (
    <SidebarConfigProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset>
          <AppHeader />
          <div className="flex-1 overflow-auto p-4">
            <Outlet />
          </div>
        </SidebarInset>
      </SidebarProvider>
    </SidebarConfigProvider>
  )
}
