"use client";

import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth/context";
import { getAccessToken } from "@/lib/auth/tokens";
import { getMessages } from "./get-messages";

/**
 * Hook to fetch messages for a specific conversation.
 * Only fetches when conversationId is provided and user is authenticated.
 */
export function useMessages(conversationId: string | null) {
  const { isAuthenticated } = useAuth();

  return useQuery({
    queryKey: ["messages", conversationId],
    queryFn: async ({ signal }) => {
      if (!conversationId) {
        throw new Error("No conversation selected");
      }
      const token = getAccessToken();
      if (!token) {
        throw new Error("Not authenticated");
      }
      return getMessages(conversationId, token, { signal });
    },
    enabled: isAuthenticated && !!conversationId,
    staleTime: 10 * 1000, // 10 seconds
  });
}
