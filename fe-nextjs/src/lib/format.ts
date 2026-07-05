import type { Item, Price } from "./api/types";

/** Inline SVG placeholder shown when a listing has no photo. */
export const PLACEHOLDER_IMAGE =
  "data:image/svg+xml;utf8," +
  encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" width="400" height="300" viewBox="0 0 400 300">` +
      `<rect width="400" height="300" fill="#f3f6fb"/>` +
      `<path d="M150 190l30-40 25 30 20-25 45 55H120z" fill="#e7eaf3"/>` +
      `<circle cx="165" cy="120" r="18" fill="#e7eaf3"/></svg>`,
  );

const CURRENCY_SYMBOL: Record<string, string> = {
  RUB: "₽",
  USD: "$",
  EUR: "€",
};

/**
 * Format a listing price. `amount` is in MINOR units (e.g. kopecks/cents),
 * so we divide by 100. Currency symbol stays after the amount for RUB/EUR,
 * before for USD.
 */
export function formatPrice(
  price: Price | null | undefined,
  locale: string,
  priceOnRequest: string,
): string {
  if (!price || price.amount == null) return priceOnRequest;
  const currency = price.currency ?? "RUB";
  const major = price.amount / 100;
  const grouped = new Intl.NumberFormat(locale, {
    maximumFractionDigits: major % 1 === 0 ? 0 : 2,
  }).format(major);
  const symbol = CURRENCY_SYMBOL[currency] ?? currency;
  return price.currency === "USD" ? `${symbol}${grouped}` : `${grouped} ${symbol}`;
}

/** Format an ISO timestamp as a short relative time ("2 hours ago"). */
export function formatRelativeTime(iso: string | undefined, locale: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const diffSec = Math.round((Date.now() - then) / 1000);
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 31536000],
    ["month", 2592000],
    ["week", 604800],
    ["day", 86400],
    ["hour", 3600],
    ["minute", 60],
  ];
  for (const [unit, secs] of units) {
    if (Math.abs(diffSec) >= secs) {
      return rtf.format(-Math.round(diffSec / secs), unit);
    }
  }
  return rtf.format(-diffSec, "second");
}

/** Pick the best display image for a listing: thumbnail → first photo → placeholder. */
export function pickListingImage(item: Item): string {
  return item.thumbnail?.url || item.photos?.[0]?.url || PLACEHOLDER_IMAGE;
}

/** Human-readable city name for the current locale, falling back to slug. */
export function cityLabel(item: Item, locale: string): string {
  const names = item.city?.names;
  return names?.[locale] || names?.en || item.city?.slug || "";
}
