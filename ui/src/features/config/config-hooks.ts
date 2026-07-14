import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { api } from "@/lib/api/endpoints"
import type { SingBoxConfig } from "@/lib/api/types"

export const configKey = ["config"] as const

export function useConfigQuery() {
  return useQuery({ queryKey: configKey, queryFn: api.config.get })
}

export function useRawConfigQuery() {
  return useQuery({ queryKey: [...configKey, "raw"], queryFn: api.config.getRaw })
}

export function useSaveConfigMutation(raw = false) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (config: SingBoxConfig) => raw ? api.config.updateRaw(config) : api.config.update(config),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: configKey })
    },
  })
}
