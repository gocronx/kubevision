import { describe, expect, it } from "vitest"
import { artifactPackageSource } from "./helm-source"
import type { ArtifactPackage } from "@/hooks/use-helm-catalog"

const artifact = (repositoryUrl: string): ArtifactPackage => ({
  packageId: "package-1",
  name: "nginx",
  displayName: "NGINX",
  description: "web server",
  version: "1.2.3",
  appVersion: "1.0.0",
  repository: "example",
  repositoryUrl,
})

describe("artifactPackageSource", () => {
  it("uses an HTTPS repository URL with the chart name", () => {
    expect(artifactPackageSource(artifact("https://charts.example.com"))).toEqual({
      chart: "nginx",
      repoUrl: "https://charts.example.com",
      version: "1.2.3",
    })
  })

  it("uses the complete OCI reference returned by Artifact Hub", () => {
    expect(artifactPackageSource(artifact("oci://registry.example.com/team/nginx"))).toEqual({
      chart: "oci://registry.example.com/team/nginx",
      version: "1.2.3",
    })
  })
})
