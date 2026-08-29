import { useQuery } from "@tanstack/react-query";
import { apiGet, ApiError } from "./client";
import type { DiscoveryResponse } from "./types";

export function useDiscoveryTargets() {
  return useQuery<DiscoveryResponse, ApiError>({
    queryKey: ["discovery-targets"],
    queryFn: () => apiGet<DiscoveryResponse>("/discovery/targets"),
  });
}
