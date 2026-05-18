"use client";

import { useEffect, useRef, useState } from "react";
import { ExternalLink } from "lucide-react";
import { Document, Page, pdfjs } from "react-pdf";
import "react-pdf/dist/Page/AnnotationLayer.css";
import "react-pdf/dist/Page/TextLayer.css";
import { Spinner } from "@/components/ui/spinner";
import { Button } from "@/components/ui/button";
import { api, ApiError, joinUrl } from "@/lib/api";

pdfjs.GlobalWorkerOptions.workerSrc = `https://unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.mjs`;

export function ResumePanel({ applicantId }: { applicantId: string }) {
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null);
  const [fileSrc, setFileSrc] = useState<{
    url: string;
    withCredentials: boolean;
  } | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "missing" | "error">(
    "loading",
  );
  const [numPages, setNumPages] = useState(0);
  const [width, setWidth] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    setState("loading");
    setDownloadUrl(null);
    setFileSrc(null);
    setNumPages(0);
    api
      .get<{ url: string }>(`/api/applicants/${applicantId}/resume`)
      .then((res) => {
        if (cancelled) return;
        setDownloadUrl(res.url);
        setFileSrc({
          url: joinUrl(`/api/applicants/${applicantId}/resume/file`),
          withCredentials: true,
        });
        setState("ready");
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 404) {
          setState("missing");
        } else {
          setState("error");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [applicantId]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const update = () => setWidth(el.clientWidth);
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, [state]);

  return (
    <div className="flex-1 min-w-0 h-full flex flex-col border-r border-foreground/5 bg-white">
      <header className="sticky top-0 z-10 bg-white border-b border-foreground/5 px-4 py-3 flex items-center justify-between min-h-12">
        <h2 className="text-sm font-medium">Resume</h2>
        {downloadUrl && (
          <Button
            asChild
            variant="ghost"
            size="icon"
            aria-label="Open in new tab"
            title="Open in new tab"
            className="h-7 w-7"
          >
            <a href={downloadUrl} target="_blank" rel="noopener noreferrer">
              <ExternalLink className="size-3.5" />
            </a>
          </Button>
        )}
      </header>
      <div
        ref={containerRef}
        className="flex-1 min-h-0 overflow-y-auto bg-muted/40"
      >
        {state === "loading" && (
          <div className="flex justify-center pt-8">
            <Spinner className="size-4" />
          </div>
        )}
        {state === "missing" && (
          <p className="text-sm text-muted-foreground text-center mt-8">
            No resume uploaded.
          </p>
        )}
        {state === "error" && (
          <p className="text-sm text-muted-foreground text-center mt-8">
            Failed to load resume.
          </p>
        )}
        {state === "ready" && fileSrc && width > 0 && (
          <Document
            file={fileSrc}
            onLoadSuccess={({ numPages }) => setNumPages(numPages)}
            onLoadError={() => setState("error")}
            loading={
              <div className="flex justify-center pt-8">
                <Spinner className="size-4" />
              </div>
            }
            className="flex flex-col items-center gap-3 py-3"
          >
            {Array.from({ length: numPages }, (_, i) => (
              <Page
                key={i}
                pageNumber={i + 1}
                width={Math.max(width - 24, 200)}
                renderAnnotationLayer={false}
                renderTextLayer
                className="shadow-sm"
              />
            ))}
          </Document>
        )}
      </div>
    </div>
  );
}
