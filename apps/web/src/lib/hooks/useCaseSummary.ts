"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { api, APIError } from "@/lib/api";

interface CaseSummary {
  case_number: string;
  title?: string;
  status?: string;
}

/**
 * Fetches minimal case data for sub-pages (CaseSubNav header).
 * Redirects to /cases if the case is not found (deleted or wrong id).
 */
export function useCaseSummary(id: string): CaseSummary | undefined {
  const router = useRouter();

  const { data, isError, error } = useQuery<CaseSummary>({
    queryKey: ["case-summary", id],
    queryFn: () => api.get<CaseSummary>(`/api/v1/cases/${id}`),
    staleTime: 5 * 60 * 1000,
    retry: (failCount, err) => {
      if (err instanceof APIError && err.status === 404) return false;
      return failCount < 2;
    },
  });

  useEffect(() => {
    if (isError && error instanceof APIError && error.status === 404) {
      router.replace("/cases");
    }
  }, [isError, error, router]);

  return data;
}
