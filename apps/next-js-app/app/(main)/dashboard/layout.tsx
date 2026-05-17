"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useUser } from "@/lib/hooks/useUser";
import { SelectedPositionProvider } from "@/lib/hooks/useSelectedPosition";
import { SelectedApplicantProvider } from "@/lib/hooks/useSelectedApplicant";
import { logout } from "@/lib/session";
import PositionSelect from "./components/PositionSelect";
import ApplicantsSidebar from "./components/ApplicantsSidebar";
import SetTemplateDialog from "./components/SetTemplateDialog";

export default function DashboardLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const router = useRouter();
  const { isLoading, isLoggedIn, mutate } = useUser();
  const [signingOut, setSigningOut] = useState(false);

  useEffect(() => {
    if (!isLoading && !isLoggedIn) {
      router.replace("/connect");
    }
  }, [isLoading, isLoggedIn, router]);

  async function handleSignOut() {
    setSigningOut(true);
    try {
      await logout();
    } finally {
      await mutate(null);
      setSigningOut(false);
    }
  }

  if (isLoading || !isLoggedIn) {
    return (
      <div className="fixed inset-0 flex items-center justify-center bg-white">
        <Spinner className="size-5" />
      </div>
    );
  }

  return (
    <SelectedPositionProvider>
      <SelectedApplicantProvider>
        <nav className="sticky top-0 z-10 flex items-center justify-between border-b border-foreground/5 bg-white px-4 py-3">
          <PositionSelect />
          <div className="flex items-center gap-2">
            <SetTemplateDialog />
            <Button variant="outline" disabled={signingOut} onClick={handleSignOut}>
              <LogOut />
              {signingOut ? "Signing out…" : "Sign out"}
            </Button>
          </div>
        </nav>
        <div className="flex">
          <ApplicantsSidebar />
          <main className="flex-1">{children}</main>
        </div>
      </SelectedApplicantProvider>
    </SelectedPositionProvider>
  );
}
