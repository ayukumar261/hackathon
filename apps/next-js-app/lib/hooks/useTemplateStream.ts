"use client";

import { useEffect, useState } from "react";
import { API_BASE_URL } from "@/lib/api";

type Event = {
  kind: "snapshot" | "update";
  content: string;
};

export function useTemplateStream(applicantId: string | undefined) {
  const [content, setContent] = useState("");
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!applicantId) return;
    setContent("");

    const es = new EventSource(
      `${API_BASE_URL}/api/applicants/${applicantId}/template/stream`,
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
      setContent(ev.content);
    };

    return () => {
      es.close();
      setConnected(false);
    };
  }, [applicantId]);

  return { content, connected };
}
