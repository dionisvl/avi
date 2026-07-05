"use client";

import { useMemo, useState } from "react";
import { useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { ConversationList } from "./conversation-list";
import { MessageThread } from "./message-thread";
import { useConversations } from "./use-conversations";
import { useMessages } from "./use-messages";

/**
 * Main messages page content. Requires client-side auth context.
 * Two-pane UI: conversations list + message thread.
 * On mobile: single pane with navigation between list and thread.
 */
export function MessagesPageContent() {
  const t = useT();
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [selectedConvId, setSelectedConvId] = useState<string | null>(null);
  const [showThread, setShowThread] = useState(false);

  // Fetch conversations
  const {
    data: conversations = [],
    isLoading: convsLoading,
    error: convsError,
  } = useConversations();

  // Fetch messages for selected conversation
  const {
    data: messages = [],
    isLoading: msgsLoading,
    error: msgsError,
  } = useMessages(selectedConvId && showThread ? selectedConvId : null);

  // Find selected conversation object
  const selectedConversation = useMemo(
    () => conversations.find((c) => c.id === selectedConvId) || null,
    [conversations, selectedConvId],
  );

  // Handle conversation selection on mobile: show thread
  const handleSelectConversation = (id: string) => {
    setSelectedConvId(id);
    setShowThread(true);
  };

  // Handle back button on mobile: return to list
  const handleBack = () => {
    setShowThread(false);
  };

  // Auth state: show login prompt
  if (!isAuthenticated && !authLoading) {
    return (
      <main className="flex flex-1 flex-col items-center justify-center px-4 py-8">
        <div className="max-w-md text-center">
          <h1 className="text-h2 text-text-primary">{t.header.messages}</h1>
          <p className="mt-3 text-body text-text-secondary">{t.states.loginPrompt}</p>
          <a
            href="/login"
            className="mt-6 inline-block rounded-lg bg-primary px-6 py-2 font-medium text-white transition-colors hover:opacity-90"
          >
            {t.auth.logIn}
          </a>
        </div>
      </main>
    );
  }

  // Loading auth
  if (authLoading) {
    return (
      <main className="flex flex-1 items-center justify-center">
        <p className="text-text-secondary">{t.common.loading}</p>
      </main>
    );
  }

  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <div className="container-avi min-h-0 flex-1 py-6 pb-24 md:py-8 lg:pb-8">
        <div
          className="overflow-hidden rounded-container border border-border bg-surface"
          style={{ height: "calc(100dvh - 160px)", minHeight: "520px" }}
        >
          <div className="flex h-full min-h-0 flex-col md:flex-row">
            {/* Conversations list pane */}
            <div
              className={`min-h-0 border-r border-border md:w-80 ${
                showThread ? "hidden md:flex" : "flex"
              } flex-col`}
            >
              <div className="shrink-0 border-b border-border px-4 py-3">
                <h1 className="text-h2 text-text-primary">{t.header.messages}</h1>
              </div>
              <div className="flex-1 overflow-y-auto">
                {convsError && (
                  <div className="p-4 text-center text-sm text-red-600">{t.states.error}</div>
                )}
                {!convsError && (
                  <ConversationList
                    conversations={conversations}
                    selectedId={selectedConvId}
                    onSelect={handleSelectConversation}
                    isLoading={convsLoading}
                  />
                )}
              </div>
            </div>

            {/* Message thread pane */}
            <div className={`min-h-0 flex-1 flex-col ${showThread ? "flex" : "hidden md:flex"}`}>
              {msgsError && (
                <div className="flex flex-1 items-center justify-center">
                  <p className="text-text-secondary">{t.states.error}</p>
                </div>
              )}
              {!msgsError && (
                <MessageThread
                  conversation={selectedConversation}
                  messages={messages}
                  isLoading={msgsLoading}
                  onBack={showThread ? handleBack : undefined}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
