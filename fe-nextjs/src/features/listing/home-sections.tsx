"use client";

import { useT } from "@/i18n";
import { ListingSection } from "./listing-section";
import { useListings } from "./use-listings";

export default function HomeSections() {
  const t = useT();

  // Recommendations: higher-price listings first.
  const rec = useListings({ per_page: 6, sort: "-price" });

  // New listings: newest listings first.
  const fresh = useListings({ per_page: 8, sort: "-created_at" });

  return (
    <div className="space-y-8">
      {/* Recommendations section */}
      <ListingSection
        title={t.sections.recommendations}
        seeAllHref="/items"
        items={rec.data?.data ?? []}
        isLoading={rec.isLoading}
        isError={rec.isError}
        onRetry={rec.refetch}
        columns={{ base: 2, md: 3, lg: 6 }}
      />

      {/* New listings section */}
      <ListingSection
        title={t.sections.newListings}
        seeAllHref="/items"
        items={fresh.data?.data ?? []}
        isLoading={fresh.isLoading}
        isError={fresh.isError}
        onRetry={fresh.refetch}
        columns={{ base: 2, md: 4, lg: 8 }}
        framed
      />
    </div>
  );
}
