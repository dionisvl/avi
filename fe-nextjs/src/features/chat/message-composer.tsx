"use client";

import { useRef, useState } from "react";
import { useT } from "@/i18n";
import { useSendMessage } from "./use-send-message";

interface MessageComposerProps {
  conversationId: string;
}

export function MessageComposer({ conversationId }: MessageComposerProps) {
  const t = useT();
  const [body, setBody] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [filePreview, setFilePreview] = useState<string | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const mutation = useSendMessage(conversationId);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      setErrorMessage(null);

      // Create preview
      const reader = new FileReader();
      reader.onload = (event) => {
        setFilePreview(event.target?.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleClearFile = () => {
    setSelectedFile(null);
    setFilePreview(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage(null);

    if (!body && !selectedFile) {
      return;
    }

    mutation.mutate(
      {
        body: body || undefined,
        file: selectedFile || undefined,
      },
      {
        onSuccess: () => {
          setBody("");
          handleClearFile();
        },
        onError: (error) => {
          setErrorMessage(error.message || t.messages.sendError);
        },
      },
    );
  };

  const isLoading = mutation.isPending;
  const isDisabled = (!body && !selectedFile) || isLoading;

  return (
    <div className="shrink-0 border-t border-border bg-surface px-4 py-3">
      {/* Error message */}
      {errorMessage && (
        <div className="mb-3 rounded-md bg-red-50 p-2 text-sm text-red-600 dark:bg-red-950 dark:text-red-200">
          {errorMessage}
        </div>
      )}

      {/* File preview */}
      {filePreview && (
        <div className="mb-3 flex items-start gap-2">
          <div className="relative inline-block">
            {/* biome-ignore lint/performance/noImgElement: Plain img required for external host support */}
            <img
              src={filePreview}
              alt={t.messages.imageAlt}
              className="max-h-20 max-w-20 rounded-lg object-cover"
            />
            <button
              type="button"
              onClick={handleClearFile}
              className="absolute -right-2 -top-2 flex h-6 w-6 items-center justify-center rounded-full bg-red-600 text-white hover:bg-red-700"
              aria-label={t.messages.removeAttachment}
            >
              ×
            </button>
          </div>
          <div className="flex flex-1 flex-col">
            <p className="text-xs font-medium text-text-primary">{selectedFile?.name}</p>
            <p className="text-xs text-text-muted">{((selectedFile?.size ?? 0) / 1024) | 0} KB</p>
          </div>
        </div>
      )}

      {/* Composer form */}
      <form onSubmit={handleSubmit} className="flex gap-2">
        <div className="flex flex-1 items-end gap-2">
          {/* Text input */}
          <textarea
            value={body}
            onChange={(e) => {
              setBody(e.target.value);
              setErrorMessage(null);
            }}
            placeholder={t.messages.composerPlaceholder}
            className="max-h-24 flex-1 resize-none rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text-primary placeholder-text-muted focus:border-primary focus:outline-none"
            rows={1}
            style={{ minHeight: "40px" }}
          />

          {/* Attach file button */}
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={isLoading}
            className="flex h-10 w-10 items-center justify-center rounded-lg border border-border bg-surface-soft text-text-primary hover:bg-surface disabled:opacity-50"
            aria-label={t.messages.attachImage}
            title={t.messages.attachImage}
          >
            <svg
              className="h-5 w-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              role="img"
            >
              <title>{t.messages.attachImage}</title>
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a2 2 0 00-2-2H8a2 2 0 00-2 2v4m0 0a2 2 0 002 2h8a2 2 0 002-2m0 0V7a2 2 0 00-2-2H8a2 2 0 00-2 2"
              />
            </svg>
          </button>

          {/* Send button */}
          <button
            type="submit"
            disabled={isDisabled}
            className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-white hover:opacity-90 disabled:opacity-50"
            aria-label={t.messages.send}
            title={t.messages.send}
          >
            {isLoading ? (
              <svg
                className="h-5 w-5 animate-spin"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                role="img"
              >
                <title>{t.common.loading}</title>
                <circle cx="12" cy="12" r="1" fill="currentColor" />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 19a7 7 0 100-14 7 7 0 000 14z"
                />
              </svg>
            ) : (
              <svg
                className="h-5 w-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                role="img"
              >
                <title>{t.messages.send}</title>
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
                />
              </svg>
            )}
          </button>
        </div>
      </form>

      {/* Hidden file input */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        onChange={handleFileSelect}
        className="hidden"
      />
    </div>
  );
}
