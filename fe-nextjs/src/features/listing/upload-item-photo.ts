import { API_BASE_URL } from "@/lib/api/client";
import { getAccessToken } from "@/lib/auth/tokens";

export interface UploadedItemPhoto {
  id: string;
  url: string;
  thumbnail_url: string;
}

export async function uploadItemPhoto(file: File, itemId?: string): Promise<UploadedItemPhoto> {
  const token = getAccessToken();
  if (!token) {
    throw new Error("No access token available");
  }

  const form = new FormData();
  form.append("type", "item");
  if (itemId) {
    form.append("item_id", itemId);
  }
  form.append("file", file);

  const response = await fetch(`${API_BASE_URL}/api/v1/upload`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
    },
    body: form,
  });

  if (!response.ok) {
    throw new Error("Failed to upload photo");
  }

  const payload = (await response.json()) as { data?: UploadedItemPhoto };
  if (!payload.data?.id || !payload.data.url) {
    throw new Error("Invalid upload response");
  }

  return payload.data;
}
