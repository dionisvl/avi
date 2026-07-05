"use client";

import { CreditCard, ExternalLink, X } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { useLocale, useT } from "@/i18n";
import { useAuth } from "@/lib/auth/context";
import { useDemoCheckout } from "./use-demo-checkout";

const YOOKASSA_TEST_CARDS_URL =
  "https://yookassa.ru/developers/payment-acceptance/testing-and-going-live/testing#test-bank-card";

export function DemoCheckoutButton({
  itemId,
  itemSlug,
  className,
}: {
  itemId: string;
  itemSlug: string;
  className?: string;
}) {
  const t = useT();
  const locale = useLocale();
  const router = useRouter();
  const { isAuthenticated, isLoading: isAuthLoading } = useAuth();
  const checkout = useDemoCheckout();
  const [isOpen, setIsOpen] = useState(false);

  const openDialog = (event: React.MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    if (isAuthLoading) return;
    if (!isAuthenticated) {
      router.push("/login");
      return;
    }
    setIsOpen(true);
  };

  const closeDialog = (event?: React.MouseEvent) => {
    event?.preventDefault();
    event?.stopPropagation();
    if (!checkout.isPending) {
      setIsOpen(false);
    }
  };

  const startCheckout = async (event: React.MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();

    const payment = await checkout.mutateAsync({
      itemId,
      returnUrl: `${window.location.origin}/items/${itemSlug}?payment=demo_checkout`,
    });

    window.location.assign(localizedConfirmationURL(payment.confirmation_url as string, locale));
  };

  return (
    <>
      <Button
        type="button"
        size="sm"
        className={className}
        onClick={openDialog}
        disabled={isAuthLoading || checkout.isPending}
      >
        <CreditCard className="size-3.5" />
        {t.demoPayment.buy}
      </Button>

      {isOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center px-4 py-6">
          <button
            type="button"
            className="absolute inset-0 bg-black/35"
            aria-label={t.demoPayment.close}
            onClick={closeDialog}
            disabled={checkout.isPending}
          />
          <div
            className="relative w-full max-w-[420px] rounded-lg border border-border bg-white p-5 shadow-[0_22px_70px_rgba(32,28,64,0.22)]"
            onClick={(event) => event.stopPropagation()}
            onKeyDown={(event) => {
              event.stopPropagation();
              if (event.key === "Escape") closeDialog();
            }}
            role="dialog"
            aria-modal="true"
            aria-labelledby="demo-checkout-title"
            tabIndex={-1}
          >
            <div className="mb-4 flex items-start justify-between gap-3">
              <div>
                <h2 id="demo-checkout-title" className="text-section text-text-primary">
                  {t.demoPayment.title}
                </h2>
                <p className="mt-1 text-sm text-text-secondary">{t.demoPayment.subtitle}</p>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t.demoPayment.close}
                onClick={closeDialog}
                disabled={checkout.isPending}
              >
                <X className="size-4" />
              </Button>
            </div>

            <div className="space-y-3 rounded-lg bg-surface-soft p-3 text-sm text-text-secondary">
              <p>{t.demoPayment.notice}</p>
              <p>{t.demoPayment.cardHint}</p>
              <Link
                href={YOOKASSA_TEST_CARDS_URL}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 font-semibold text-primary hover:text-primary/80"
                onClick={(event) => event.stopPropagation()}
              >
                {t.demoPayment.cardsLink}
                <ExternalLink className="size-3.5" />
              </Link>
            </div>

            {checkout.isError && (
              <p className="mt-3 text-sm text-destructive">{t.demoPayment.error}</p>
            )}

            <div className="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={closeDialog}
                disabled={checkout.isPending}
              >
                {t.demoPayment.cancel}
              </Button>
              <Button type="button" onClick={startCheckout} disabled={checkout.isPending}>
                {checkout.isPending ? t.common.loading : t.demoPayment.continue}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function localizedConfirmationURL(rawURL: string, locale: string): string {
  if (locale !== "en") return rawURL;

  try {
    const url = new URL(rawURL);
    url.searchParams.set("lang", "en");
    return url.toString();
  } catch {
    const separator = rawURL.includes("?") ? "&" : "?";
    return `${rawURL}${separator}lang=en`;
  }
}
