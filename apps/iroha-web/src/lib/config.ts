import { env } from '$env/dynamic/public';

// API base URL for iroha-server. Configurable via PUBLIC_IROHA_API_BASE;
// defaults to the local server address (127.0.0.1:8080).
export const API_BASE = (env.PUBLIC_IROHA_API_BASE ?? 'http://127.0.0.1:8080').replace(/\/$/, '');
