import { DEFAULT_TIMEZONE, resolveTimezone } from "@iroha/shared/format/date";

// API base URL for iroha-server. Empty by default so requests are same-origin
// and go through the Vite dev proxy (see vite.config.ts) — this is what makes
// remote access (Tailscale/LAN) work without CORS or a per-host base. Set
// PUBLIC_IROHA_API_BASE to an absolute URL when the API is on another origin
// (e.g. a production deploy not fronted by a reverse proxy).
export const API_BASE = (
  (import.meta.env.PUBLIC_IROHA_API_BASE as string | undefined) ?? ""
).replace(/\/$/, "");
export const APP_VERSION =
  (import.meta.env.PUBLIC_IROHA_VERSION as string | undefined) ?? "dev";
// The web build and iroha-server must use the same IANA timezone. The server
// remains authoritative for API boundaries; this value keeps local URL state,
// date controls, and instant formatting aligned before the first response.
export const IROHA_TIMEZONE = resolveTimezone(
  (import.meta.env.PUBLIC_IROHA_TIMEZONE as string | undefined) ??
    DEFAULT_TIMEZONE,
);
