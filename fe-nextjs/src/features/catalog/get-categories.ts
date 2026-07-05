import { api } from "@/lib/api/client";
import type { Category } from "@/lib/api/types";

/** Fetch top-level categories. `locale` defaults to the API default (en). */
export async function getCategories(
  locale: "en" | "ru" = "en",
  init?: { signal?: AbortSignal },
): Promise<Category[]> {
  const { data, error } = await api.GET("/api/v1/categories", {
    params: { query: { locale } },
    signal: init?.signal,
  });
  if (error) throw new Error("Failed to load categories");
  return data?.data ?? [];
}
