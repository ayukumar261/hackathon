"use client";

import { useParams, useRouter } from "next/navigation";
import { Spinner } from "@/components/ui/spinner";
import { useSelectedApplicant } from "@/lib/hooks/useSelectedApplicant";

export default function ApplicantsSidebar() {
  const router = useRouter();
  const params = useParams<{ applicantId?: string }>();
  const activeId = params?.applicantId ?? null;
  const { applicants, isLoading, setSelectedId } = useSelectedApplicant();

  return (
    <aside className="w-1/4 min-w-[260px] min-h-[calc(100vh-3.75rem)] border-r border-foreground/5 overflow-y-auto px-4 py-3 space-y-2">
      {isLoading ? (
        <div className="flex justify-center pt-8">
          <Spinner className="size-4" />
        </div>
      ) : applicants.length === 0 ? (
        <p className="text-sm text-gray-500 px-2 pt-4">No applicants yet</p>
      ) : (
        applicants.map((a) => {
          const selected = a.id === activeId;
          return (
            <button
              key={a.id}
              type="button"
              onClick={() => {
                setSelectedId(a.id);
                router.push(`/dashboard/${a.id}`);
              }}
              className={`w-full text-left border rounded-md p-3 transition-colors ${
                selected
                  ? "border-gray-300 bg-gray-50"
                  : "border-gray-200 hover:bg-gray-50"
              }`}
            >
              <div className="font-medium text-sm">{a.name}</div>
              <div className="text-xs text-gray-600 truncate">{a.email}</div>
              <div className="text-xs text-gray-500">{a.phone}</div>
            </button>
          );
        })
      )}
    </aside>
  );
}
