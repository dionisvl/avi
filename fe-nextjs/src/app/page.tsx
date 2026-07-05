import { BottomNav } from "@/features/catalog/bottom-nav";
import { CategoryChips } from "@/features/catalog/category-chips";
import { Header } from "@/features/catalog/header";
import PromoCards from "@/features/catalog/promo-cards";
import HomeSections from "@/features/listing/home-sections";
import InfiniteLoader from "@/features/listing/infinite-loader";
import { Hero } from "@/features/search/hero";
import { SearchBar } from "@/features/search/search-bar";

/**
 * Avi homepage. Block order follows the design manual §8:
 * Header → Hero → Search → Categories → Promo (3) → Recommendations (6)
 * → New listings (8) → Infinite scroll.
 * Data is fetched client-side from the Go API via TanStack Query hooks.
 */
export default function HomePage() {
  return (
    <>
      <Header />

      <main className="flex-1 pb-24 lg:pb-12">
        <div className="container-avi flex flex-col gap-7 pt-7 md:gap-8 md:pt-9">
          <section className="mx-auto w-full max-w-[1360px]">
            <Hero />
            <SearchBar />
          </section>

          <section className="mx-auto w-full max-w-[1540px]">
            <CategoryChips />
          </section>

          <section className="mx-auto w-full max-w-[1540px]">
            <PromoCards />
          </section>

          <div className="mx-auto w-full max-w-[1540px]">
            <HomeSections />
          </div>

          <section className="mx-auto w-full max-w-[1600px] rounded-container border border-border/80 bg-white/55 px-5 py-6 shadow-[0_18px_60px_rgba(54,49,104,0.06)]">
            <InfiniteLoader />
          </section>
        </div>
      </main>

      <BottomNav />
    </>
  );
}
