import type { paths } from "./schema";

/**
 * Clean domain type aliases derived from the generated OpenAPI schema.
 * Components import these ("Item", "Category", "City") instead of the verbose
 * generated schema keys. If the API changes, `npm run gen:api` refreshes them.
 */

type ItemsListResponse =
  paths["/api/v1/items"]["get"]["responses"]["200"]["content"]["application/json"];

/** A single marketplace listing as returned by GET /api/v1/items. */
export type Item = NonNullable<ItemsListResponse["data"]>[number];

/** Pagination envelope shared by list endpoints. */
export type Pagination = NonNullable<ItemsListResponse["pagination"]>;

type CategoriesListResponse =
  paths["/api/v1/categories"]["get"]["responses"]["200"]["content"]["application/json"];

/** A category as returned by GET /api/v1/categories. */
export type Category = NonNullable<CategoriesListResponse["data"]>[number];

type CitiesListResponse =
  paths["/api/v1/cities"]["get"]["responses"]["200"]["content"]["application/json"];

/** A city as returned by GET /api/v1/cities. */
export type City = NonNullable<CitiesListResponse["data"]>[number];

/** Photo attached to a listing. */
export type Photo = NonNullable<Item["photos"]>[number];

/** Price value object: `amount` is in MINOR units (e.g. cents/kopecks). */
export type Price = NonNullable<Item["price"]>;

/** Query params accepted by the items list endpoint. */
export type ItemsQuery = NonNullable<paths["/api/v1/items"]["get"]["parameters"]["query"]>;
