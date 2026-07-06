"use client";

import Image from "next/image";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { useLocale, useT } from "@/i18n";
import type { Item } from "@/lib/api/types";
import { cityLabel, formatPrice, formatRelativeTime, pickListingImage } from "@/lib/format";
import { FavoriteHeartButton } from "./favorite-heart-button";

interface ListingCardProps {
  item: Item;
}

export function ListingCard({ item }: ListingCardProps) {
  const t = useT();
  const locale = useLocale();

  const slug = item.slug || item.id || "";
  const imageUrl = pickListingImage(item);

  return (
    <Link href={`/items/${slug}`} className="group block h-full">
      <div className="flex h-full flex-col">
        <div className="relative aspect-[1.58/1] w-full overflow-hidden rounded-[14px] bg-surface-soft shadow-[0_8px_22px_rgba(43,39,78,0.06)]">
          <Image
            src={imageUrl}
            alt={item.title ?? ""}
            fill
            className="object-cover transition-transform duration-300 group-hover:scale-[1.025]"
            sizes="(max-width: 379px) 100vw, (max-width: 640px) 50vw, (max-width: 1024px) 33vw, 16vw"
          />

          {item.id && (
            <FavoriteHeartButton
              itemId={item.id}
              initialFavorited={item.is_favorited ?? false}
              className="absolute right-3 top-3 size-8"
              iconClassName="size-[18px]"
            />
          )}

          {item.condition === "new" && (
            <div className="absolute top-3 left-3">
              <Badge variant="default">{t.listing.badgeNew}</Badge>
            </div>
          )}
        </div>

        <div className="flex flex-1 flex-col gap-1.5 pt-3">
          <h3 className="line-clamp-2 text-card-title text-text-primary">{item.title ?? ""}</h3>

          <p className="text-price text-text-primary">
            {formatPrice(item.price, locale, t.listing.priceOnRequest)}
          </p>

          <p className="mt-1 text-[12px] font-medium leading-4 text-text-muted">
            {cityLabel(item, locale)}
            <br />
            {formatRelativeTime(item.created_at, locale)}
          </p>
        </div>
      </div>
    </Link>
  );
}
