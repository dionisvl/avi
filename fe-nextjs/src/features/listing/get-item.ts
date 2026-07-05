import { api } from "@/lib/api/client";
import type { Item } from "@/lib/api/types";

/** Fetch a single listing by UUID or slug. */
export async function getItem(id: string, init?: { signal?: AbortSignal }): Promise<Item | null> {
  const { data, error, response } = await api.GET("/api/v1/items/{id}", {
    params: { path: { id } },
    signal: init?.signal,
  });

  if (error) {
    if (response.status === 404) return null;
    throw new Error("Failed to load item");
  }

  return data?.data ?? null;
}
