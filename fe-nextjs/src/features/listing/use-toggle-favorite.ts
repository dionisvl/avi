"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { addFavorite, removeFavorite } from "./favorites";

/**
 * Toggle a listing's favorite state.
 *
 * The heart button tracks its own optimistic state, so we deliberately do NOT
 * invalidate `["items"]` here — that refetched the whole catalog on every click,
 * remounting cards and desyncing items shown in more than one homepage section.
 * Only the favorites *page* list is refreshed.
 */
export function useToggleFavorite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ itemId, favorited }: { itemId: string; favorited: boolean }) =>
      favorited ? removeFavorite(itemId) : addFavorite(itemId),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["favorites"] });
    },
  });
}
