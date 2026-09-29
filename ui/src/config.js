// VITE_API_URL may be a same-origin path like "/api" (production, behind
// Caddy) or a full origin like "http://localhost:8090" (local dev).
// Resolve it to an absolute URL so `new URL(`${API_BASE}/x`)` works
// everywhere -- a bare "/api" throws "Failed to construct 'URL'".
const rawApiBase = import.meta.env.VITE_API_URL || 'http://localhost:8090'
export const API_BASE = /^https?:\/\//.test(rawApiBase)
  ? rawApiBase
  : window.location.origin + rawApiBase
export const WS_BASE = API_BASE.replace(/^http/, 'ws')

// Temporary device gate: blocks mobile/tablet user agents (see App.vue),
// not narrow viewports. Set VITE_BLOCK_NON_DESKTOP=false to restore access
// on those devices without removing the unsupported view.
export const BLOCK_NON_DESKTOP = import.meta.env.VITE_BLOCK_NON_DESKTOP !== 'false'
