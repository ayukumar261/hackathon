"use client";

import { useEffect, useRef, useState } from "react";
import { Loader2, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useTranscriptStreamContext } from "@/lib/hooks/useTranscriptStream";

export function TranscriptPanel() {
  const { turns, connected, reset } = useTranscriptStreamContext();
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
      const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
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
    <div
      ref={scrollRef}
      className="flex-1 min-w-0 h-full overflow-y-auto border-r border-foreground/5 bg-white"
    >
      <header className="sticky top-0 z-10 bg-white border-b border-foreground/5 px-4 py-3 flex items-center justify-between min-h-12">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-medium">Live Transcript</h2>
          <span
            className={cn(
              "inline-block size-2 rounded-full",
              connected ? "bg-emerald-500 animate-pulse" : "bg-gray-400",
            )}
          />
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={handleReset}
          disabled={resetting}
          aria-label="Clear transcript"
          title="Clear transcript"
          className="h-7 w-7"
        >
          {resetting ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Trash2 className="size-3.5" />
          )}
        </Button>
      </header>
      <div className="px-4 py-3 flex flex-col gap-3">
        {turns.length === 0 ? (
          <p className="text-sm text-muted-foreground self-center mt-8">
            Waiting for call activity…
          </p>
        ) : (
          turns.map((t) =>
            t.role === "system" ? (
              t.text.startsWith("Sub-agent ") ? (
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
    </div>
  );
}
