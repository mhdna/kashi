import { ref, shallowRef } from 'vue'

// Rows per page of a home feed.
export const FEED_PAGE_SIZE = 30

// feedQuery builds a home feed's query string: the page size and, after the
// first page, the cursor -- the sort key of the last row the feed already
// has (see feedCursor in api/home_latest.go).
export function feedQuery (cursor) {
  const params = new URLSearchParams({ page_size: String(FEED_PAGE_SIZE) })
  if (cursor) {
    params.set('before_at', cursor.at)
    params.set('before_id', String(cursor.id))
    if (cursor.kind) {params.set('before_kind', cursor.kind)}
  }
  return params.toString()
}

// useKeysetFeed keeps one newest-first feed that loads FEED_PAGE_SIZE rows
// at a time as it's scrolled.
//
//   fetchPage(last)  resolves to the page of rows after `last` (undefined
//                    for the first page)
//   keyOf(row)       a row's unique key
//
// loadMore() appends the next page; refresh() re-reads the first page for a
// live push or poll and splices it on top of what's loaded, so a feed
// scrolled far down keeps its rows and its place. Rows live in a shallowRef:
// they're replaced, never edited, and a long feed shouldn't be made deeply
// reactive.
export function useKeysetFeed ({ fetchPage, keyOf, onError }) {
  const items = shallowRef([])
  const loaded = ref(false)
  const loading = ref(false)
  const done = ref(false)
  // Bumped when refresh() throws the loaded rows away, so a loadMore()
  // still in flight from before doesn't append to the new list.
  let generation = 0

  async function loadMore () {
    if (loading.value || done.value || !loaded.value) {return}
    const started = generation
    loading.value = true
    try {
      const rows = await fetchPage(items.value.at(-1))
      if (started !== generation) {return}
      const seen = new Set(items.value.map(row => keyOf(row)))
      items.value = [...items.value, ...rows.filter(row => !seen.has(keyOf(row)))]
      done.value = rows.length < FEED_PAGE_SIZE
    } catch (error) {
      onError?.(error)
    } finally {
      if (started === generation) {loading.value = false}
    }
  }

  async function refresh () {
    try {
      const rows = await fetchPage()
      const current = items.value
      // Where the fresh page's last row sits in what's loaded. Everything
      // above it is covered by the fresh page (rows that changed, like a
      // bumped repeat count, or were removed come out right); everything
      // below is kept as it was.
      const last = rows.length === FEED_PAGE_SIZE ? keyOf(rows.at(-1)) : null
      const at = last === null ? -1 : current.findIndex(row => keyOf(row) === last)
      if (at === -1) {
        // First load, a short feed that fits in one page, or more new rows
        // than a page (a branch catching up): start over from the top.
        generation++
        loading.value = false
        items.value = rows
        done.value = rows.length < FEED_PAGE_SIZE
      } else {
        items.value = [...rows, ...current.slice(at + 1)]
      }
    } catch (error) {
      onError?.(error)
    } finally {
      loaded.value = true
    }
  }

  return { items, loaded, loading, done, loadMore, refresh }
}

// nearFeedEnd: whether a scrolled feed is within a few rows of its bottom,
// time to fetch the next page so it's there before it's needed.
export function nearFeedEnd (element, margin = 600) {
  return element.scrollHeight - element.scrollTop - element.clientHeight < margin
}
