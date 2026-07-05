import { API_BASE_URL } from "@/lib/api/client";

/**
 * Conversation from the API.
 */
export interface Conversation {
  id?: string;
  peer_id?: string;
  peer_name?: string;
  peer_type?: string;
  last_message_preview?: string;
  last_message_has_photo?: boolean;
  last_message_at?: string;
  unread_count?: number;
  peer_avatar_url?: string;
}

/**
 * Fetch conversations for the authenticated user.
 * Returns a bare array (not wrapped in {data: []}).
 */
export async function getConversations(
  accessToken: string,
  init?: { signal?: AbortSignal },
): Promise<Conversation[]> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/chat/conversations`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${accessToken}`,
        "Content-Type": "application/json",
      },
      signal: init?.signal,
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Unauthorized");
      }
      throw new Error(`Failed to load conversations: ${response.status}`);
    }

    const data = await response.json();
    return Array.isArray(data) ? data : [];
  } catch (err) {
    if (err instanceof Error && err.message.includes("Unauthorized")) {
      throw err;
    }
    throw new Error("Failed to load conversations");
  }
}
