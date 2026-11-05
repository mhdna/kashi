import { computed, ref, watch } from 'vue'

// The labels waiting to be printed together, for the Barcodes page: add products
// as you find them, print them all into one PDF later. Kept in this browser, so it
// survives a reload but isn't shared with other people.
const STORAGE_KEY = 'kashi.barcodeQueue'

function load () {
  try {
    const saved = JSON.parse(localStorage.getItem(STORAGE_KEY))
    return Array.isArray(saved) ? saved : []
  } catch {
    return []
  }
}

export function useBarcodeQueue () {
  const queue = ref(load())

  watch(queue, value => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
    } catch {
      // A browser refusing storage still has the queue until the page closes.
    }
  }, { deep: true })

  const labelCount = computed(() => queue.value.reduce((n, entry) => n + entry.qty, 0))

  // Adding a product already queued adds to its quantity, as a second batch of it.
  function add (row, qty) {
    const amount = Math.max(1, Number(qty) || 1)
    const existing = queue.value.find(entry => entry.id === row.id)
    if (existing) {
      existing.qty += amount
      return
    }
    queue.value.push({
      id: row.id,
      name: row.product_name,
      code: row.product_code,
      variant: [row.color_name, row.size_name].filter(Boolean).join(' / '),
      barcode: row.barcode,
      qty: amount,
    })
  }

  function remove (id) {
    queue.value = queue.value.filter(entry => entry.id !== id)
  }

  function clear () {
    queue.value = []
  }

  return { queue, labelCount, add, remove, clear }
}
