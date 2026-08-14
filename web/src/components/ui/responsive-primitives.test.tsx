import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { Card, CardContent, CardHeader } from "./card"
import { Tabs, TabsList, TabsTrigger } from "./tabs"

describe("responsive UI primitives", () => {
  it("allows tab labels to scroll without shrinking", () => {
    render(
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="configuration">Configuration</TabsTrigger>
        </TabsList>
      </Tabs>,
    )

    expect(screen.getByRole("tablist")).toHaveClass("max-w-full", "overflow-x-auto")
    for (const tab of screen.getAllByRole("tab")) {
      expect(tab).toHaveClass("flex-none", "whitespace-nowrap")
    }
  })

  it("uses compact card spacing until the small breakpoint", () => {
    render(
      <Card>
        <CardHeader>Header</CardHeader>
        <CardContent>Content</CardContent>
      </Card>,
    )

    expect(screen.getByText("Header")).toHaveClass("px-4", "sm:px-6")
    expect(screen.getByText("Content")).toHaveClass("px-4", "sm:px-6")
  })
})
