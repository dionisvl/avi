"use client";

import { Badge } from "@/components/ui/badge";
import { FavoriteHeartButton } from "@/features/listing/favorite-heart-button";
import { MessageSellerButton } from "@/features/listing/message-seller-button";
import { useLocale, useT } from "@/i18n";
import type { Item } from "@/lib/api/types";
import { cityLabel, formatPrice } from "@/lib/format";
import { DemoCheckoutAction } from "./demo-checkout-action";

interface ItemDetailClientProps {
  item: Item;
  locale: string;
}

export function ItemDetailClient({ item, locale }: ItemDetailClientProps) {
  const t = useT();
  const currentLocale = useLocale() || locale;
  const conditionLabel =
    item.condition === "new" ? t.newListing.condition_new : t.newListing.condition_used;

  return (
    <div className="rounded-[16px] border border-border bg-white/92 p-5 shadow-card md:p-6">
      <div className="flex flex-wrap items-center gap-2">
        {item.condition === "new" && <Badge>{t.listing.badgeNew}</Badge>}
        {item.condition === "used" && <Badge variant="secondary">{conditionLabel}</Badge>}
      </div>

      <div className="mt-4">
        <div className="flex items-start justify-between gap-4">
          <h1 className="min-w-0 text-[28px] font-bold leading-8 text-text-primary md:text-[34px] md:leading-10">
            {item.title ?? ""}
          </h1>
          {item.id && (
            <FavoriteHeartButton
              itemId={item.id}
              initialFavorited={item.is_favorited ?? false}
              syncWithFavorites
              className="mt-0.5 size-11 shrink-0 border border-border bg-white"
              iconClassName="size-5"
            />
          )}
        </div>
        <p className="mt-4 text-[30px] font-bold leading-9 text-text-primary">
          {formatPrice(item.price, currentLocale, t.listing.priceOnRequest)}
        </p>
      </div>

      {item.id && (
        <div className="mt-5">
          <DemoCheckoutAction itemId={item.id} itemSlug={item.slug || item.id} />
        </div>
      )}

      <div className="mt-3">
        <MessageSellerButton item={item} />
      </div>

      <dl className="mt-6 divide-y divide-border rounded-lg border border-border bg-surface-soft px-4 text-body">
        <div className="grid grid-cols-[112px_minmax(0,1fr)] gap-4 py-3">
          <dt className="text-text-muted">{t.listing.city}</dt>
          <dd className="min-w-0 font-medium text-text-primary">
            {cityLabel(item, currentLocale) || "-"}
          </dd>
        </div>
        <div className="grid grid-cols-[112px_minmax(0,1fr)] gap-4 py-3">
          <dt className="text-text-muted">{t.listing.seller}</dt>
          <dd className="min-w-0 font-medium text-text-primary">
            {item.seller?.name || t.listing.privateSeller}
          </dd>
        </div>
        {item.condition && (
          <div className="grid grid-cols-[112px_minmax(0,1fr)] gap-4 py-3">
            <dt className="text-text-muted">{t.newListing.condition_label}</dt>
            <dd className="min-w-0 font-medium text-text-primary">{conditionLabel}</dd>
          </div>
        )}
      </dl>

      <section className="mt-6">
        <h2 className="text-section text-text-primary">{t.listing.description}</h2>
        <p className="mt-3 whitespace-pre-wrap text-body leading-7 text-text-secondary">
          {item.description || t.listing.noDescription}
        </p>
      </section>
    </div>
  );
}
