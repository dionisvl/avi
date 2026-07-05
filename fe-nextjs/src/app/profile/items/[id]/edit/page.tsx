"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { CreateListingForm } from "@/app/items/new/create-listing-form";
import { Button } from "@/components/ui/button";
import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { getOwnedItem } from "@/features/listing/get-owned-item";
import { useLocale, useT } from "@/i18n";
import type { Item } from "@/lib/api/types";
import { useAuth } from "@/lib/auth/context";

export default function EditListingPage() {
  const t = useT();
  const locale = useLocale();
  const params = useParams<{ id: string }>();
  const { isAuthenticated, isLoading } = useAuth();
  const [item, setItem] = useState<Item | null>(null);
  const [isItemLoading, setIsItemLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!isAuthenticated || !params.id) {
      setIsItemLoading(false);
      return;
    }

    const controller = new AbortController();
    setIsItemLoading(true);
    getOwnedItem(params.id, { signal: controller.signal })
      .then((data) => {
        setItem(data);
        if (!data) {
          setError(t.myListings.not_found);
        }
      })
      .catch((err: unknown) => {
        if (err instanceof Error && err.name === "AbortError") return;
        setError(t.states.error);
      })
      .finally(() => {
        setIsItemLoading(false);
      });

    return () => controller.abort();
  }, [isAuthenticated, params.id, t.myListings.not_found, t.states.error]);

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <div className="container-avi py-6 md:py-8">
          <div className="mb-6 max-w-3xl md:mb-8">
            <p className="text-meta font-semibold uppercase text-primary">
              {t.myListings.edit_eyebrow}
            </p>
            <h1 className="mt-2 text-h1 text-text-primary">{t.myListings.edit_title}</h1>
            <p className="mt-3 text-body text-text-secondary">{t.myListings.edit_subtitle}</p>
          </div>

          {!isAuthenticated && !isLoading ? (
            <div className="flex min-h-[340px] items-center justify-center rounded-lg border border-border bg-white/88 p-8 text-center shadow-card">
              <div>
                <p className="mb-4 text-body text-text-secondary">{t.states.loginPrompt}</p>
                <Link href="/login">
                  <Button>{t.auth.logIn}</Button>
                </Link>
              </div>
            </div>
          ) : isLoading || isItemLoading ? (
            <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_380px]">
              <div className="h-[520px] animate-pulse rounded-lg bg-surface-soft" />
              <div className="h-[360px] animate-pulse rounded-lg bg-surface-soft" />
            </div>
          ) : error || !item ? (
            <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-5 text-sm font-medium text-destructive">
              {error || t.myListings.not_found}
            </div>
          ) : (
            <Suspense fallback={<p>{t.common.loading}</p>}>
              <CreateListingForm locale={locale} item={item} mode="edit" />
            </Suspense>
          )}
        </div>
      </main>

      <BottomNav />
    </>
  );
}
