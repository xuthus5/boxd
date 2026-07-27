import { useQuery } from "@tanstack/react-query"

import { api } from "@/lib/api/endpoints"

export function useDefaultPasswordStatus() {
  return useQuery({
    queryKey: ["settings", "password"],
    queryFn: api.settings.password,
    staleTime: 15_000,
    retry: false,
    refetchOnWindowFocus: false,
  })
}
