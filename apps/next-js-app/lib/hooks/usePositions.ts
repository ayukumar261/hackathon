"use client";

import useSWR from "swr";
import { ApiError, api } from "@/lib/api";

export const POSITIONS_KEY = "/api/positions";

export type Position = {
  id: string;
  title: string;
  company: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};

const fetcher = async (path: string): Promise<Position[]> => {
  try {
    return await api.get<Position[]>(path);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return [];
    throw err;
  }
};

export function usePositions() {
  const { data, error, isLoading, mutate } = useSWR<Position[]>(
    POSITIONS_KEY,
    fetcher,
  );

  return {
    positions: data ?? [],
    isLoading,
    error,
    mutate,
  };
}
