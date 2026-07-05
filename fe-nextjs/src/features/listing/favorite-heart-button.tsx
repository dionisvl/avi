"use client";

import { useQuery } from "@tanstack/react-query";
import { Heart } from "lucide-react";
import { useRouter } from "next/navigation";
import { type MouseEvent, useEffect, useState } from "react";
import { getFavorites } from "@/features/listing/favorites";
import { useToggleFavorite } from "@/features/listing/use-toggle-favorite";
import { useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { getAccessToken } from "@/lib/auth/tokens";
import { cn } from "@/lib/utils";

interface FavoriteHeartButtonProps {
  itemId: string;
  initialFavorited: boolean;
  className?: string;
  iconClassName?: string;
  syncWithFavorites?: boolean;
}

export function FavoriteHeartButton({
  itemId,
  initialFavorited,
  className,
  iconClassName,
  syncWithFavorites = false,
}: FavoriteHeartButtonProps) {
  const t = useT();
  const router = useRouter();
  const { isAuthenticated, isLoading: isAuthLoading } = useAuth();
  const toggleFavorite = useToggleFavorite();
  const [isFavorited, setIsFavorited] = useState(initialFavorited);
  const hasStoredAccessToken = typeof document !== "undefined" && Boolean(getAccessToken());

  const favorites = useQuery({
    queryKey: ["favorites"],
    queryFn: ({ signal }) => getFavorites({ signal }),
    enabled: isAuthenticated && syncWithFavorites,
    retry: false,
  });

  useEffect(() => {
    setIsFavorited(initialFavorited);
  }, [initialFavorited]);

  useEffect(() => {
    if (!favorites.data) return;
    setIsFavorited(favorites.data.some((item) => item.id === itemId));
  }, [favorites.data, itemId]);

  const handleClick = async (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();

    if (toggleFavorite.isPending) return;
    if (!isAuthenticated && !hasStoredAccessToken) {
      router.push("/login");
      return;
    }

    const previousFavorited = isFavorited;

    // Flip the heart only after the server confirms; the button shows a pending
    // state via `toggleFavorite.isPending` while the request is in flight.
    try {
      await toggleFavorite.mutateAsync({
        itemId,
        favorited: previousFavorited,
      });
      setIsFavorited(!previousFavorited);
    } catch {
      // Leave the heart as it was; nothing changed on the server.
    }
  };

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={(isAuthLoading && !hasStoredAccessToken) || toggleFavorite.isPending}
      aria-label={isFavorited ? t.listing.removeFavorite : t.listing.addFavorite}
      aria-pressed={isFavorited}
      className={cn(
        "flex size-9 items-center justify-center rounded-full bg-white/92 text-text-muted shadow-[0_8px_18px_rgba(32,28,64,0.12)] transition-all hover:scale-105 hover:text-primary disabled:cursor-not-allowed disabled:opacity-60",
        isFavorited && "text-primary",
        className,
      )}
    >
      <Heart
        className={cn(
          "size-[18px] transition-colors",
          isFavorited && "fill-primary",
          iconClassName,
        )}
      />
    </button>
  );
}
