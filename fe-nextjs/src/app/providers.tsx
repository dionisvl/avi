"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useState } from "react";
import { LocaleProvider } from "@/i18n";
import type { Locale } from "@/i18n/config";
import { AuthProvider } from "@/lib/auth/context";

/** Client-side app providers: TanStack Query cache + auth + locale context. */
export function Providers({
  children,
  initialLocale,
}: {
  children: ReactNode;
  initialLocale: Locale;
}) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60_000,
            refetchOnWindowFocus: false,
            retry: 1,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <LocaleProvider initialLocale={initialLocale}>{children}</LocaleProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}
