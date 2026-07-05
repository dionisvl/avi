"use client";

import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { useT } from "@/i18n";

export default function HelpPage() {
  const t = useT();

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <section className="container-avi flex min-h-[360px] flex-col justify-center py-8">
          <h1 className="text-h1 text-text-primary">{t.help.title}</h1>
          <p className="mt-3 max-w-xl text-body text-text-secondary">{t.help.subtitle}</p>
        </section>
      </main>

      <BottomNav />
    </>
  );
}
