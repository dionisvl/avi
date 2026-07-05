"use client";

import { MapPin, Search } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { useCities } from "@/features/catalog/use-catalog";
import { useLocale, useT } from "@/i18n";
import type { City } from "@/lib/api/types";

export function SearchBar() {
  const router = useRouter();
  const t = useT();
  const locale = useLocale();
  const [query, setQuery] = useState("");
  const [selectedCityId, setSelectedCityId] = useState<string | null>(null);
  const { data: cities } = useCities();

  // Only reflect a city the user actually picked; no implicit default, so the
  // label stays "select city" and search isn't silently scoped to one city.
  const activeCity = selectedCityId
    ? (cities?.find((c) => isCityObject(c) && c.id === selectedCityId) as
        | Exclude<City, string>
        | undefined)
    : undefined;

  const activeCityName = activeCity
    ? activeCity.names?.[locale] || activeCity.names?.en || activeCity.slug || t.header.selectCity
    : t.header.selectCity;

  const handleSearch = () => {
    const params = new URLSearchParams();
    const searchQuery = query.trim();

    if (searchQuery) {
      params.set("search", searchQuery);
    }

    // Only filter by city when the user explicitly picked one. `activeCity`
    // falls back to the first city for the dropdown label, but using that
    // fallback as a filter silently drops every listing outside it.
    if (selectedCityId) {
      params.set("city_uuid", selectedCityId);
    }

    const searchParams = params.toString();
    router.push(searchParams ? `/items?${searchParams}` : "/items");
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  return (
    <div className="mt-5 md:mt-6">
      <div className="group/search flex min-h-[66px] flex-col items-center gap-3 rounded-search border border-border bg-white p-2 shadow-[0_18px_54px_rgba(42,38,80,0.1)] transition-all focus-within:border-primary focus-within:shadow-[0_20px_58px_rgba(42,38,80,0.14),0_0_0_4px_rgba(91,69,245,0.14)] md:flex-row md:gap-0 md:p-0">
        <div className="flex min-h-[58px] flex-1 items-center gap-4 px-4 md:px-6">
          <Search className="size-5 shrink-0 text-primary transition-transform group-focus-within/search:scale-110" />
          <Input
            type="search"
            placeholder={t.hero.searchPlaceholder}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            className="h-auto border-0 bg-transparent p-0 text-[15px] font-semibold text-text-primary [caret-color:var(--color-primary)] placeholder:text-text-muted focus-visible:border-0 focus-visible:ring-0"
          />
        </div>

        <div className="flex w-full items-center gap-2 md:w-auto md:gap-0">
          <div className="hidden h-9 w-px bg-border md:block" />

          <div className="flex-1 md:flex-none md:px-4">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="h-11 w-full justify-start gap-2 rounded-[12px] px-3 text-text-primary hover:bg-surface-soft md:h-[58px] md:w-[145px] md:justify-center md:px-0 md:hover:bg-transparent"
                >
                  <MapPin className="size-4 text-primary" />
                  <span className="min-w-0 truncate text-[14px] font-semibold">
                    {activeCityName}
                  </span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-48">
                {cities?.filter(isCityObject).map((city) => {
                  const cityName = city.names?.[locale] || city.names?.en || city.slug || "Unknown";
                  return (
                    <DropdownMenuItem
                      key={city.id}
                      onClick={() => setSelectedCityId(city.id || null)}
                      className="cursor-pointer"
                    >
                      {cityName}
                    </DropdownMenuItem>
                  );
                })}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          <Button
            onClick={handleSearch}
            className="h-11 flex-1 rounded-[13px] px-7 text-[14px] font-bold shadow-[0_12px_26px_rgba(91,69,245,0.28)] md:mr-2 md:h-[50px] md:flex-none md:px-9"
          >
            {t.common.search}
          </Button>
        </div>
      </div>
    </div>
  );
}

/**
 * Type guard to filter out string cities and keep city objects.
 */
function isCityObject(city: City): city is Exclude<City, string> {
  return typeof city === "object" && city !== null;
}
