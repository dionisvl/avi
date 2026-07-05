"use client";

import { MessageCircle } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useT } from "@/i18n";
import type { Item } from "@/lib/api/types";
import { useAuth } from "@/lib/auth/context";
import { useOpenConversation } from "./use-open-conversation";

interface MessageSellerButtonProps {
  item: Item;
}

export function MessageSellerButton({ item }: MessageSellerButtonProps) {
  const t = useT();
  const { isAuthenticated, user } = useAuth();
  const router = useRouter();
  const mutation = useOpenConversation();

  // Can't message yourself
  const isSeller = user?.id === item.seller?.id;

  // Not authenticated: show login link
  if (!isAuthenticated) {
    return (
      <Link
        href="/login"
        className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-primary bg-white px-4 py-3 font-bold text-primary shadow-[0_8px_22px_rgba(32,28,64,0.08)] transition-colors hover:bg-primary/5"
      >
        <MessageCircle className="size-4" />
        {t.listing.messageSeller}
      </Link>
    );
  }

  if (isSeller) {
    return (
      <button
        type="button"
        disabled
        className="inline-flex w-full cursor-not-allowed items-center justify-center gap-2 rounded-lg border border-border bg-surface-soft px-4 py-3 font-bold text-text-muted"
      >
        <MessageCircle className="size-4" />
        {t.listing.ownListing}
      </button>
    );
  }

  const handleClick = async () => {
    if (!item.seller?.id) return;

    try {
      await mutation.mutateAsync({ peer_user_id: item.seller.id });
      router.push("/messages");
    } catch {
      // Error is handled by mutation state; user sees error message below
    }
  };

  return (
    <div className="space-y-2">
      <button
        type="button"
        onClick={handleClick}
        disabled={mutation.isPending}
        className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-primary bg-white px-4 py-3 font-bold text-primary shadow-[0_8px_22px_rgba(32,28,64,0.08)] transition-colors hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-50"
      >
        <MessageCircle className="size-4" />
        {mutation.isPending ? t.listing.messageSelling : t.listing.messageSeller}
      </button>
      {mutation.isError && (
        <p className="text-center text-sm text-text-error">{t.listing.messageError}</p>
      )}
    </div>
  );
}
