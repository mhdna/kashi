import { ref, watch } from 'vue'
import { dedupedFetch } from './useRequestDedup'

// Infinite-scroll paging for one low-stock column (Out of Stock / Low
// Stock / Ignored). Mirrors the near-bottom paging in ClientLoyaltyBoard:
// `done` stops us paging past the end and `seq` guards against an older
// in-flight page overwriting a newer search result.
const PAGE_SIZE = 40

export function useAlertList (endpoint, status = '') {
  const items = ref([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref('')
  const search = ref('')

  let done = false
  let seq = 0

  async function load (reset) {
    if (loading.value) {return}
    if (!reset && done) {return}

    loading.value = true
    error.value = ''
    const mySeq = ++seq
    try {
      const params = new URLSearchParams()
      if (status) {params.set('status', status)}
      if (search.value.trim()) {params.set('search', search.value.trim())}
      params.set('page_size', String(PAGE_SIZE))
      params.set('page_id', String(reset ? 0 : items.value.length))

      const data = await dedupedFetch(`${endpoint}?${params}`)
      if (mySeq !== seq) {return}

      const batch = data?.products ?? []
      total.value = data?.total ?? 0
      items.value = reset ? batch : [...items.value, ...batch]
      done = batch.length < PAGE_SIZE || items.value.length >= total.value
    } catch (error_) {
      if (mySeq !== seq) {return}
      if (reset) {items.value = []}
      total.value = 0
      error.value = error_.message
    } finally {
      if (mySeq === seq) {loading.value = false}
    }
  }

  function reload () {
    done = false
    load(true)
  }

  function onScroll (event) {
    const el = event.target
    if (el && el.scrollTop + el.clientHeight >= el.scrollHeight - 240) {load(false)}
  }

  let searchTimeout = null
  watch(search, () => {
    clearTimeout(searchTimeout)
    searchTimeout = setTimeout(reload, 300)
  })

  return { items, total, loading, error, search, load, reload, onScroll }
}
