"use client";

import { useQuery } from "@tanstack/react-query";
import { getMyListings } from "./get-my-listings";

export function useMyListings(sellerId?: string) {
  return useQuery({
    queryKey: ["my-items", sellerId],
    queryFn: ({ signal }) => getMyListings(sellerId ?? "", { signal }),
    enabled: !!sellerId,
  });
}
