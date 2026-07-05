"use client";

import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth/context";
import { getAccessToken } from "@/lib/auth/tokens";
import { getConversations } from "./get-conversations";

/**
 * Hook to fetch conversations for the authenticated user.
 */
export function useConversations() {
  const { isAuthenticated } = useAuth();

  return useQuery({
    queryKey: ["conversations"],
    queryFn: async ({ signal }) => {
      const token = getAccessToken();
      if (!token) {
        throw new Error("Not authenticated");
      }
      return getConversations(token, { signal });
    },
    enabled: isAuthenticated,
    staleTime: 30 * 1000, // 30 seconds
  });
}
