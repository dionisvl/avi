"use client";

import { useT } from "@/i18n";

export function Hero() {
  const t = useT();

  return (
    <div className="mt-1">
      <h1 className="text-[30px] font-extrabold leading-10 text-text-primary md:text-[34px]">
        {t.hero.title}
      </h1>
    </div>
  );
}
