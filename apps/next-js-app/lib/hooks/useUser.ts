"use client";

import useSWR from "swr";
import { ApiError, api } from "@/lib/api";
import type { SessionUser } from "@/lib/session";

export const USER_KEY = "/sessions";

const fetcher = async (path: string): Promise<SessionUser | null> => {
  try {
    const { user } = await api.get<{ user: SessionUser }>(path);
    return user;
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return null;
    throw err;
  }
};

export function useUser() {
  const { data, error, isLoading, mutate } = useSWR<SessionUser | null>(
    USER_KEY,
    fetcher,
  );

  return {
    user: data ?? null,
    isLoading,
    isLoggedIn: !!data,
    error,
    mutate,
  };
}
