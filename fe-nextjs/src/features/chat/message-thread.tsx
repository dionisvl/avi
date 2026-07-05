"use client";

import { useEffect, useState } from "react";
import { useLocale, useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { formatRelativeTime } from "@/lib/format";
import type { Conversation } from "./get-conversations";
import type { Message } from "./get-messages";
import { MessageComposer } from "./message-composer";

interface MessageThreadProps {
  conversation: Conversation | null;
  messages: Message[];
  isLoading?: boolean;
  onBack?: () => void;
}

export function MessageThread({ conversation, messages, isLoading, onBack }: MessageThreadProps) {
  const t = useT();
  const locale = useLocale();
  const { user } = useAuth();
  const [lightboxUrl, setLightboxUrl] = useState<string | null>(null);

  // Close the image lightbox on Escape.
  useEffect(() => {
    if (!lightboxUrl) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setLightboxUrl(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [lightboxUrl]);

  if (!conversation) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-center text-text-muted">
        <p>{t.messages.selectConversation}</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-3 border-b border-border px-4 py-3 md:py-4">
        {onBack && (
          <button type="button" onClick={onBack} className="md:hidden" aria-label={t.messages.back}>
            <svg
              className="h-5 w-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              role="img"
            >
              <title>{t.messages.back}</title>
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 19l-7-7 7-7"
              />
            </svg>
          </button>
        )}
        <div>
          <h2 className="font-semibold text-text-primary">
            {conversation.peer_name || t.messages.unknownPeer}
          </h2>
          {conversation.peer_type && (
            <p className="text-xs text-text-secondary">{conversation.peer_type}</p>
          )}
        </div>
      </div>

      {/* Messages */}
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 py-4">
        {isLoading && (
          <div className="flex items-center justify-center py-8 text-text-secondary">
            {t.common.loading}
          </div>
        )}
        {!isLoading && messages.length === 0 && (
          <div className="flex items-center justify-center py-8 text-center text-text-muted">
            <p>{t.messages.noMessagesPreview}</p>
          </div>
        )}
        {!isLoading &&
          messages.map((msg) => {
            const isMine = msg.sender_id === user?.id || msg.is_mine;
            return (
              <div key={msg.id} className={`flex ${isMine ? "justify-end" : "justify-start"}`}>
                <div
                  className={`max-w-[75%] rounded-2xl px-3 py-2 ${
                    isMine ? "bg-primary text-white" : "bg-surface-soft text-text-primary"
                  }`}
                >
                  {msg.body && <p className="break-words text-sm">{msg.body}</p>}
                  {msg.attachment_url && (
                    <button
                      type="button"
                      onClick={() => setLightboxUrl(msg.attachment_url ?? null)}
                      className="mt-2 block cursor-zoom-in overflow-hidden rounded-lg transition-opacity hover:opacity-90"
                      aria-label={t.messages.openImage}
                    >
                      {/* biome-ignore lint/performance/noImgElement: Plain img required for external host support */}
                      <img
                        src={msg.attachment_url}
                        alt={t.messages.imageAlt}
                        className="max-h-[220px] max-w-[220px] object-cover"
                      />
                    </button>
                  )}
                  {msg.created_at && (
                    <p
                      className={`mt-1 text-xs ${
                        isMine ? "text-white opacity-70" : "text-text-muted"
                      }`}
                    >
                      {formatRelativeTime(msg.created_at, locale)}
                    </p>
                  )}
                </div>
              </div>
            );
          })}
      </div>

      {/* Message Composer */}
      {conversation.id && <MessageComposer conversationId={conversation.id} />}

      {/* Image lightbox */}
      {lightboxUrl && (
        <button
          type="button"
          onClick={() => setLightboxUrl(null)}
          className="fixed inset-0 z-50 flex cursor-zoom-out items-center justify-center bg-black/80 p-4"
          aria-label={t.messages.closeImage}
        >
          {/* biome-ignore lint/performance/noImgElement: Plain img required for external host support */}
          <img
            src={lightboxUrl}
            alt={t.messages.imageAlt}
            className="max-h-[90vh] max-w-[90vw] rounded-lg object-contain shadow-2xl"
          />
        </button>
      )}
    </div>
  );
}
