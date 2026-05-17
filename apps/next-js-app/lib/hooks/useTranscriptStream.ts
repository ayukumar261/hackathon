"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { API_BASE_URL, api } from "@/lib/api";

export type Turn = {
  id: string;
  role: "user" | "assistant" | "system";
  text: string;
  done: boolean;
  ts: number;
};

type Event = {
  id: string;
  role: "user" | "assistant" | "system";
  kind: "utterance" | "token" | "turn_end" | "call_ended" | "sub_agent_invoked";
  text: string;
  ts: number;
};

export function useTranscriptStream(applicantId: string | undefined) {
  const [turns, setTurns] = useState<Turn[]>([]);
  const [connected, setConnected] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const seen = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (!applicantId) return;
    setTurns([]);
    seen.current = new Set();
    void reloadKey;

    const es = new EventSource(
      `${API_BASE_URL}/api/applicants/${applicantId}/transcript/stream`,
      { withCredentials: true },
    );

    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);

    es.onmessage = (msg) => {
      let ev: Event;
      try {
        ev = JSON.parse(msg.data);
      } catch {
        return;
      }
      if (seen.current.has(ev.id)) return;
      seen.current.add(ev.id);

      setTurns((prev) => {
        if (ev.kind === "utterance") {
          return [
            ...prev,
            { id: ev.id, role: "user", text: ev.text, done: true, ts: ev.ts },
          ];
        }
        if (ev.kind === "token") {
          const last = prev[prev.length - 1];
          if (last && last.role === "assistant" && !last.done) {
            const next = prev.slice(0, -1);
            next.push({ ...last, text: last.text + ev.text });
            return next;
          }
          return [
            ...prev,
            {
              id: ev.id,
              role: "assistant",
              text: ev.text,
              done: false,
              ts: ev.ts,
            },
          ];
        }
        if (ev.kind === "sub_agent_invoked") {
          const last = prev[prev.length - 1];
          const base =
            last && last.role === "assistant" && !last.done
              ? [...prev.slice(0, -1), { ...last, done: true }]
              : prev;
          return [
            ...base,
            {
              id: ev.id,
              role: "system",
              text: `Sub-agent invoked — ${ev.text}`,
              done: true,
              ts: ev.ts,
            },
          ];
        }
        if (ev.kind === "call_ended") {
          const last = prev[prev.length - 1];
          const next =
            last && last.role === "assistant" && !last.done
              ? [...prev.slice(0, -1), { ...last, done: true }]
              : prev;
          return [
            ...next,
            {
              id: ev.id,
              role: "system",
              text: ev.text || "Call ended",
              done: true,
              ts: ev.ts,
            },
          ];
        }
        if (ev.kind === "turn_end") {
          const last = prev[prev.length - 1];
          if (last && last.role === "assistant" && !last.done) {
            const next = prev.slice(0, -1);
            next.push({ ...last, text: ev.text || last.text, done: true });
            return next;
          }
          return [
            ...prev,
            {
              id: ev.id,
              role: "assistant",
              text: ev.text,
              done: true,
              ts: ev.ts,
            },
          ];
        }
        return prev;
      });
    };

    return () => {
      es.close();
      setConnected(false);
    };
  }, [applicantId, reloadKey]);

  const reset = useCallback(async () => {
    if (!applicantId) return;
    await api.delete(`/api/applicants/${applicantId}/transcript`);
    setTurns([]);
    seen.current = new Set();
    setReloadKey((k) => k + 1);
  }, [applicantId]);

  return { turns, connected, reset };
}
