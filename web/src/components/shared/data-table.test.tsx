import { render, screen, within } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { DataTable, type DataTableColumn } from "./data-table"

type Item = {
  metadata: { uid: string; namespace?: string; name: string }
  status?: { phase?: string }
}

const columns: DataTableColumn<Item>[] = [
  {
    key: "name",
    label: "Name",
    sortable: true,
    render: (item) => item.metadata.name,
  },
]

const items: Item[] = [
  { metadata: { uid: "3", namespace: "team-b", name: "api" } },
  { metadata: { uid: "2", namespace: "team-a", name: "web" } },
  { metadata: { uid: "1", namespace: "team-a", name: "api" } },
]

function rowNames() {
  return within(screen.getByRole("table"))
    .getAllByRole("row")
    .slice(1)
    .map((row) => row.textContent)
}

describe("DataTable stable ordering", () => {
  it("keeps wide tables scrollable on narrow viewports", () => {
    const { container } = render(<DataTable columns={columns} data={items} />)

    expect(container.firstElementChild).toHaveClass("max-w-full", "overflow-x-auto")
    expect(screen.getByRole("table")).toHaveStyle({ minWidth: "36rem" })
  })

  it("keeps the namespace and name order when refreshed data arrives shuffled", () => {
    const view = render(
      <DataTable columns={columns} data={items} getRowKey={(item) => item.metadata.uid} />,
    )

    expect(rowNames()).toEqual(["api", "web", "api"])

    view.rerender(
      <DataTable columns={columns} data={[items[1], items[2], items[0]]} getRowKey={(item) => item.metadata.uid} />,
    )

    expect(rowNames()).toEqual(["api", "web", "api"])
  })

  it("uses namespace and name as tie breakers for a configured sort", () => {
    const sameStatus = items.map((item) => ({ ...item, status: { phase: "Running" } }))
    render(
      <DataTable
        columns={[...columns, { key: "phase", label: "Phase", sortable: true, render: (item) => item.status?.phase }]}
        data={sameStatus}
        getRowKey={(item) => item.metadata.uid}
        defaultSort={{ key: "phase", direction: "asc" }}
      />,
    )

    expect(rowNames()).toEqual(["apiRunning", "webRunning", "apiRunning"])
  })
})
