"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { Item, Pagination } from "@/lib/api/types";
import { getAccessToken } from "@/lib/auth/tokens";

export interface FavoritesPage {
  data: Item[];
  pagination: Pagination;
}

/**
 * Fetch the current user's favorited items.
 * Requires an access token stored in cookies.
 */
async function getFavorites(accessToken: string, signal?: AbortSignal): Promise<FavoritesPage> {
  const { data, error } = await api.GET("/api/v1/items/favorites", {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
    signal,
  });

  if (error) throw new Error("Failed to load favorites");

  // Each favorite has structure { id, item_id, item, created_at }
  // We extract the item from each favorite
  const rows = (data?.data ?? []) as Array<{ item?: Item }>;
  const items = rows.map((r) => r.item).filter((it): it is Item => Boolean(it));

  return {
    data: items,
    pagination: data?.pagination ?? { page: 1, per_page: 0, total: 0, total_pages: 0 },
  };
}

/**
 * Hook to fetch the current user's favorites.
 * Returns loading/error states automatically.
 */
export function useFavorites() {
  return useQuery({
    // Distinct key from the flat Item[] query used by FavoriteHeartButton
    // (["favorites"]) — this one returns a {data, pagination} page object, so
    // sharing the key would corrupt the shared cache and crash consumers.
    queryKey: ["favorites", "page"],
    queryFn: async ({ signal }) => {
      const token = getAccessToken();
      if (!token) {
        throw new Error("Not authenticated");
      }
      return getFavorites(token, signal);
    },
    retry: false,
  });
}
