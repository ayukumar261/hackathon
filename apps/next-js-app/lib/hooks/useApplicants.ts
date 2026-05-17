"use client";

import useSWR from "swr";
import { ApiError, api } from "@/lib/api";

export const APPLICANTS_KEY = "/api/applicants";

export type Applicant = {
  id: string;
  positionId: string;
  name: string;
  email: string;
  phone: string;
  resume?: string;
  createdAt: string;
  updatedAt: string;
};

const fetcher = async ([path, positionId]: [string, string]): Promise<
  Applicant[]
> => {
  try {
    return await api.get<Applicant[]>(`${path}?positionId=${positionId}`);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return [];
    throw err;
  }
};

export function useApplicants(positionId: string | null) {
  const { data, error, isLoading, mutate } = useSWR<Applicant[]>(
    positionId ? [APPLICANTS_KEY, positionId] : null,
    fetcher,
  );

  return {
    applicants: data ?? [],
    isLoading,
    error,
    mutate,
  };
}
