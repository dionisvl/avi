"use client";

import { Check, Globe2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useLocale, useSetLocale, useT } from "@/i18n";
import { type Locale, localeLabels, SUPPORTED_LOCALES } from "@/i18n/config";
import { cn } from "@/lib/utils";

export function LanguageSwitcher({ className }: { className?: string }) {
  const t = useT();
  const locale = useLocale();
  const setLocale = useSetLocale();
  const router = useRouter();

  const handleLocaleChange = (nextLocale: Locale) => {
    setLocale(nextLocale);
    router.refresh();
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className={cn("h-10 min-w-16 gap-1.5 px-2 text-text-primary", className)}
          aria-label={t.header.language}
        >
          <Globe2 className="size-4" />
          <span className="text-[13px] font-bold">{localeLabels[locale].short}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-36">
        {SUPPORTED_LOCALES.map((option) => (
          <DropdownMenuItem
            key={option}
            onClick={() => handleLocaleChange(option)}
            aria-current={locale === option ? "true" : undefined}
            className="gap-2"
          >
            <Check className={cn("size-4", locale === option ? "opacity-100" : "opacity-0")} />
            <span>{localeLabels[option].name}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
