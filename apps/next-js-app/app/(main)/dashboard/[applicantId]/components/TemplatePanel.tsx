"use client";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { Markdown } from "@/components/markdown";
import { useTemplateStream } from "@/lib/hooks/useTemplateStream";

export function TemplatePanel({ applicantId }: { applicantId: string }) {
  const { content, connected } = useTemplateStream(applicantId);

  return (
    <Card className="flex h-[calc(100vh-9rem)] flex-col">
      <CardHeader className="flex flex-row items-center justify-between border-b">
        <CardTitle>Screening template</CardTitle>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span
            className={cn(
              "inline-block size-2 rounded-full",
              connected ? "bg-emerald-500 animate-pulse" : "bg-gray-400",
            )}
          />
          {connected ? "live" : "offline"}
        </div>
      </CardHeader>
      <CardContent className="flex-1 overflow-hidden p-0">
        <div className="h-full overflow-y-auto px-4 py-3">
          {content ? (
            <Markdown>{content}</Markdown>
          ) : (
            <p className="text-sm text-muted-foreground text-center mt-8">
              No template yet. Start a screening call to load it.
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
