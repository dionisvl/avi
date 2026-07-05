"use client";

import { createContext, type ReactNode, useCallback, useContext, useEffect, useState } from "react";
import { DEFAULT_LOCALE, LOCALE_COOKIE, type Locale } from "./config";
import { type Dictionary, en } from "./en";
import { ru } from "./ru";

const dictionaries: Record<Locale, Dictionary> = { en, ru };

const LocaleContext = createContext<{
  locale: Locale;
  dict: Dictionary;
  setLocale: (locale: Locale) => void;
}>({
  locale: DEFAULT_LOCALE,
  dict: en,
  setLocale: () => {},
});

/** Provides the active locale + dictionary. Defaults to English. */
export function LocaleProvider({
  initialLocale = DEFAULT_LOCALE,
  children,
}: {
  initialLocale?: Locale;
  children: ReactNode;
}) {
  const [locale, setLocaleState] = useState<Locale>(initialLocale);

  const persistLocale = useCallback((nextLocale: Locale) => {
    setLocaleState(nextLocale);
    document.documentElement.lang = nextLocale;
    window.localStorage.setItem(LOCALE_COOKIE, nextLocale);
    // biome-ignore lint/suspicious/noDocumentCookie: Next server components read this cookie after router.refresh().
    document.cookie = `${LOCALE_COOKIE}=${nextLocale}; Max-Age=31536000; Path=/; SameSite=Lax`;
  }, []);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  return (
    <LocaleContext.Provider
      value={{ locale, dict: dictionaries[locale], setLocale: persistLocale }}
    >
      {children}
    </LocaleContext.Provider>
  );
}

/**
 * Access the translation dictionary. Usage: `const t = useT(); t.header.favorites`.
 * Strings are typed, so missing keys are caught at compile time.
 */
export function useT(): Dictionary {
  return useContext(LocaleContext).dict;
}

/** Access the active locale code (for API `locale` params, date formatting, etc.). */
export function useLocale(): Locale {
  return useContext(LocaleContext).locale;
}

export function useSetLocale(): (locale: Locale) => void {
  return useContext(LocaleContext).setLocale;
}
