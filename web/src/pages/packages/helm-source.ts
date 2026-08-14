import type { ArtifactPackage } from "@/hooks/use-helm-catalog"
import type { PackageChangeInput } from "@/hooks/use-package-releases"

export function artifactPackageSource(item: ArtifactPackage): PackageChangeInput["source"] {
  if (item.repositoryUrl.startsWith("oci://")) {
    return { chart: item.repositoryUrl, version: item.version }
  }
  return { chart: item.name, repoUrl: item.repositoryUrl, version: item.version }
}
