"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useApplicants, type Applicant } from "@/lib/hooks/useApplicants";
import { useSelectedPosition } from "@/lib/hooks/useSelectedPosition";

const STORAGE_KEY_PREFIX = "dashboard.selectedApplicantId:";

const storageKey = (positionId: string) => `${STORAGE_KEY_PREFIX}${positionId}`;

type SelectedApplicantContextValue = {
  selectedId: string | null;
  setSelectedId: (id: string | null) => void;
  selected: Applicant | null;
  applicants: Applicant[];
  isLoading: boolean;
};

const SelectedApplicantContext =
  createContext<SelectedApplicantContextValue | null>(null);

export function SelectedApplicantProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const { selectedId: positionId } = useSelectedPosition();
  const { applicants, isLoading } = useApplicants(positionId);
  const [selectedId, setSelectedIdState] = useState<string | null>(null);

  useEffect(() => {
    if (!positionId) {
      setSelectedIdState(null);
      return;
    }
    if (typeof window === "undefined") return;
    setSelectedIdState(window.localStorage.getItem(storageKey(positionId)));
  }, [positionId]);

  const setSelectedId = useCallback(
    (id: string | null) => {
      setSelectedIdState(id);
      if (typeof window === "undefined" || !positionId) return;
      if (id) window.localStorage.setItem(storageKey(positionId), id);
      else window.localStorage.removeItem(storageKey(positionId));
    },
    [positionId],
  );

  useEffect(() => {
    if (!positionId) return;
    if (applicants.length === 0) return;
    const exists = selectedId && applicants.some((a) => a.id === selectedId);
    if (!exists) setSelectedId(applicants[0].id);
  }, [positionId, applicants, selectedId, setSelectedId]);

  const value = useMemo<SelectedApplicantContextValue>(() => {
    const selected =
      applicants.find((a) => a.id === selectedId) ?? null;
    return { selectedId, setSelectedId, selected, applicants, isLoading };
  }, [selectedId, setSelectedId, applicants, isLoading]);

  return (
    <SelectedApplicantContext.Provider value={value}>
      {children}
    </SelectedApplicantContext.Provider>
  );
}

export function useSelectedApplicant() {
  const ctx = useContext(SelectedApplicantContext);
  if (!ctx) {
    throw new Error(
      "useSelectedApplicant must be used within a SelectedApplicantProvider",
    );
  }
  return ctx;
}
