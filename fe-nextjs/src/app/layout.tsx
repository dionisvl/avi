import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { getRequestLocale } from "@/i18n/server";
import { cn } from "@/lib/utils";
import { Providers } from "./providers";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  display: "swap",
  fallback: ["system-ui", "-apple-system", "Segoe UI", "sans-serif"],
});

export const metadata: Metadata = {
  title: "Avi — Classifieds Marketplace",
  description: "Buy and sell locally on Avi. Find what you need.",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const initialLocale = await getRequestLocale();

  return (
    <html lang={initialLocale} className={cn("h-full antialiased", inter.variable)}>
      <body className="min-h-full flex flex-col bg-background text-foreground font-sans">
        <Providers initialLocale={initialLocale}>{children}</Providers>
      </body>
    </html>
  );
}
