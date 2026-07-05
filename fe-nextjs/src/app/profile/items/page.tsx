"use client";

import { Edit3, Eye, PackageOpen, Plus } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { useMyListings } from "@/features/listing/use-my-listings";
import { useLocale, useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { cityLabel, formatPrice, formatRelativeTime, pickListingImage } from "@/lib/format";

const skeletonRows = ["first", "second", "third"];

export default function MyListingsPage() {
  const t = useT();
  const locale = useLocale();
  const { user, isAuthenticated, isLoading } = useAuth();
  const { data, isLoading: isListingsLoading, isError } = useMyListings(user?.id);
  const items = data?.data ?? [];

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <div className="container-avi py-6 md:py-8">
          <div className="mb-6 flex flex-col justify-between gap-4 md:mb-8 md:flex-row md:items-end">
            <div className="max-w-3xl">
              <p className="text-meta font-semibold uppercase text-primary">
                {t.myListings.eyebrow}
              </p>
              <h1 className="mt-2 text-h1 text-text-primary">{t.myListings.title}</h1>
              <p className="mt-3 text-body text-text-secondary">{t.myListings.subtitle}</p>
            </div>
            <Link href="/items/new">
              <Button className="h-11 gap-2 rounded-lg px-5 font-bold">
                <Plus className="size-4" />
                {t.header.postListing}
              </Button>
            </Link>
          </div>

          {!isAuthenticated && !isLoading ? (
            <LoginPrompt />
          ) : isLoading || isListingsLoading ? (
            <div className="grid gap-4">
              {skeletonRows.map((key) => (
                <div
                  key={key}
                  className="h-[156px] animate-pulse rounded-lg border border-border bg-surface-soft"
                />
              ))}
            </div>
          ) : isError ? (
            <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-5 text-sm font-medium text-destructive">
              {t.states.error}
            </div>
          ) : items.length === 0 ? (
            <div className="flex min-h-[340px] items-center justify-center rounded-lg border border-border bg-white/88 p-8 text-center shadow-card">
              <div className="max-w-sm">
                <div className="mx-auto flex size-12 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <PackageOpen className="size-6" />
                </div>
                <h2 className="mt-4 text-section text-text-primary">{t.myListings.empty_title}</h2>
                <p className="mt-2 text-body text-text-secondary">{t.myListings.empty_subtitle}</p>
                <Link href="/items/new" className="mt-5 inline-block">
                  <Button className="h-10 rounded-lg px-5 font-bold">{t.header.postListing}</Button>
                </Link>
              </div>
            </div>
          ) : (
            <div className="grid gap-4">
              {items.map((item) => {
                const slug = item.slug || item.id || "";
                return (
                  <article
                    key={item.id}
                    className="grid gap-4 rounded-lg border border-border bg-white/94 p-4 shadow-card md:grid-cols-[180px_minmax(0,1fr)_auto] md:items-center"
                  >
                    <Link
                      href={`/items/${slug}`}
                      className="relative aspect-[4/3] overflow-hidden rounded-lg bg-surface-soft"
                    >
                      <Image
                        src={pickListingImage(item)}
                        alt={item.title ?? ""}
                        fill
                        className="object-cover"
                        sizes="180px"
                      />
                    </Link>

                    <div className="min-w-0">
                      <div className="mb-2 flex flex-wrap items-center gap-2">
                        <span className="rounded-md bg-primary/10 px-2 py-1 text-[11px] font-bold uppercase text-primary">
                          {statusLabel(item.status, t)}
                        </span>
                        <span className="text-meta text-text-muted">
                          {formatRelativeTime(item.created_at, locale)}
                        </span>
                      </div>
                      <Link href={`/items/${slug}`} className="group">
                        <h2 className="line-clamp-2 text-section text-text-primary group-hover:text-primary">
                          {item.title}
                        </h2>
                      </Link>
                      <p className="mt-2 text-price text-text-primary">
                        {formatPrice(item.price, locale, t.listing.priceOnRequest)}
                      </p>
                      <p className="mt-2 text-body text-text-secondary">
                        {cityLabel(item, locale) || "-"}
                      </p>
                    </div>

                    <div className="flex gap-2 md:flex-col">
                      <Link href={`/profile/items/${item.id}/edit`} className="flex-1 md:flex-none">
                        <Button variant="default" className="h-10 w-full gap-2 rounded-lg px-4">
                          <Edit3 className="size-4" />
                          {t.myListings.edit}
                        </Button>
                      </Link>
                      <Link href={`/items/${slug}`} className="flex-1 md:flex-none">
                        <Button variant="outline" className="h-10 w-full gap-2 rounded-lg px-4">
                          <Eye className="size-4" />
                          {t.myListings.view}
                        </Button>
                      </Link>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </div>
      </main>

      <BottomNav />
    </>
  );
}

function LoginPrompt() {
  const t = useT();

  return (
    <div className="flex min-h-[340px] items-center justify-center rounded-lg border border-border bg-white/88 p-8 text-center shadow-card">
      <div>
        <p className="mb-4 text-body text-text-secondary">{t.states.loginPrompt}</p>
        <Link href="/login">
          <Button>{t.auth.logIn}</Button>
        </Link>
      </div>
    </div>
  );
}

function statusLabel(status: string | undefined, t: ReturnType<typeof useT>) {
  switch (status) {
    case "draft":
      return t.myListings.status_draft;
    case "archived":
      return t.myListings.status_archived;
    case "sold":
      return t.myListings.status_sold;
    default:
      return t.myListings.status_published;
  }
}
