"use client";

import { useEffect, useRef, useState } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTranscriptStream } from "@/lib/hooks/useTranscriptStream";

export function TranscriptPanel({ applicantId }: { applicantId: string }) {
  const { turns, connected, reset } = useTranscriptStream(applicantId);
  const [resetting, setResetting] = useState(false);

  const handleReset = async () => {
    if (resetting) return;
    setResetting(true);
    try {
      await reset();
    } finally {
      setResetting(false);
    }
  };
  const scrollRef = useRef<HTMLDivElement>(null);
  const stickToBottom = useRef(true);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onScroll = () => {
      const nearBottom =
        el.scrollHeight - el.scrollTop - el.clientHeight < 40;
      stickToBottom.current = nearBottom;
    };
    el.addEventListener("scroll", onScroll);
    return () => el.removeEventListener("scroll", onScroll);
  }, []);

  const lastText = turns[turns.length - 1]?.text ?? "";
  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !stickToBottom.current) return;
    el.scrollTop = el.scrollHeight;
  }, [turns.length, lastText]);

  return (
    <Card className="flex h-[calc(100vh-9rem)] flex-col">
      <CardHeader className="flex flex-row items-center justify-between border-b">
        <CardTitle>Live transcript</CardTitle>
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleReset}
            disabled={resetting}
            className="h-7 px-2 text-xs"
          >
            {resetting ? "Clearing…" : "Clear"}
          </Button>
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "inline-block size-2 rounded-full",
                connected
                  ? "bg-emerald-500 animate-pulse"
                  : "bg-gray-400",
              )}
            />
            {connected ? "live" : "offline"}
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex-1 overflow-hidden p-0">
        <div
          ref={scrollRef}
          className="h-full overflow-y-auto px-4 py-3 flex flex-col gap-3"
        >
          {turns.length === 0 ? (
            <p className="text-sm text-muted-foreground self-center mt-8">
              Waiting for call activity…
            </p>
          ) : (
            turns.map((t) =>
              t.role === "system" ? (
                t.text.startsWith("Sub-agent invoked") ? (
                  <div
                    key={t.id}
                    className="self-start text-xs italic text-muted-foreground py-1"
                  >
                    {t.text}
                  </div>
                ) : (
                  <div
                    key={t.id}
                    className="self-center flex items-center gap-2 my-2 text-xs uppercase tracking-wide text-muted-foreground"
                  >
                    <span className="h-px w-8 bg-border" />
                    {t.text}
                    <span className="h-px w-8 bg-border" />
                  </div>
                )
              ) : (
                <div
                  key={t.id}
                  className={cn(
                    "max-w-[80%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap",
                    t.role === "user"
                      ? "self-end bg-muted text-foreground"
                      : "self-start bg-primary/10 text-foreground",
                  )}
                >
                  {t.text}
                  {t.role === "assistant" && !t.done && (
                    <span className="ml-0.5 inline-block animate-pulse">▍</span>
                  )}
                </div>
              ),
            )
          )}
        </div>
      </CardContent>
    </Card>
  );
}
