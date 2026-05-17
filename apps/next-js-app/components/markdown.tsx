"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/lib/utils";

type MarkdownProps = {
  children: string;
  className?: string;
};

export function Markdown({ children, className }: MarkdownProps) {
  return (
    <div className={cn("text-sm leading-relaxed text-foreground", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ node, ...props }) => (
            <h1 className="mt-6 mb-3 text-2xl font-semibold tracking-tight" {...props} />
          ),
          h2: ({ node, ...props }) => (
            <h2 className="mt-5 mb-2 text-xl font-semibold tracking-tight" {...props} />
          ),
          h3: ({ node, ...props }) => (
            <h3 className="mt-4 mb-2 text-lg font-semibold" {...props} />
          ),
          p: ({ node, ...props }) => <p className="my-3" {...props} />,
          a: ({ node, ...props }) => (
            <a
              className="text-primary underline underline-offset-2 hover:opacity-80"
              target="_blank"
              rel="noreferrer"
              {...props}
            />
          ),
          ul: ({ node, ...props }) => (
            <ul className="my-3 list-disc pl-6 space-y-1" {...props} />
          ),
          ol: ({ node, ...props }) => (
            <ol className="my-3 list-decimal pl-6 space-y-1" {...props} />
          ),
          li: ({ node, ...props }) => <li className="leading-relaxed" {...props} />,
          blockquote: ({ node, ...props }) => (
            <blockquote
              className="my-4 border-l-2 border-muted-foreground/30 pl-4 italic text-muted-foreground"
              {...props}
            />
          ),
          code: ({ node, className, children, ...props }) => {
            const isInline = !className?.includes("language-");
            if (isInline) {
              return (
                <code
                  className="rounded bg-muted px-1.5 py-0.5 font-mono text-[0.85em]"
                  {...props}
                >
                  {children}
                </code>
              );
            }
            return (
              <code className={cn("font-mono text-[0.85em]", className)} {...props}>
                {children}
              </code>
            );
          },
          pre: ({ node, ...props }) => (
            <pre
              className="my-4 overflow-x-auto rounded-md bg-muted p-4 text-[0.85em]"
              {...props}
            />
          ),
          hr: () => <hr className="my-6 border-border" />,
          table: ({ node, ...props }) => (
            <div className="my-4 w-full overflow-x-auto">
              <table className="w-full border-collapse text-left" {...props} />
            </div>
          ),
          th: ({ node, ...props }) => (
            <th className="border-b border-border px-3 py-2 font-semibold" {...props} />
          ),
          td: ({ node, ...props }) => (
            <td className="border-b border-border/50 px-3 py-2" {...props} />
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}

export default Markdown;
