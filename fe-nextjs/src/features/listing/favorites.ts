import { api } from "@/lib/api/client";
import type { Item } from "@/lib/api/types";
import { getAccessToken } from "@/lib/auth/tokens";

/**
 * Favorites require an authenticated user (bearer token). The homepage renders
 * without auth, so these are wired for later use — failures surface as thrown
 * errors the UI can ignore for anonymous visitors.
 */

/** Add an item to the current user's favorites. */
export async function addFavorite(itemId: string): Promise<void> {
  const accessToken = getAccessToken();
  if (!accessToken) throw new Error("Not authenticated");

  const { error } = await api.POST("/api/v1/items/favorites", {
    body: { item_id: itemId },
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });
  if (error) throw new Error("Failed to add favorite");
}

/** Remove an item from the current user's favorites. */
export async function removeFavorite(itemId: string): Promise<void> {
  const accessToken = getAccessToken();
  if (!accessToken) throw new Error("Not authenticated");

  const { error } = await api.DELETE("/api/v1/items/favorites/{item_id}", {
    params: { path: { item_id: itemId } },
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });
  if (error) throw new Error("Failed to remove favorite");
}

/** Fetch the current user's favorited items. */
export async function getFavorites(init?: { signal?: AbortSignal }): Promise<Item[]> {
  const accessToken = getAccessToken();
  if (!accessToken) throw new Error("Not authenticated");

  const { data, error } = await api.GET("/api/v1/items/favorites", {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    signal: init?.signal,
  });
  if (error) throw new Error("Failed to load favorites");
  const rows = (data?.data ?? []) as Array<{ item?: Item }>;
  return rows.map((r) => r.item).filter((it): it is Item => Boolean(it));
}
