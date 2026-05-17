"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Phone } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useSelectedApplicant } from "@/lib/hooks/useSelectedApplicant";
import { api } from "@/lib/api";

export default function ApplicantLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const { applicantId } = useParams<{ applicantId: string }>();
  const { applicants, selectedId, setSelectedId } = useSelectedApplicant();
  const [isCalling, setIsCalling] = useState(false);

  const handleScreen = async () => {
    if (!applicantId || isCalling) return;
    setIsCalling(true);
    try {
      await api.post(`/api/applicants/${applicantId}/screen`);
      alert("Calling your phone…");
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to start call");
    } finally {
      setIsCalling(false);
    }
  };

  useEffect(() => {
    if (applicantId && applicantId !== selectedId) {
      setSelectedId(applicantId);
    }
  }, [applicantId, selectedId, setSelectedId]);

  const applicant = applicants.find((a) => a.id === applicantId);

  return (
    <div className="flex flex-col">
      <nav className="sticky top-0 z-10 flex items-center justify-between border-b border-foreground/5 bg-white px-4 py-3">
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
      <main className="p-6">{children}</main>
    </div>
  );
}
