"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { type UpdateItemRequest, updateItem } from "./update-item";

export function useUpdateItem(itemId: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (req: UpdateItemRequest) => updateItem(itemId, req),
    onSuccess: (item) => {
      qc.invalidateQueries({ queryKey: ["items"] });
      qc.invalidateQueries({ queryKey: ["my-items"] });
      if (item.id) {
        qc.invalidateQueries({ queryKey: ["item", item.id] });
      }
    },
  });
}
