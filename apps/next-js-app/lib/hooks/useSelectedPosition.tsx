"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { usePositions, type Position } from "@/lib/hooks/usePositions";

const STORAGE_KEY = "dashboard.selectedPositionId";

type SelectedPositionContextValue = {
  selectedId: string | null;
  setSelectedId: (id: string | null) => void;
  selected: Position | null;
  positions: Position[];
  isLoading: boolean;
};

const SelectedPositionContext =
  createContext<SelectedPositionContextValue | null>(null);

export function SelectedPositionProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { positions, isLoading } = usePositions();
  const [selectedId, setSelectedIdState] = useState<string | null>(() => {
    if (typeof window === "undefined") return null;
    return window.localStorage.getItem(STORAGE_KEY);
  });

  const setSelectedId = useCallback((id: string | null) => {
    setSelectedIdState(id);
    if (typeof window === "undefined") return;
    if (id) window.localStorage.setItem(STORAGE_KEY, id);
    else window.localStorage.removeItem(STORAGE_KEY);
  }, []);

  useEffect(() => {
    if (positions.length === 0) return;
    const exists = selectedId && positions.some((p) => p.id === selectedId);
    if (!exists) setSelectedId(positions[0].id);
  }, [positions, selectedId, setSelectedId]);

  const value = useMemo<SelectedPositionContextValue>(() => {
    const selected =
      positions.find((p) => p.id === selectedId) ?? null;
    return { selectedId, setSelectedId, selected, positions, isLoading };
  }, [selectedId, setSelectedId, positions, isLoading]);

  return (
    <SelectedPositionContext.Provider value={value}>
      {children}
    </SelectedPositionContext.Provider>
  );
}

export function useSelectedPosition() {
  const ctx = useContext(SelectedPositionContext);
  if (!ctx) {
    throw new Error(
      "useSelectedPosition must be used within a SelectedPositionProvider",
    );
  }
  return ctx;
}
