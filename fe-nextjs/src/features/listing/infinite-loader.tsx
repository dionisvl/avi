"use client";

import { Loader2 } from "lucide-react";
import { useEffect, useRef } from "react";
import { useT } from "@/i18n";
import { ListingCard } from "./listing-card";
import { ListingCardSkeleton } from "./listing-card-skeleton";
import { useInfiniteListings } from "./use-listings";

export default function InfiniteLoader() {
  const t = useT();
  const sentinelRef = useRef<HTMLDivElement>(null);

  const { data, isLoading, isError, hasNextPage, isFetchingNextPage, fetchNextPage } =
    useInfiniteListings({ sort: "price" }, 12);

  // Flatten all pages' data into a single array
  const items = data?.pages?.flatMap((page) => page.data) ?? [];

  // IntersectionObserver for infinite scroll
  useEffect(() => {
    if (!sentinelRef.current) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && hasNextPage && !isFetchingNextPage && !isLoading) {
            fetchNextPage();
          }
        });
      },
      { threshold: 0.1 },
    );

    observer.observe(sentinelRef.current);

    return () => {
      if (sentinelRef.current) {
        observer.unobserve(sentinelRef.current);
      }
    };
  }, [hasNextPage, isFetchingNextPage, isLoading, fetchNextPage]);

  // Initial load state
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-x-5 gap-y-6 md:grid-cols-4 lg:grid-cols-8">
        {Array.from({ length: 8 }, (_, idx) => `init-${idx}`).map((key) => (
          <ListingCardSkeleton key={key} />
        ))}
      </div>
    );
  }

  // Error state
  if (isError) {
    return (
      <div className="flex justify-center items-center py-12">
        <p className="text-body text-text-secondary">{t.states.error}</p>
      </div>
    );
  }

  // Main content grid
  return (
    <div>
      <div className="grid grid-cols-2 gap-x-5 gap-y-6 md:grid-cols-4 lg:grid-cols-8">
        {items.map((item) => (
          <ListingCard key={item.id} item={item} />
        ))}
      </div>

      {/* Loading next page state */}
      {isFetchingNextPage && (
        <div className="mt-10">
          <div className="mb-6 flex items-center justify-center gap-4">
            <Loader2 className="size-9 animate-spin text-primary" />
            <div>
              <p className="text-[17px] font-extrabold text-text-primary">{t.infinite.loading}</p>
              <p className="text-meta text-text-muted">{t.infinite.more}</p>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-x-5 gap-y-6 md:grid-cols-4 lg:grid-cols-8">
            {Array.from({ length: 4 }, (_, idx) => `next-${idx}`).map((key) => (
              <ListingCardSkeleton key={key} />
            ))}
          </div>
        </div>
      )}

      {/* End state */}
      {!hasNextPage && items.length > 0 && !isFetchingNextPage && (
        <div className="flex items-center justify-center py-12">
          <p className="text-meta text-text-muted">{t.infinite.end}</p>
        </div>
      )}

      {/* Sentinel for intersection observer */}
      <div ref={sentinelRef} />
    </div>
  );
}
