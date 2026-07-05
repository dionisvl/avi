"use client";

import Link from "next/link";
import { Suspense } from "react";
import { Button } from "@/components/ui/button";
import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { useLocale, useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { CreateListingForm } from "./create-listing-form";

export default function NewItemPage() {
  const t = useT();
  const locale = useLocale();
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <>
        <Header />
        <main className="flex-1 pb-24 lg:pb-16">
          <div className="container-avi py-8">
            <p className="text-body text-text-secondary">{t.common.loading}</p>
          </div>
        </main>
        <BottomNav />
      </>
    );
  }

  if (!isAuthenticated) {
    return (
      <>
        <Header />
        <main className="flex-1 pb-24 lg:pb-16">
          <section className="container-avi flex min-h-[360px] flex-col justify-center py-8">
            <h1 className="text-h1 text-text-primary">{t.newListing.title}</h1>
            <p className="mt-6 max-w-xl text-body text-text-secondary">
              {t.newListing.loginPrompt}
            </p>
            <div className="mt-4">
              <Link href="/login">
                <Button variant="default">{t.auth.logIn}</Button>
              </Link>
            </div>
          </section>
        </main>
        <BottomNav />
      </>
    );
  }

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <div className="container-avi py-6 md:py-8">
          <div className="mb-6 max-w-3xl md:mb-8">
            <p className="text-meta font-semibold uppercase text-primary">
              {t.newListing.page_eyebrow}
            </p>
            <h1 className="mt-2 text-h1 text-text-primary">{t.newListing.title}</h1>
            <p className="mt-3 max-w-2xl text-body text-text-secondary">{t.newListing.subtitle}</p>
          </div>

          <Suspense fallback={<p>{t.common.loading}</p>}>
            <CreateListingForm locale={locale} />
          </Suspense>
        </div>
      </main>

      <BottomNav />
    </>
  );
}
