"use client";

import { useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { Phone } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useSelectedApplicant } from "@/lib/hooks/useSelectedApplicant";
import { api } from "@/lib/api";
import {
  TranscriptStreamProvider,
  useTranscriptStreamContext,
} from "@/lib/hooks/useTranscriptStream";

export default function ApplicantLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const { applicantId } = useParams<{ applicantId: string }>();

  return (
    <TranscriptStreamProvider applicantId={applicantId}>
      <ApplicantLayoutInner applicantId={applicantId}>
        {children}
      </ApplicantLayoutInner>
    </TranscriptStreamProvider>
  );
}

function ApplicantLayoutInner({
  applicantId,
  children,
}: {
  applicantId: string;
  children: React.ReactNode;
}) {
  const { applicants, selectedId, setSelectedId } = useSelectedApplicant();
  const { turns } = useTranscriptStreamContext();
  const [isCalling, setIsCalling] = useState(false);
  const toastIdRef = useRef<string | number | null>(null);
  const connectedRef = useRef(false);

  const handleScreen = async () => {
    if (!applicantId || isCalling) return;
    setIsCalling(true);
    connectedRef.current = false;
    toastIdRef.current = toast.loading("Starting call…");
    try {
      await api.post(`/api/applicants/${applicantId}/screen`);
      toast.loading("Calling your phone…", { id: toastIdRef.current });
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Failed to start call";
      toast.error(message, { id: toastIdRef.current });
      toastIdRef.current = null;
    } finally {
      setIsCalling(false);
    }
  };

  useEffect(() => {
    if (toastIdRef.current == null || turns.length === 0) return;
    const last = turns[turns.length - 1];

    const isCallEnded =
      last.role === "system" && /call ended/i.test(last.text);
    if (isCallEnded) {
      toast.success("Call ended", {
        id: toastIdRef.current,
        duration: 4000,
      });
      toastIdRef.current = null;
      connectedRef.current = false;
      return;
    }

    if (!connectedRef.current && (last.role === "user" || last.role === "assistant")) {
      connectedRef.current = true;
      toast.success("Call connected", { id: toastIdRef.current });
    }
  }, [turns]);

  useEffect(() => {
    if (applicantId && applicantId !== selectedId) {
      setSelectedId(applicantId);
    }
  }, [applicantId, selectedId, setSelectedId]);

  const applicant = applicants.find((a) => a.id === applicantId);

  return (
    <div className="flex flex-col h-full">
      <nav className="flex items-center justify-between border-b border-foreground/5 bg-white px-4 py-3">
        <h1 className="text-sm font-medium">{applicant?.name ?? ""}</h1>
        <Button
          size="lg"
          className="bg-green-600 hover:bg-green-700 text-white"
          onClick={handleScreen}
          disabled={isCalling}
        >
          <Phone data-icon="inline-start" fill="currentColor" strokeWidth={0} />
          {isCalling ? "Calling…" : "Screen applicant"}
        </Button>
      </nav>
      <main className="flex-1 min-h-0">{children}</main>
    </div>
  );
}
