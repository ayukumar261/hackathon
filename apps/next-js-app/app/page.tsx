"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { getSession, logout, type SessionUser } from "@/lib/session";

type State =
  | { status: "loading" }
  | { status: "signed-in"; user: SessionUser }
  | { status: "signed-out" };

export default function Home() {
  const [state, setState] = useState<State>({ status: "loading" });

  const refresh = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const user = await getSession();
      setState(user ? { status: "signed-in", user } : { status: "signed-out" });
    } catch {
      setState({ status: "signed-out" });
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const onLogout = async () => {
    await logout();
    await refresh();
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      {state.status === "loading" && <Spinner />}

      {state.status === "signed-in" && (
        <Card className="w-full max-w-sm p-6">
          <CardHeader>
            <div className="flex items-center gap-3">
              {state.user.picture && (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={state.user.picture}
                  alt={state.user.name}
                  className="h-12 w-12 rounded-full"
                />
              )}
              <div>
                <CardTitle>{state.user.name}</CardTitle>
                <CardDescription>{state.user.email}</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardFooter>
            <Button variant="outline" onClick={onLogout} className="w-full">
              Log out
            </Button>
          </CardFooter>
        </Card>
      )}

      {state.status === "signed-out" && (
        <Card className="w-full max-w-sm p-6">
          <CardHeader>
            <CardTitle>Not signed in</CardTitle>
            <CardDescription>Sign in to continue.</CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild className="w-full">
              <Link href="/connect">Go to sign-in</Link>
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
