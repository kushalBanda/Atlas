import { useQuery } from "@tanstack/react-query";
import { apiGet, ApiError } from "./client";
import type { Stats } from "./types";

export function useStats() {
  return useQuery<Stats, ApiError>({
    queryKey: ["stats"],
    queryFn: () => apiGet<Stats>("/stats"),
  });
}
