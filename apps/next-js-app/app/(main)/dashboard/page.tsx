"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useSelectedApplicant } from "@/lib/hooks/useSelectedApplicant";

export default function DashboardPage() {
  const router = useRouter();
  const { selectedId } = useSelectedApplicant();

  useEffect(() => {
    if (selectedId) router.replace(`/dashboard/${selectedId}`);
  }, [selectedId, router]);

  return (
    <p className="text-sm text-gray-500 p-6">
      Select an applicant from the sidebar.
    </p>
  );
}
