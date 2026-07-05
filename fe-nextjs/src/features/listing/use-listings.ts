"use client";

import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { getListings, type ItemsPage, type ListingsQuery } from "./get-listings";

/** One page of listings (non-paginated fetch), e.g. Recommendations / New listings. */
export function useListings(query: ListingsQuery = {}) {
  return useQuery({
    queryKey: ["items", query],
    queryFn: ({ signal }) => getListings(query, { signal }),
  });
}

/** Infinite listings feed for the homepage bottom scroll. */
export function useInfiniteListings(query: ListingsQuery = {}, perPage = 12) {
  return useInfiniteQuery({
    queryKey: ["items", "infinite", query, perPage],
    initialPageParam: 1,
    queryFn: ({ pageParam, signal }) =>
      getListings({ ...query, page: pageParam, per_page: perPage }, { signal }),
    getNextPageParam: (last: ItemsPage) => {
      const page = last.pagination.page ?? 1;
      const totalPages = last.pagination.total_pages ?? 1;
      return page < totalPages ? page + 1 : undefined;
    },
  });
}
