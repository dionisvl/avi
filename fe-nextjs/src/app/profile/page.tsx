"use client";

import { Package, Plus } from "lucide-react";
import Link from "next/link";
import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { useLocale, useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";

export default function ProfilePage() {
  const t = useT();
  const locale = useLocale();
  const { user, isAuthenticated, isLoading } = useAuth();

  const displayName = user?.name || user?.email || "User";
  const email = user?.email || "";
  const createdAt = user?.created_at;
  const emailVerified = user?.email_verified ?? false;
  const roles = user?.roles ?? [];

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <div className="container-avi py-6 md:py-8">
          <div className="mb-8">
            <h1 className="text-h1 text-text-primary">{t.header.profile}</h1>
            <p className="mt-2 text-body text-text-secondary">{t.profile.subtitle}</p>
          </div>

          {!isAuthenticated && !isLoading ? (
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
          ) : isLoading ? (
            <div className="space-y-6">
              <div className="h-20 animate-pulse rounded-lg bg-surface-soft" />
              <div className="h-20 animate-pulse rounded-lg bg-surface-soft" />
              <div className="h-20 animate-pulse rounded-lg bg-surface-soft" />
            </div>
          ) : (
            <div className="max-w-2xl">
              <div className="rounded-lg border border-border bg-surface p-6 space-y-6">
                <div>
                  <div className="text-sm font-medium text-text-secondary">{t.profile.name}</div>
                  <p className="mt-1 text-body text-text-primary">{displayName}</p>
                </div>

                <div>
                  <div className="text-sm font-medium text-text-secondary">{t.profile.email}</div>
                  <p className="mt-1 text-body text-text-primary">{email}</p>
                  {emailVerified ? (
                    <p className="mt-2 text-xs text-green-600">✓ {t.profile.emailVerified}</p>
                  ) : (
                    <p className="mt-2 text-xs text-text-muted">{t.profile.emailNotVerified}</p>
                  )}
                </div>

                {createdAt && (
                  <div>
                    <div className="text-sm font-medium text-text-secondary">
                      {t.profile.memberSince}
                    </div>
                    <p className="mt-1 text-body text-text-primary">
                      {new Date(createdAt).toLocaleDateString(locale === "ru" ? "ru-RU" : "en-US", {
                        year: "numeric",
                        month: "long",
                        day: "numeric",
                      })}
                    </p>
                  </div>
                )}

                {roles && roles.length > 0 && (
                  <div>
                    <div className="text-sm font-medium text-text-secondary">{t.profile.roles}</div>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {roles.map((role) => (
                        <span
                          key={role}
                          className="inline-flex items-center rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary"
                        >
                          {role}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                <div className="grid gap-3 border-t border-border pt-6 sm:grid-cols-2">
                  <Link
                    href="/profile/items"
                    className="flex items-center gap-3 rounded-lg border border-border bg-surface-soft p-4 transition-colors hover:border-primary/30 hover:bg-primary/5"
                  >
                    <div className="flex size-10 items-center justify-center rounded-lg bg-white text-primary shadow-card">
                      <Package className="size-5" />
                    </div>
                    <div>
                      <div className="text-sm font-bold text-text-primary">
                        {t.header.myListings}
                      </div>
                      <p className="mt-1 text-meta text-text-secondary">
                        {t.myListings.profile_hint}
                      </p>
                    </div>
                  </Link>
                  <Link
                    href="/items/new"
                    className="flex items-center gap-3 rounded-lg border border-border bg-surface-soft p-4 transition-colors hover:border-primary/30 hover:bg-primary/5"
                  >
                    <div className="flex size-10 items-center justify-center rounded-lg bg-white text-primary shadow-card">
                      <Plus className="size-5" />
                    </div>
                    <div>
                      <div className="text-sm font-bold text-text-primary">
                        {t.header.postListing}
                      </div>
                      <p className="mt-1 text-meta text-text-secondary">
                        {t.newListing.profile_hint}
                      </p>
                    </div>
                  </Link>
                </div>
              </div>
            </div>
          )}
        </div>
      </main>

      <BottomNav />
    </>
  );
}
