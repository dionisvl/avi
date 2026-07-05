import { api } from "@/lib/api/client";
import { getAccessToken } from "@/lib/auth/tokens";

/**
 * Request body for POST /api/v1/items.
 * All required fields must be provided.
 */
export interface CreateItemRequest {
  category_id: string;
  title: string;
  description?: string;
  condition?: "new" | "used";
  city_uuid?: string;
  photo_ids?: string[];
  thumbnail_id?: string;
  price?: {
    amount: number;
    currency: string;
  };
}

/**
 * Response type for POST /api/v1/items.
 */
export interface CreateItemResponse {
  id: string;
  slug?: string;
  category_id: string;
  title: string;
  description?: string;
  condition?: string;
  city_uuid?: string;
  price?: {
    amount: number;
    currency: string;
  };
}

/**
 * Create a new listing. Requires Bearer authentication (access token in cookie).
 * Throws an error if the request fails.
 */
export async function createItem(req: CreateItemRequest): Promise<CreateItemResponse> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("No access token available");
  }

  const { data, error } = await api.POST("/api/v1/items", {
    body: req,
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (error) {
    throw new Error(`Failed to create listing: ${error.toString()}`);
  }

  if (!data?.data) {
    throw new Error("Invalid response from server");
  }

  return data.data as CreateItemResponse;
}
