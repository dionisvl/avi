import { api } from "@/lib/api/client";
import type { Item, Pagination } from "@/lib/api/types";
import { getAccessToken } from "@/lib/auth/tokens";

export type MyListingsPage = { data: Item[]; pagination: Pagination };

export async function getMyListings(
  sellerId: string,
  init?: { signal?: AbortSignal },
): Promise<MyListingsPage> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("No access token available");
  }

  const { data, error } = await api.GET("/api/v1/items", {
    params: {
      query: {
        seller_id: sellerId,
        statuses: "published,draft,archived,sold",
        sort: "-created_at",
        per_page: 100,
      } as never,
    },
    headers: {
      Authorization: `Bearer ${token}`,
    },
    signal: init?.signal,
  });

  if (error) {
    throw new Error("Failed to load listings");
  }

  return {
    data: data?.data ?? [],
    pagination: data?.pagination ?? { page: 1, per_page: 0, total: 0, total_pages: 0 },
  };
}
