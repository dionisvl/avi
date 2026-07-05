"use client";

import { useLocale, useT } from "@/i18n";
import { formatRelativeTime } from "@/lib/format";
import type { Conversation } from "./get-conversations";

interface ConversationListProps {
  conversations: Conversation[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  isLoading?: boolean;
}

export function ConversationList({
  conversations,
  selectedId,
  onSelect,
  isLoading,
}: ConversationListProps) {
  const t = useT();
  const locale = useLocale();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-4 text-text-secondary">
        {t.common.loading}
      </div>
    );
  }

  if (conversations.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center p-4 text-center text-text-muted">
        <p>{t.messages.noConversations}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      {conversations.map((conv) => (
        <button
          type="button"
          key={conv.id}
          onClick={() => conv.id && onSelect(conv.id)}
          className={`flex flex-1 flex-col gap-2 border-b border-border px-4 py-3 text-left transition-colors last:border-b-0 ${
            selectedId === conv.id ? "bg-surface-soft" : "hover:bg-surface-soft"
          }`}
        >
          <div className="flex items-center justify-between gap-2">
            <h3 className="flex-1 font-medium text-text-primary">
              {conv.peer_name || t.messages.unknownPeer}
            </h3>
            {(conv.unread_count ?? 0) > 0 && (
              <span className="inline-flex min-w-5 items-center justify-center rounded-full bg-primary px-1.5 py-0.5 text-xs font-medium text-white">
                {(conv.unread_count ?? 0) > 99 ? "99+" : conv.unread_count}
              </span>
            )}
          </div>
          <p className="line-clamp-1 text-sm text-text-muted">
            {conv.last_message_preview || t.messages.noMessagesPreview}
          </p>
          {conv.last_message_at && (
            <p className="text-xs text-text-muted">
              {formatRelativeTime(conv.last_message_at, locale)}
            </p>
          )}
        </button>
      ))}
    </div>
  );
}
