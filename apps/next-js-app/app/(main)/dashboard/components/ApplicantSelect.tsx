"use client";

import { useRouter } from "next/navigation";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { useSelectedApplicant } from "@/lib/hooks/useSelectedApplicant";

export function ApplicantSelect() {
  const router = useRouter();
  const { applicants, selectedId, setSelectedId, isLoading } =
    useSelectedApplicant();

  if (isLoading) {
    return (
      <Select disabled>
        <SelectTrigger aria-label="Applicant" className="text-muted-foreground">
          <span className="flex items-center gap-2">
            <Spinner className="size-4 text-muted-foreground" />
            <span>Loading applicants…</span>
          </span>
        </SelectTrigger>
      </Select>
    );
  }

  if (applicants.length === 0) {
    return (
      <div className="text-sm text-muted-foreground">No applicants yet.</div>
    );
  }

  return (
    <Select
      value={selectedId ?? undefined}
      onValueChange={(value) => {
        setSelectedId(value);
        router.push(`/dashboard/${value}`);
      }}
    >
      <SelectTrigger aria-label="Applicant">
        <SelectValue placeholder="Select an applicant…" />
      </SelectTrigger>
      <SelectContent>
        {applicants.map((applicant) => (
          <SelectItem key={applicant.id} value={applicant.id}>
            <span className="truncate">{applicant.name}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export default ApplicantSelect;
