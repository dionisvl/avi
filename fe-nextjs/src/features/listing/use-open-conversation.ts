"use client";

import { useMutation } from "@tanstack/react-query";
import {
  type ConversationResponse,
  type OpenConversationRequest,
  openConversation,
} from "./open-conversation";

export type { ConversationResponse, OpenConversationRequest };

/**
 * Mutation hook for opening a conversation with a seller.
 * Usage: const mutation = useOpenConversation(); mutation.mutate({ peer_user_id: sellerId });
 */
export function useOpenConversation() {
  return useMutation<ConversationResponse, Error, OpenConversationRequest>({
    mutationFn: openConversation,
  });
}
