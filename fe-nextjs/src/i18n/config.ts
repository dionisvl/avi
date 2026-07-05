export const DEFAULT_LOCALE = "en";
export const LOCALE_COOKIE = "avi_locale";
export const SUPPORTED_LOCALES = ["en", "ru"] as const;

export type Locale = (typeof SUPPORTED_LOCALES)[number];

export const localeLabels: Record<Locale, { name: string; short: string }> = {
  en: { name: "English", short: "EN" },
  ru: { name: "Русский", short: "RU" },
};

export function isLocale(value: string | null | undefined): value is Locale {
  return SUPPORTED_LOCALES.includes(value as Locale);
}

export function normalizeLocale(value: string | null | undefined): Locale {
  return isLocale(value) ? value : DEFAULT_LOCALE;
}
