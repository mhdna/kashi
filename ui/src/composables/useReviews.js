import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (path, options) {
  const res = await authFetch(`${API_BASE}${path}`, options)
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {throw new Error(body.error || `Request failed with status ${res.status}`)}
  return body
}

function sendJSON (method, path, payload) {
  return requestJSON(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

// The Reviews page: the review links in the post-sale WhatsApp message and
// what Google shows for each place -- see api/review_stats.go and
// api/review_link.go.
export function useReviews () {
  return {
    getReviews: params => requestJSON(`/reviews?${new URLSearchParams(params)}`),
    refreshReviews: () => sendJSON('POST', '/reviews/refresh', {}),
    getReviewSettings: async () => (await requestJSON('/reviews/settings')).settings,
    saveReviewSettings: async settings => (await sendJSON('PUT', '/reviews/settings', settings)).settings,
  }
}
