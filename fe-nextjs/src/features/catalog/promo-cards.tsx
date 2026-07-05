"use client";

import Image from "next/image";
import Link from "next/link";
import { useT } from "@/i18n";
import { cn } from "@/lib/utils";

type PromoCardKey = "home" | "tech" | "travel";

interface PromoCard {
  key: PromoCardKey;
  bgClass: string;
  imageSrc: string;
  imagePosition: string;
  href: string;
}

const cards: PromoCard[] = [
  {
    key: "home",
    bgClass: "bg-promo-home",
    imageSrc: "/design-assets/promo-sofa.png",
    imagePosition: "object-[63%_58%]",
    href: "/items?category=home-garden",
  },
  {
    key: "tech",
    bgClass: "bg-promo-tech",
    imageSrc: "/design-assets/promo-tech.png",
    imagePosition: "object-[70%_56%]",
    href: "/items?category=electronics",
  },
  {
    key: "travel",
    bgClass: "bg-promo-travel",
    imageSrc: "/design-assets/promo-backpack.png",
    imagePosition: "object-[72%_52%]",
    href: "/items?category=hobbies",
  },
];

export default function PromoCards() {
  const t = useT();

  return (
    <div className="flex snap-x snap-mandatory gap-6 overflow-x-auto sm:grid sm:grid-cols-2 sm:overflow-visible lg:grid-cols-3 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      {cards.map((card) => {
        const promo = t.promo[card.key];

        return (
          <Link
            key={card.key}
            href={card.href}
            className={cn(
              "group/promo relative block min-h-[170px] shrink-0 basis-[86%] snap-start overflow-hidden rounded-promo p-7 sm:shrink sm:basis-auto",
              "border border-white/70 shadow-[0_12px_38px_rgba(44,40,80,0.06)] transition-shadow hover:shadow-[0_16px_46px_rgba(44,40,80,0.12)]",
              card.bgClass,
            )}
          >
            <Image
              src={card.imageSrc}
              alt=""
              fill
              className={cn("object-cover", card.imagePosition)}
              sizes="(max-width: 640px) 86vw, (max-width: 1024px) 50vw, 33vw"
              priority={card.key === "home"}
            />
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_48%,rgba(255,255,255,0.92)_0%,rgba(255,255,255,0.74)_24%,rgba(255,255,255,0.18)_48%,rgba(255,255,255,0)_72%)]" />
            <div className="absolute inset-0 bg-[linear-gradient(90deg,rgba(20,19,36,0.24)_0%,rgba(20,19,36,0.10)_34%,rgba(20,19,36,0)_70%)]" />
            <div className="absolute left-5 top-6 h-28 w-[56%] rounded-full bg-white/45 blur-2xl transition-opacity group-hover/promo:opacity-90" />

            <div className="relative z-10 inline-flex max-w-[72%] flex-col items-start gap-3 rounded-[14px] border border-white/45 bg-white/34 p-4 shadow-[0_16px_42px_rgba(44,40,80,0.14),0_0_34px_rgba(255,255,255,0.42)]">
              <div>
                <h3 className="text-[18px] font-extrabold leading-6 text-text-primary drop-shadow-[0_1px_10px_rgba(255,255,255,0.92)]">
                  {promo.title}
                </h3>
                <p className="mt-2 text-[14px] font-semibold leading-5 text-text-primary/75 drop-shadow-[0_1px_8px_rgba(255,255,255,0.78)]">
                  {promo.subtitle}
                </p>
              </div>

              <span className="inline-flex h-9 items-center rounded-[10px] border border-primary/30 bg-white/42 px-4 text-[12px] font-bold text-primary shadow-[0_8px_20px_rgba(91,69,245,0.10)] transition-colors group-hover/promo:bg-white/70">
                {promo.cta}
              </span>
            </div>
          </Link>
        );
      })}
    </div>
  );
}
