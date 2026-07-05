"use client";

import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useT } from "@/i18n";
import type { Item } from "@/lib/api/types";
import { cn } from "@/lib/utils";
import { ListingCard } from "./listing-card";
import { ListingCardSkeleton } from "./listing-card-skeleton";

interface ListingSectionProps {
  title: string;
  seeAllHref?: string;
  items: Item[];
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
  columns: {
    base: number;
    md: number;
    lg: number;
  };
  skeletonCount?: number;
  framed?: boolean;
}

// Full, static responsive class strings — Tailwind must see these literally.
const GRID_VARIANTS: Record<string, string> = {
  "2-3-6": "grid-cols-2 md:grid-cols-3 lg:grid-cols-6",
  "2-4-8": "grid-cols-2 md:grid-cols-4 lg:grid-cols-8",
};

/** Helper to build responsive grid classes from column config. */
function getGridClasses(columns: { base: number; md: number; lg: number }): string {
  const key = `${columns.base}-${columns.md}-${columns.lg}`;
  return GRID_VARIANTS[key] ?? "grid-cols-2 md:grid-cols-4 lg:grid-cols-6";
}

export function ListingSection({
  title,
  seeAllHref,
  items,
  isLoading = false,
  isError = false,
  onRetry,
  columns,
  skeletonCount,
  framed = false,
}: ListingSectionProps) {
  const t = useT();

  // Determine skeleton count: default to lg columns value
  const skeletonsToShow = skeletonCount ?? columns.lg;

  // Determine what to render in the grid
  let gridContent = null;

  if (isLoading) {
    // Show skeleton loaders
    gridContent = Array.from({ length: skeletonsToShow }, (_, i) => `skeleton-${i}`).map((key) => (
      <ListingCardSkeleton key={key} />
    ));
  } else if (isError) {
    // Show error state (centered, full width)
    return (
      <section>
        <div className="flex flex-col items-center justify-center gap-4 py-12">
          <p className="text-text-secondary">{t.states.error}</p>
          {onRetry && (
            <Button onClick={onRetry} variant="default">
              {t.common.retry}
            </Button>
          )}
        </div>
      </section>
    );
  } else if (items.length === 0) {
    // Show empty state (centered, full width)
    return (
      <section>
        <div className="flex items-center justify-center py-12">
          <p className="text-text-muted">{t.states.empty}</p>
        </div>
      </section>
    );
  } else {
    // Show items
    gridContent = items.map((item) => <ListingCard key={item.id} item={item} />);
  }

  return (
    <section
      className={cn(
        framed &&
          "rounded-container border border-border/80 bg-white/58 px-5 py-5 shadow-[0_18px_60px_rgba(54,49,104,0.06)] md:px-6",
      )}
    >
      <div className="mb-5 flex items-center justify-between">
        <h2 className="text-section text-text-primary">{title}</h2>
        {seeAllHref && (
          <Link href={seeAllHref}>
            <div className="flex items-center gap-1 text-[13px] font-bold text-primary transition-colors hover:text-primary/80">
              {t.common.seeAll}
              <ChevronRight size={16} />
            </div>
          </Link>
        )}
      </div>

      <div className={cn("grid gap-x-5 gap-y-6", getGridClasses(columns))}>{gridContent}</div>
    </section>
  );
}
