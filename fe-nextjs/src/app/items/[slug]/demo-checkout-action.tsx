"use client";

import { DemoCheckoutButton } from "@/features/payment/demo-checkout-button";

export function DemoCheckoutAction({ itemId, itemSlug }: { itemId: string; itemSlug: string }) {
  return (
    <DemoCheckoutButton
      itemId={itemId}
      itemSlug={itemSlug}
      className="h-11 w-full rounded-lg text-[14px] font-bold"
    />
  );
}
