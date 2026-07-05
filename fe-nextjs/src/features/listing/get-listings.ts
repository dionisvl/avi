import { api } from "@/lib/api/client";
import type { Item, ItemsQuery, Pagination } from "@/lib/api/types";
import { getAccessToken } from "@/lib/auth/tokens";

export type ItemsPage = { data: Item[]; pagination: Pagination };
export type ListingsQuery = ItemsQuery & { sort?: string };

/**
 * Fetch a page of listings. Auth is optional; when a bearer token is present
 * (client-side only) we send it so the API fills `is_favorited` — otherwise
 * every card would report `is_favorited: false` and toggling one heart would
 * reset the others on the next `["items"]` refetch. `sort` uses the API
 * convention e.g. "-created_at".
 */
export async function getListings(
  query: ListingsQuery = {},
  init?: { signal?: AbortSignal },
): Promise<ItemsPage> {
  // getAccessToken reads document.cookie, which is unavailable during SSR.
  const accessToken = typeof document !== "undefined" ? getAccessToken() : null;

  const { data, error } = await api.GET("/api/v1/items", {
    params: { query: query as ItemsQuery },
    ...(accessToken ? { headers: { Authorization: `Bearer ${accessToken}` } } : {}),
    signal: init?.signal,
  });
  if (error) throw new Error("Failed to load listings");
  return {
    data: data?.data ?? [],
    pagination: data?.pagination ?? { page: 1, per_page: 0, total: 0, total_pages: 0 },
  };
}
