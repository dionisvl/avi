import { API_BASE_URL } from "@/lib/api/client";
import { getAccessToken } from "@/lib/auth/tokens";

/**
 * Request body for POST /api/v1/chat/conversations.
 */
export interface OpenConversationRequest {
  peer_user_id: string;
}

/**
 * Conversation returned by POST /api/v1/chat/conversations.
 */
export interface ConversationResponse {
  id?: string;
  peer_id?: string;
  peer_name?: string;
  peer_type?: string;
  peer_avatar_url?: string;
  last_message_preview?: string;
  last_message_at?: string;
  last_message_has_photo?: boolean;
  unread_count?: number;
}

/**
 * Open or get an existing conversation with a seller.
 * Requires Bearer authentication (access token in cookie).
 *
 * Uses a raw fetch (not the typed openapi client) because the generated
 * schema registers chat paths without the `/api/v1` prefix that the client's
 * baseUrl expects — mirrors src/features/chat/get-conversations.ts.
 * Throws an error if the request fails.
 */
export async function openConversation(
  req: OpenConversationRequest,
): Promise<ConversationResponse> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("No access token available");
  }

  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/chat/conversations`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(req),
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Unauthorized");
      }
      throw new Error(`Failed to open conversation: ${response.status}`);
    }

    const data = await response.json();
    return data as ConversationResponse;
  } catch (err) {
    if (err instanceof Error && err.message.includes("Unauthorized")) {
      throw err;
    }
    throw new Error("Failed to open conversation");
  }
}
