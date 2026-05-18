"use client";

import { Download } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Markdown } from "@/components/markdown";
import { useTemplateStream } from "@/lib/hooks/useTemplateStream";

export function TemplatePanel({ applicantId }: { applicantId: string }) {
  const { content, connected } = useTemplateStream(applicantId);

  return (
    <div className="flex-1 min-w-0 h-full overflow-y-auto bg-white">
      <header className="sticky top-0 z-10 bg-white border-b border-foreground/5 px-4 py-3 flex items-center justify-between min-h-12 no-print">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-medium">Screening Document</h2>
          <span
            className={cn(
              "inline-block size-2 rounded-full",
              connected ? "bg-emerald-500 animate-pulse" : "bg-gray-400",
            )}
          />
        </div>
        <Button
          variant="ghost"
          size="icon"
          disabled={!content}
          onClick={() => window.print()}
          aria-label="Download as PDF"
          title="Download as PDF"
          className="h-7 w-7"
        >
          <Download />
        </Button>
      </header>
      <div id="template-print-area" className="px-4 py-3">
        {content ? (
          <Markdown>{content}</Markdown>
        ) : (
          <p className="text-sm text-muted-foreground text-center mt-8">
            No template yet. Start a screening call to load it.
          </p>
        )}
      </div>
    </div>
  );
}
