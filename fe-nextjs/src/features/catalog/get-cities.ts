import { api } from "@/lib/api/client";
import type { City } from "@/lib/api/types";

/** Fetch the list of active cities for the location selector. */
export async function getCities(init?: { signal?: AbortSignal }): Promise<City[]> {
  const { data, error } = await api.GET("/api/v1/cities", { signal: init?.signal });
  if (error) throw new Error("Failed to load cities");
  return data?.data ?? [];
}
