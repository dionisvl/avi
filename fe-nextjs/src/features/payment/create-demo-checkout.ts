import { API_BASE_URL } from "@/lib/api/client";

export interface DemoCheckoutPayment {
  id?: string;
  status?: string;
  confirmation_url?: string;
  amount?: {
    value?: string;
    currency?: string;
  };
}

export async function createDemoCheckoutPayment({
  itemId,
  accessToken,
  returnUrl,
  signal,
}: {
  itemId: string;
  accessToken: string;
  returnUrl: string;
  signal?: AbortSignal;
}): Promise<DemoCheckoutPayment> {
  const response = await fetch(`${API_BASE_URL}/api/v1/payments`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${accessToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      purpose: "demo_checkout",
      subject_id: itemId,
      return_url: returnUrl,
    }),
    signal,
  });

  if (!response.ok) {
    throw new Error("Failed to create demo checkout");
  }

  const payment = (await response.json()) as DemoCheckoutPayment;
  if (!payment.confirmation_url) {
    throw new Error("Payment confirmation URL is missing");
  }

  return payment;
}
