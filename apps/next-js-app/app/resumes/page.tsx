"use client";

import { useState } from "react";
import Link from "next/link";
import useSWR from "swr";
import { ApiError, api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";

type Resume = {
  id: string;
  filename: string;
  size: number;
  contentType: string;
  createdAt: string;
};

const ACCEPT = ".pdf,.doc,.docx";

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

const fetcher = (path: string) => api.get<Resume[]>(path);

export default function ResumesPage() {
  const { data, error, isLoading, mutate } = useSWR<Resume[]>(
    "/api/resumes",
    fetcher,
  );
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const onUpload = async () => {
    if (!file) return;
    setActionError(null);
    setUploading(true);
    try {
      const fd = new FormData();
      fd.append("file", file);
      await api.post<Resume>("/api/resumes", fd);
      setFile(null);
      const input = document.getElementById("resume-input") as HTMLInputElement | null;
      if (input) input.value = "";
      await mutate();
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 413) setActionError("File too large (max 10 MB).");
        else if (err.status === 415) setActionError("Unsupported file type.");
        else if (err.status === 401) setActionError("You must be signed in.");
        else setActionError(`Upload failed (${err.status}).`);
      } else {
        setActionError("Upload failed.");
      }
    } finally {
      setUploading(false);
    }
  };

  const onDownload = async (id: string) => {
    setActionError(null);
    try {
      const { url } = await api.get<{ url: string }>(`/api/resumes/${id}`);
      window.open(url, "_blank");
    } catch {
      setActionError("Could not get download URL.");
    }
  };

  const onDelete = async (id: string) => {
    setActionError(null);
    try {
      await api.delete(`/api/resumes/${id}`);
      await mutate();
    } catch {
      setActionError("Delete failed.");
    }
  };

  const unauthorized = error instanceof ApiError && error.status === 401;

  return (
    <div className="min-h-screen p-6 max-w-2xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Resumes</h1>
        <Button asChild variant="outline">
          <Link href="/">Home</Link>
        </Button>
      </div>

      {unauthorized ? (
        <Card>
          <CardHeader>
            <CardTitle>Sign in required</CardTitle>
          </CardHeader>
          <CardContent>
            <Button asChild>
              <Link href="/connect">Sign in</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Upload</CardTitle>
            </CardHeader>
            <CardContent className="flex gap-3 items-center">
              <Input
                id="resume-input"
                type="file"
                accept={ACCEPT}
                onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              />
              <Button onClick={onUpload} disabled={!file || uploading}>
                {uploading ? "Uploading…" : "Upload"}
              </Button>
            </CardContent>
          </Card>

          {actionError && (
            <p className="text-sm text-red-600">{actionError}</p>
          )}

          {isLoading && <Spinner />}

          {data && data.length === 0 && (
            <p className="text-sm text-muted-foreground">No resumes yet.</p>
          )}

          <div className="space-y-3">
            {data?.map((r) => (
              <Card key={r.id}>
                <CardContent className="flex items-center justify-between gap-4 py-4">
                  <div className="min-w-0">
                    <p className="font-medium truncate">{r.filename}</p>
                    <p className="text-xs text-muted-foreground">
                      {formatSize(r.size)} · {new Date(r.createdAt).toLocaleString()}
                    </p>
                  </div>
                  <div className="flex gap-2 shrink-0">
                    <Button variant="outline" onClick={() => onDownload(r.id)}>
                      Download
                    </Button>
                    <Button variant="destructive" onClick={() => onDelete(r.id)}>
                      Delete
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
