import { Grid2X2, List } from "lucide-react";
import Link from "next/link";
import { BottomNav } from "@/features/catalog/bottom-nav";
import { getCategories } from "@/features/catalog/get-categories";
import { Header } from "@/features/catalog/header";
import { getListings } from "@/features/listing/get-listings";
import { ListingCard } from "@/features/listing/listing-card";
import type { Locale } from "@/i18n/config";
import { getRequestLocale } from "@/i18n/server";
import type { ItemsQuery } from "@/lib/api/types";
import { cn } from "@/lib/utils";

export const dynamic = "force-dynamic";

type SearchParams = Record<string, string | string[] | undefined>;

interface ItemsPageProps {
  searchParams: Promise<SearchParams>;
}

const copy = {
  en: {
    title: "Items",
    subtitle: "Browse local listings",
    resultsFor: "Results for",
    empty: "No listings found",
    gridView: "Grid view",
    listView: "List view",
  },
  ru: {
    title: "Объявления",
    subtitle: "Смотрите объявления рядом с вами",
    resultsFor: "Результаты по запросу",
    empty: "Объявления не найдены",
    gridView: "Вид сеткой",
    listView: "Вид списком",
  },
} as const;

function firstParam(value: string | string[] | undefined): string | undefined {
  if (Array.isArray(value)) return value[0];
  return value;
}

function nonEmptyParam(value: string | string[] | undefined): string | undefined {
  const first = firstParam(value)?.trim();
  return first || undefined;
}

interface CategoryFilter {
  categoryId?: string;
  categoryName?: string;
}

async function resolveCategoryFilter(
  params: SearchParams,
  locale: Locale,
): Promise<CategoryFilter> {
  const categorySlug = nonEmptyParam(params.category);
  const categoryId = nonEmptyParam(params.category_id);

  if (!categorySlug && !categoryId) {
    return {};
  }

  try {
    const categories = await getCategories(locale);
    const found = categories.find((cat) =>
      categorySlug ? cat.slug === categorySlug : cat.id === categoryId,
    );

    return {
      categoryId: found?.id ?? categoryId,
      categoryName: found?.name,
    };
  } catch {
    return categoryId ? { categoryId } : {};
  }
}

function buildQuery(params: SearchParams, categoryId?: string): ItemsQuery {
  const search = nonEmptyParam(params.search);
  const cityUuid = nonEmptyParam(params.city_uuid);

  return {
    page: 1,
    per_page: 24,
    ...(search ? { search } : {}),
    ...(categoryId ? { category_id: categoryId } : {}),
    ...(cityUuid ? { city_uuid: cityUuid } : {}),
    statuses: "published",
  };
}

function buildViewHref(params: SearchParams, view: "grid" | "list"): string {
  const urlParams = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    const first = firstParam(value);
    if (first && key !== "view") urlParams.set(key, first);
  }

  urlParams.set("view", view);
  return `/items?${urlParams.toString()}`;
}

export default async function ItemsPage({ searchParams }: ItemsPageProps) {
  const params = await searchParams;
  const locale = await getRequestLocale();
  const t = copy[locale];
  const search = nonEmptyParam(params.search);
  const view = nonEmptyParam(params.view) === "list" ? "list" : "grid";
  const categoryFilter = await resolveCategoryFilter(params, locale);
  const query = buildQuery(params, categoryFilter.categoryId);
  const { data: items } = await getListings(query);
  const pageTitle = categoryFilter.categoryName ?? t.title;

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <div className="container-avi py-6 md:py-8">
          <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h1 className="text-h1 text-text-primary">{pageTitle}</h1>
              <p className="mt-2 text-body text-text-secondary">
                {search ? `${t.resultsFor} "${search}"` : t.subtitle}
              </p>
            </div>

            <div className="flex items-center gap-2">
              <Link
                href={buildViewHref(params, "grid")}
                aria-label={t.gridView}
                className={cn(
                  "inline-flex size-9 items-center justify-center rounded-lg border border-border bg-surface text-text-secondary transition-colors hover:text-primary",
                  view === "grid" && "border-primary text-primary",
                )}
              >
                <Grid2X2 size={18} />
              </Link>
              <Link
                href={buildViewHref(params, "list")}
                aria-label={t.listView}
                className={cn(
                  "inline-flex size-9 items-center justify-center rounded-lg border border-border bg-surface text-text-secondary transition-colors hover:text-primary",
                  view === "list" && "border-primary text-primary",
                )}
              >
                <List size={18} />
              </Link>
            </div>
          </div>

          {items.length > 0 ? (
            <div
              className={cn(
                "grid gap-4",
                view === "list"
                  ? "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3"
                  : "grid-cols-2 md:grid-cols-3 lg:grid-cols-4",
              )}
            >
              {items.map((item) => (
                <ListingCard key={item.id} item={item} />
              ))}
            </div>
          ) : (
            <div className="flex min-h-[240px] items-center justify-center">
              <p className="text-body text-text-muted">{t.empty}</p>
            </div>
          )}
        </div>
      </main>

      <BottomNav />
    </>
  );
}
