"use client";

import { useEffect, useState } from "react";
import { FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Markdown } from "@/components/markdown";
import { cn } from "@/lib/utils";
import { api } from "@/lib/api";
import { TEMPLATE_KEY, Template, useTemplate } from "@/lib/hooks/useTemplate";

export default function SetTemplateDialog() {
  const { template, mutate } = useTemplate();
  const [open, setOpen] = useState(false);
  const [content, setContent] = useState("");
  const [mode, setMode] = useState<"edit" | "preview">("edit");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setContent(template?.content ?? "");
      setMode("edit");
    }
  }, [open, template?.content]);

  async function handleSubmit() {
    setSaving(true);
    try {
      const updated = await api.put<Template>(TEMPLATE_KEY, { content });
      await mutate(updated, { revalidate: false });
      setOpen(false);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <FileText />
          Set template
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Update template</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-2 px-6 py-4">
          <div className="inline-flex w-fit rounded-md border border-foreground/10 bg-muted/40 p-0.5 text-sm">
            <button
              type="button"
              onClick={() => setMode("edit")}
              className={cn(
                "rounded px-3 py-1 transition",
                mode === "edit"
                  ? "bg-background shadow-sm"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              Edit
            </button>
            <button
              type="button"
              onClick={() => setMode("preview")}
              className={cn(
                "rounded px-3 py-1 transition",
                mode === "preview"
                  ? "bg-background shadow-sm"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              Preview
            </button>
          </div>
          {mode === "edit" ? (
            <textarea
              id="template"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Write your template here…"
              className="min-h-80 max-h-[60vh] w-full resize-none overflow-y-auto rounded-lg border border-foreground/10 bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          ) : (
            <div className="min-h-80 max-h-[60vh] w-full overflow-y-auto rounded-lg border border-foreground/10 bg-background px-3 py-2 shadow-sm">
              {content.trim() ? (
                <Markdown>{content}</Markdown>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Nothing to preview yet.
                </p>
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" disabled={saving}>
              Cancel
            </Button>
          </DialogClose>
          <Button onClick={handleSubmit} disabled={saving}>
            {saving ? "Saving…" : "Submit"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
