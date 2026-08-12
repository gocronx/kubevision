import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { SidebarConfigProvider, useSidebarConfig } from "./sidebar-config-provider"

function GroupPreferenceHarness() {
  const { isGroupCollapsed, setGroupCollapsed } = useSidebarConfig()
  const groupKey = "nav.workloads"

  return (
    <button onClick={() => setGroupCollapsed(groupKey, !isGroupCollapsed(groupKey))}>
      {isGroupCollapsed(groupKey) ? "collapsed" : "expanded"}
    </button>
  )
}

function renderHarness() {
  return render(
    <SidebarConfigProvider>
      <GroupPreferenceHarness />
    </SidebarConfigProvider>
  )
}

describe("SidebarConfigProvider group state", () => {
  it("restores a collapsed navigation group after remounting", () => {
    const firstRender = renderHarness()
    fireEvent.click(screen.getByRole("button", { name: "expanded" }))
    expect(screen.getByRole("button", { name: "collapsed" })).toBeInTheDocument()

    firstRender.unmount()
    renderHarness()

    expect(screen.getByRole("button", { name: "collapsed" })).toBeInTheDocument()
  })

  it("persists expanding a previously collapsed group", () => {
    localStorage.setItem("kubevision-sidebar-config", JSON.stringify({
      hiddenItems: [],
      pinnedItems: [],
      groupOrder: ["nav.workloads"],
      collapsedGroups: ["nav.workloads"],
    }))

    const firstRender = renderHarness()
    fireEvent.click(screen.getByRole("button", { name: "collapsed" }))
    firstRender.unmount()
    renderHarness()

    expect(screen.getByRole("button", { name: "expanded" })).toBeInTheDocument()
  })
})
