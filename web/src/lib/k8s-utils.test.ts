import { extractColumnValue, getNestedValue, getResourceStatus, toYaml } from "./k8s-utils"

describe("Kubernetes resource formatting", () => {
  it("surfaces a container waiting reason ahead of the generic Pod phase", () => {
    const pod = {
      status: {
        phase: "Pending",
        containerStatuses: [{ ready: false, restartCount: 2, state: { waiting: { reason: "ImagePullBackOff" } } }],
      },
    }

    expect(getResourceStatus("pods", pod)).toBe("ImagePullBackOff")
    expect(extractColumnValue("pods", pod, "restarts")).toBe("2")
  })

  it("reads indexed nested paths without throwing on missing data", () => {
    const resource = { spec: { containers: [{ name: "api" }] } }
    expect(getNestedValue(resource, "spec.containers[0].name")).toBe("api")
    expect(getNestedValue(resource, "spec.containers[1].name")).toBeUndefined()
  })

  it("quotes YAML-like strings so displayed values retain their type", () => {
    expect(toYaml({ enabled: "false", count: "10", note: "a:b" })).toBe(
      'enabled: "false"\ncount: "10"\nnote: "a:b"'
    )
  })
})
