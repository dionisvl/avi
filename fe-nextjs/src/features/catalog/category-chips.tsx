"use client";

import {
  Briefcase,
  Building2,
  Car,
  ChevronDown,
  PawPrint,
  Shirt,
  Smartphone,
  Sofa,
  Sparkles,
  Tag,
} from "lucide-react";
import Link from "next/link";
import { Skeleton } from "@/components/ui/skeleton";
import { useLocale, useT } from "@/i18n";
import type { Category } from "@/lib/api/types";
import { useCategories } from "./use-catalog";

/** Map category slug (as served by the API) to a lucide icon. */
const SLUG_TO_ICON: Record<string, React.ComponentType<{ className?: string; size: number }>> = {
  electronics: Smartphone,
  transport: Car,
  "real-estate": Building2,
  clothing: Shirt,
  "home-garden": Sofa,
  hobbies: Sparkles,
  jobs: Briefcase,
  animals: PawPrint,
};

/** Get icon for a category slug; fallback to Tag if not found. */
function getIconForSlug(slug?: string): React.ComponentType<{ className?: string; size: number }> {
  if (!slug) return Tag;
  return SLUG_TO_ICON[slug] ?? Tag;
}

interface CategoryChipProps {
  category: Category;
}

/** A single category pill chip. */
function CategoryChip({ category }: CategoryChipProps) {
  const Icon = getIconForSlug(category.slug);
  const href = category.slug ? `/items?category=${category.slug}` : "#";

  return (
    <Link href={href}>
      <div className="inline-flex h-[58px] min-w-[128px] flex-shrink-0 items-center justify-center gap-3 whitespace-nowrap rounded-[16px] border border-border bg-white/88 px-5 text-[14px] font-bold text-text-primary shadow-[0_10px_26px_rgba(50,46,90,0.04)] transition-all hover:-translate-y-0.5 hover:border-primary hover:text-primary hover:shadow-card">
        <Icon size={22} className="text-primary" />
        <span>{category.name}</span>
      </div>
    </Link>
  );
}

/** "More" chip with chevron icon. */
function MoreChip() {
  const t = useT();

  return (
    <div className="inline-flex h-[58px] min-w-[96px] flex-shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-[16px] border border-border bg-white/88 px-5 text-[14px] font-bold text-text-primary shadow-[0_10px_26px_rgba(50,46,90,0.04)] transition-all hover:border-primary hover:text-primary">
      <span>{t.common.more}</span>
      <ChevronDown size={16} />
    </div>
  );
}

/** Skeleton loader for category chips. */
function CategoryChipSkeleton() {
  return <Skeleton className="h-[58px] w-40 flex-shrink-0 rounded-[16px]" />;
}

/** Horizontal row of pill chips from useCategories(). */
export function CategoryChips() {
  const locale = useLocale();
  const { data: categories = [], isLoading } = useCategories(locale);

  // Show 6 skeleton pills while loading
  if (isLoading) {
    return (
      <div>
        <div className="flex snap-x snap-mandatory gap-4 overflow-x-auto scrollbar-hide 2xl:overflow-visible">
          {Array.from({ length: 6 }, (_, i) => `chip-${i}`).map((key) => (
            <CategoryChipSkeleton key={key} />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex snap-x snap-mandatory gap-4 overflow-x-auto scrollbar-hide 2xl:overflow-visible">
        {categories.map((category) => (
          <CategoryChip key={category.id} category={category} />
        ))}
        <MoreChip />
      </div>
    </div>
  );
}
