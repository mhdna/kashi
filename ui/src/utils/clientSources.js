// "Where did they hear about us?" options, as GET /client_sources returns
// them: { id, uid, name, sort_order, is_active, client_count }. The till asks
// the question from the active ones, in this order.

export function sortSources (rows) {
  return (rows || []).toSorted((a, b) => (a.sort_order - b.sort_order) || (a.id - b.id))
}

// Each option's share of the customers who answered, in whole percent that
// add up to exactly 100 (largest remainders get the leftover points).
export function withShares (rows) {
  const list = sortSources(rows)
  const total = list.reduce((sum, r) => sum + Number(r.client_count || 0), 0)
  if (total === 0) {return list.map(r => ({ ...r, share: 0 }))}
  const exact = list.map(r => (Number(r.client_count || 0) * 100) / total)
  const shares = exact.map(v => Math.floor(v))
  let left = 100 - shares.reduce((a, b) => a + b, 0)
  const order = exact
    .map((v, i) => ({ i, rem: v - Math.floor(v) }))
    .toSorted((a, b) => (b.rem - a.rem) || (a.i - b.i))
  for (let k = 0; left > 0; k++, left--) {shares[order[k].i]++}
  return list.map((r, i) => ({ ...r, share: shares[i] }))
}

export function totalAnswered (rows) {
  return (rows || []).reduce((sum, r) => sum + Number(r.client_count || 0), 0)
}

// The sort order a new option gets: after every other one.
export function nextSortOrder (rows) {
  return (rows || []).reduce((max, r) => Math.max(max, Number(r.sort_order || 0)), 0) + 1
}

// moveSource swaps the option at index with its neighbour (dir -1 up, +1
// down) and returns the options whose sort_order changed, numbered 1..n so
// ties left by older rows are settled. Empty when it can't move.
export function moveSource (rows, index, dir) {
  const list = sortSources(rows)
  const to = index + dir
  if (index < 0 || index >= list.length || to < 0 || to >= list.length) {return []}
  ;[list[index], list[to]] = [list[to], list[index]]
  const changed = []
  for (const [i, r] of list.entries()) {
    if (r.sort_order !== i + 1) {changed.push({ ...r, sort_order: i + 1 })}
  }
  return changed
}

// sourceNameError is the message for a name the form can't save, or ''.
export function sourceNameError (name, rows, editingId = 0) {
  const clean = String(name || '').trim()
  if (!clean) {return 'Enter a name for the option'}
  const taken = (rows || []).some(r => r.id !== editingId && String(r.name).trim().toLowerCase() === clean.toLowerCase())
  return taken ? 'There is already an option with this name' : ''
}
