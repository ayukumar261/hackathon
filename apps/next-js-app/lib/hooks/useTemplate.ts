"use client";

import useSWR from "swr";
import { ApiError, api } from "@/lib/api";

export const TEMPLATE_KEY = "/api/templates/me";

export type Template = {
  id: string;
  userId: string;
  content: string;
  createdAt: string;
  updatedAt: string;
};

const fetcher = async (path: string): Promise<Template | null> => {
  try {
    return await api.get<Template>(path);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return null;
    throw err;
  }
};

export function useTemplate() {
  const { data, error, isLoading, mutate } = useSWR<Template | null>(
    TEMPLATE_KEY,
    fetcher,
  );

  return {
    template: data ?? null,
    isLoading,
    error,
    mutate,
  };
}
