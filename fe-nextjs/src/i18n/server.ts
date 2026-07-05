import { cookies } from "next/headers";
import { LOCALE_COOKIE, type Locale, normalizeLocale } from "./config";

export async function getRequestLocale(): Promise<Locale> {
  const cookieStore = await cookies();
  return normalizeLocale(cookieStore.get(LOCALE_COOKIE)?.value);
}
