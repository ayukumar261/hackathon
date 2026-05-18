"use client";

import { useParams } from "next/navigation";
import { Spinner } from "@/components/ui/spinner";
import { useSelectedApplicant } from "@/lib/hooks/useSelectedApplicant";
import { TranscriptPanel } from "./components/TranscriptPanel";
import { TemplatePanel } from "./components/TemplatePanel";

export default function ApplicantDetailPage() {
  const { applicantId } = useParams<{ applicantId: string }>();
  const { applicants, isLoading } = useSelectedApplicant();

  if (isLoading) {
    return (
      <div className="flex justify-center pt-8">
        <Spinner className="size-4" />
      </div>
    );
  }

  const applicant = applicants.find((a) => a.id === applicantId);

  if (!applicant) {
    return <p className="text-sm text-gray-500">Applicant not found.</p>;
  }

  return (
    <div className="flex h-full min-h-0">
      <TranscriptPanel />
      <TemplatePanel applicantId={applicantId} />
    </div>
  );
}
