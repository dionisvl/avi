import { api } from "@/lib/api/client";
import type { Item } from "@/lib/api/types";
import { getAccessToken } from "@/lib/auth/tokens";

export async function getOwnedItem(
  id: string,
  init?: { signal?: AbortSignal },
): Promise<Item | null> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("No access token available");
  }

  const { data, error, response } = await api.GET("/api/v1/items/{id}", {
    params: { path: { id } },
    headers: {
      Authorization: `Bearer ${token}`,
    },
    signal: init?.signal,
  });

  if (error) {
    if (response.status === 404) return null;
    throw new Error(error.detail || error.title || "Failed to load listing");
  }

  return data?.data ?? null;
}
