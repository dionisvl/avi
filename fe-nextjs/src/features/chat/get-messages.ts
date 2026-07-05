import { API_BASE_URL } from "@/lib/api/client";

/**
 * Message from the API.
 */
export interface Message {
  id?: string;
  sender_id?: string;
  body?: string;
  created_at?: string;
  attachment_url?: string;
  attachment_mime?: string;
  attachment_size?: number;
  is_mine?: boolean;
  status?: string;
}

/**
 * Fetch messages for a conversation.
 * Returns a bare array of messages.
 */
export async function getMessages(
  conversationId: string,
  accessToken: string,
  init?: { signal?: AbortSignal },
): Promise<Message[]> {
  try {
    const response = await fetch(
      `${API_BASE_URL}/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/messages`,
      {
        method: "GET",
        headers: {
          Authorization: `Bearer ${accessToken}`,
          "Content-Type": "application/json",
        },
        signal: init?.signal,
      },
    );

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Unauthorized");
      }
      if (response.status === 403) {
        throw new Error("Forbidden");
      }
      throw new Error(`Failed to load messages: ${response.status}`);
    }

    const data = await response.json();
    return Array.isArray(data) ? data : [];
  } catch (err) {
    if (
      err instanceof Error &&
      (err.message.includes("Unauthorized") || err.message.includes("Forbidden"))
    ) {
      throw err;
    }
    throw new Error("Failed to load messages");
  }
}
