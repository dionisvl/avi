"use client";

import { useMutation } from "@tanstack/react-query";
import { type CreateItemRequest, type CreateItemResponse, createItem } from "./create-item";

export type { CreateItemRequest, CreateItemResponse };

/**
 * Mutation hook for creating a new listing.
 * Usage: const mutation = useCreateItem(); mutation.mutate({ category_id, title, ... });
 */
export function useCreateItem() {
  return useMutation<CreateItemResponse, Error, CreateItemRequest>({
    mutationFn: createItem,
  });
}
