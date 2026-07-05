"use client";

import Link from "next/link";
import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { ListingCard } from "@/features/listing/listing-card";
import { useFavorites } from "@/features/listing/use-favorites";
import { useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";

export default function FavoritesPage() {
  const t = useT();
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const { data, isLoading, error } = useFavorites();

  const isLoading$ = authLoading || isLoading;
  const hasError = error && !authLoading;
  const items = data?.data ?? [];
  const isEmpty = !isLoading$ && items.length === 0;

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <div className="container-avi py-6 md:py-8">
          <div className="mb-6">
            <h1 className="text-h1 text-text-primary">{t.header.favorites}</h1>
            <p className="mt-2 text-body text-text-secondary">{t.favorites.subtitle}</p>
          </div>

          {!isAuthenticated && !authLoading ? (
            <div className="flex min-h-[360px] items-center justify-center rounded-lg border border-border bg-surface-soft p-8 text-center">
              <div>
                <p className="mb-4 text-body text-text-secondary">{t.states.loginPrompt}</p>
                <Link
                  href="/login"
                  className="inline-flex h-10 items-center justify-center rounded-lg bg-primary px-6 text-sm font-medium text-white hover:bg-primary/90"
                >
                  {t.auth.logIn}
                </Link>
              </div>
            </div>
          ) : isLoading$ ? (
            <div className="grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {Array.from({ length: 8 }).map((_, i) => (
                <div
                  // biome-ignore lint/suspicious/noArrayIndexKey: Static skeleton loaders, order never changes
                  key={i}
                  className="aspect-[1.58/1] animate-pulse rounded-[14px] bg-surface-soft"
                />
              ))}
            </div>
          ) : hasError ? (
            <div className="flex min-h-[240px] items-center justify-center rounded-lg border border-border bg-surface-soft p-8 text-center">
              <div>
                <p className="mb-4 text-body text-text-secondary">{t.states.error}</p>
                <button
                  type="button"
                  onClick={() => window.location.reload()}
                  className="inline-flex h-10 items-center justify-center rounded-lg bg-primary px-6 text-sm font-medium text-white hover:bg-primary/90"
                >
                  {t.common.retry}
                </button>
              </div>
            </div>
          ) : isEmpty ? (
            <div className="flex min-h-[240px] items-center justify-center">
              <p className="text-body text-text-muted">{t.states.empty}</p>
            </div>
          ) : (
            <div className="grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {items.map((item) => (
                <ListingCard key={item.id} item={item} />
              ))}
            </div>
          )}
        </div>
      </main>

      <BottomNav />
    </>
  );
}
