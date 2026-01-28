import { useQuery } from "@tanstack/react-query"
import { healthApi } from "@/lib/api"

export function useHealth() {
  return useQuery({
    queryKey: ["health"],
    queryFn: () => healthApi.getHealth(),
    refetchInterval: 30000, // Refresh every 30 seconds
  })
}
