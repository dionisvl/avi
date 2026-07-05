/**
 * Client-side token storage using document.cookie.
 * Tokens are stored with Path=/, SameSite=Lax, and a 30-day max-age.
 * Note: These are NOT httpOnly cookies (intentional for demo mode).
 */

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

const ACCESS_TOKEN_KEY = "avi_access_token";
const REFRESH_TOKEN_KEY = "avi_refresh_token";
const TOKEN_MAX_AGE = 30 * 24 * 60 * 60; // 30 days in seconds

/**
 * `; Secure` on HTTPS, empty string on HTTP. The demo runs over plain HTTP in
 * dev (avi.test); adding Secure unconditionally would drop the cookie there.
 * On the HTTPS prod demo this hardens the cookie. Must match on set and clear,
 * otherwise the browser won't overwrite/remove the existing cookie.
 */
function secureAttr(): string {
  return typeof location !== "undefined" && location.protocol === "https:" ? "; Secure" : "";
}

/**
 * Save both tokens to cookies.
 */
export function saveTokens({ access_token, refresh_token }: TokenPair): void {
  const cookieOptions = `Path=/; SameSite=Lax; Max-Age=${TOKEN_MAX_AGE}${secureAttr()}`;
  // biome-ignore lint/suspicious/noDocumentCookie: Intentional cookie storage for demo mode
  document.cookie = `${ACCESS_TOKEN_KEY}=${encodeURIComponent(access_token)}; ${cookieOptions}`;
  // biome-ignore lint/suspicious/noDocumentCookie: Intentional cookie storage for demo mode
  document.cookie = `${REFRESH_TOKEN_KEY}=${encodeURIComponent(refresh_token)}; ${cookieOptions}`;
}

/**
 * Retrieve the access token from cookies.
 */
export function getAccessToken(): string | null {
  return getCookie(ACCESS_TOKEN_KEY);
}

/**
 * Retrieve the refresh token from cookies.
 */
export function getRefreshToken(): string | null {
  return getCookie(REFRESH_TOKEN_KEY);
}

/**
 * Clear both tokens from cookies by setting Max-Age=0.
 */
export function clearTokens(): void {
  const clearOptions = `Path=/; SameSite=Lax; Max-Age=0${secureAttr()}`;
  // biome-ignore lint/suspicious/noDocumentCookie: Intentional cookie clearing
  document.cookie = `${ACCESS_TOKEN_KEY}=; ${clearOptions}`;
  // biome-ignore lint/suspicious/noDocumentCookie: Intentional cookie clearing
  document.cookie = `${REFRESH_TOKEN_KEY}=; ${clearOptions}`;
}

/**
 * Internal helper to retrieve a cookie by name.
 */
function getCookie(name: string): string | null {
  const cookies = document.cookie.split(";");
  for (const cookie of cookies) {
    const [key, value] = cookie.trim().split("=");
    if (key === name && value) {
      return decodeURIComponent(value);
    }
  }
  return null;
}
