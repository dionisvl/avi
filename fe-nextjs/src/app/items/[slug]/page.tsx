import { notFound } from "next/navigation";
import { BottomNav } from "@/features/catalog/bottom-nav";
import { Header } from "@/features/catalog/header";
import { getItem } from "@/features/listing/get-item";
import { getRequestLocale } from "@/i18n/server";
import type { Item } from "@/lib/api/types";
import { PLACEHOLDER_IMAGE } from "@/lib/format";
import { ItemDetailClient } from "./item-detail-client";
import { ItemGallery } from "./item-gallery";

export const dynamic = "force-dynamic";

interface ItemPageProps {
  params: Promise<{ slug: string }>;
}

function photoUrls(item: Item): string[] {
  const urls = [
    ...(item.photos ?? []).map((photo) => photo.url || photo.thumbnail_url),
    item.thumbnail?.url || item.thumbnail?.thumbnail_url,
  ].filter((url): url is string => Boolean(url));

  return [...new Set(urls)];
}

export default async function ItemPage({ params }: ItemPageProps) {
  const { slug } = await params;
  const item = await getItem(slug);
  const locale = await getRequestLocale();

  if (!item) notFound();

  const photos = photoUrls(item);
  const galleryPhotos = photos.length > 0 ? photos : [PLACEHOLDER_IMAGE];

  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-16">
        <div className="container-avi grid gap-6 py-5 md:gap-8 md:py-8 lg:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
          <section className="min-w-0">
            <ItemGallery photos={galleryPhotos} title={item.title ?? ""} />
          </section>

          <aside className="min-w-0 lg:sticky lg:top-24 lg:self-start">
            <ItemDetailClient item={item} locale={locale} />
          </aside>
        </div>
      </main>

      <BottomNav />
    </>
  );
}
