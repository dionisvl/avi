import { API_BASE_URL } from "@/lib/api/client";
import type { Message } from "./get-messages";

/**
 * Request for sending a message with optional attachment.
 * At least one of body or file must be present.
 */
export interface SendMessageRequest {
  body?: string;
  file?: File;
}

/**
 * Send a message to a conversation.
 * Uses multipart/form-data to support both text and file uploads.
 * Requires Bearer authentication (access token).
 *
 * Uses a raw fetch (not the typed openapi client) because the generated
 * schema registers chat paths without the `/api/v1` prefix that the client's
 * baseUrl expects — mirrors src/features/chat/get-conversations.ts.
 * Throws an error if the request fails.
 */
export async function sendMessage(
  conversationId: string,
  { body, file }: SendMessageRequest,
  accessToken: string,
): Promise<Message> {
  if (!body && !file) {
    throw new Error("At least one of body or file must be provided");
  }

  const formData = new FormData();
  if (body) {
    formData.append("body", body);
  }
  if (file) {
    formData.append("file", file);
  }

  try {
    const response = await fetch(
      `${API_BASE_URL}/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/messages`,
      {
        method: "POST",
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
        body: formData,
      },
    );

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Unauthorized");
      }
      if (response.status === 400) {
        throw new Error("Invalid message data");
      }
      throw new Error(`Failed to send message: ${response.status}`);
    }

    const data = await response.json();
    return data as Message;
  } catch (err) {
    if (
      err instanceof Error &&
      (err.message.includes("Unauthorized") || err.message.includes("Invalid message data"))
    ) {
      throw err;
    }
    throw new Error("Failed to send message");
  }
}
