"use client";

import { useMutation } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/auth/tokens";
import { createDemoCheckoutPayment } from "./create-demo-checkout";

export function useDemoCheckout() {
  return useMutation({
    mutationFn: async ({ itemId, returnUrl }: { itemId: string; returnUrl: string }) => {
      const accessToken = getAccessToken();
      if (!accessToken) {
        throw new Error("Not authenticated");
      }
      return createDemoCheckoutPayment({ itemId, accessToken, returnUrl });
    },
  });
}
