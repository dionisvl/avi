"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/auth/tokens";
import type { Message } from "./get-messages";
import type { SendMessageRequest } from "./send-message";
import { sendMessage } from "./send-message";

/**
 * Mutation hook for sending a message to a conversation.
 * Invalidates the messages query for that conversation and the conversations list
 * so that previews and last-message timestamps update.
 *
 * Usage:
 *   const mutation = useSendMessage(conversationId);
 *   mutation.mutate({ body: "Hello" });
 */
export function useSendMessage(conversationId: string) {
  const queryClient = useQueryClient();

  return useMutation<Message, Error, SendMessageRequest>({
    mutationFn: async (req: SendMessageRequest) => {
      const token = getAccessToken();
      if (!token) {
        throw new Error("Not authenticated");
      }
      return sendMessage(conversationId, req, token);
    },
    onSuccess: () => {
      // Invalidate messages for this conversation
      queryClient.invalidateQueries({
        queryKey: ["messages", conversationId],
      });
      // Invalidate conversations list so preview updates
      queryClient.invalidateQueries({
        queryKey: ["conversations"],
      });
    },
  });
}
