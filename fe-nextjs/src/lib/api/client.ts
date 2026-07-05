import createClient from "openapi-fetch";
import type { paths } from "./schema";

/**
 * Base URL of the Avi Go API.
 *
 * Browser requests use the public Traefik host from NEXT_PUBLIC_API_URL.
 * Server-side rendering runs inside Docker, where public dev hosts may resolve
 * to the container itself, so it uses API_INTERNAL_URL when available.
 */
export const API_BASE_URL =
  typeof window === "undefined"
    ? (process.env.API_INTERNAL_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "http://api.avi.test")
    : (process.env.NEXT_PUBLIC_API_URL ?? "http://api.avi.test");

/** Typed API client generated from the Go API's OpenAPI schema. */
export const api = createClient<paths>({ baseUrl: API_BASE_URL });
