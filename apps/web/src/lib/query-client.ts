import { QueryClient } from "@tanstack/react-query";
import { APIError } from "./api";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Don't retry on auth errors — redirect to login instead
      retry: (failureCount, error) => {
        if (error instanceof APIError && error.status === 401) return false;
        return failureCount < 2;
      },
      staleTime: 30_000,        // 30s — tuned per query via queryOptions()
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
});
