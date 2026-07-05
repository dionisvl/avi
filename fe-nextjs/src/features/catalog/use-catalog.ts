"use client";

import { useQuery } from "@tanstack/react-query";
import { getCategories } from "./get-categories";
import { getCities } from "./get-cities";

/** All categories for chips / navigation. */
export function useCategories(locale: "en" | "ru" = "en") {
  return useQuery({
    queryKey: ["categories", locale],
    queryFn: ({ signal }) => getCategories(locale, { signal }),
  });
}

/** All cities for the location selector. */
export function useCities() {
  return useQuery({
    queryKey: ["cities"],
    queryFn: ({ signal }) => getCities({ signal }),
  });
}
