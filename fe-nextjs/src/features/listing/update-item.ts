import { api } from "@/lib/api/client";
import type { Item } from "@/lib/api/types";
import { getAccessToken } from "@/lib/auth/tokens";

export interface UpdateItemRequest {
  category_id?: string;
  title?: string;
  description?: string;
  condition?: "new" | "used";
  city_uuid?: string;
  photo_ids?: string[];
  thumbnail_id?: string | null;
  price?: {
    amount: number;
    currency: string;
  };
  status?: "published" | "draft" | "archived" | "sold";
}

export async function updateItem(id: string, req: UpdateItemRequest): Promise<Item> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("No access token available");
  }

  const { data, error } = await api.PATCH("/api/v1/items/{id}", {
    params: { path: { id } },
    body: req as never,
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (error) {
    throw new Error(error.detail || error.title || "Failed to update listing");
  }

  if (!data?.data) {
    throw new Error("Invalid response from server");
  }

  return data.data;
}
