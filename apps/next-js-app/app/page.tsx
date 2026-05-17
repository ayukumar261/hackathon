"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Spinner } from "@/components/ui/spinner";
import { useUser } from "@/lib/hooks/useUser";

export default function Home() {
  const router = useRouter();
  const { isLoading, isLoggedIn } = useUser();

  useEffect(() => {
    if (isLoading) return;
    router.replace(isLoggedIn ? "/dashboard" : "/connect");
  }, [isLoading, isLoggedIn, router]);

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-white">
      <Spinner className="size-5" />
    </div>
  );
}
