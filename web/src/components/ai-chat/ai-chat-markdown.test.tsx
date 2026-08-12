import { render, screen, within } from "@testing-library/react"
import { ChatMarkdown } from "./ai-chat-markdown"

const example = `根据当前集群状态，**整体健康，可以正常运行**。

### 集群概况

- **节点：** 1 个
- **Pod：** 6 个，全部为 Running

\`\`\`yaml
spec:
  replicas: 0
\`\`\`

| 组件 | 状态 |
| --- | --- |
| coredns | 正常 |
`

describe("ChatMarkdown", () => {
  it("renders headings, emphasis, lists, fenced code and GFM tables", () => {
    render(<ChatMarkdown content={example} />)

    expect(screen.getByRole("heading", { level: 3, name: "集群概况" })).toBeInTheDocument()
    expect(screen.getByText("整体健康，可以正常运行")).toHaveClass("font-semibold")
    expect(screen.getByRole("list")).toBeInTheDocument()
    expect(screen.getByText(/replicas: 0/).closest("pre")).toBeInTheDocument()
    expect(within(screen.getByRole("table")).getByText("coredns")).toBeInTheDocument()
  })

  it("does not render raw HTML from model output", () => {
    render(<ChatMarkdown content={'<img src=x onerror="alert(1)">'} />)

    expect(document.querySelector("img")).not.toBeInTheDocument()
  })
})
