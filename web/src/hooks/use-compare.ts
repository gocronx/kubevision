import { useMutation } from "@tanstack/react-query"
import api from "@/lib/api"

export interface CompareTarget {
  cluster: string
  namespace: string
  resource: string
  name: string
}

export interface CompareRequest {
  source: CompareTarget
  target: CompareTarget
}

export interface CompareResult {
  sourceYaml: string
  targetYaml: string
  sourceRef: string
  targetRef: string
}

export function useCompareResources() {
  return useMutation<CompareResult, Error, CompareRequest>({
    mutationFn: async (req) => {
      const res = await api.post("/compare", req)
      return res as unknown as CompareResult
    },
  })
}
