// Turns a picker range ([] or two 'YYYY-MM-DD' strings, end inclusive) into
// the API's [from, to) ISO bounds at local midnight. Either is '' when unset.
export function rangeBounds (range) {
  if (!range || range.length === 0) {return { from: '', to: '' }}
  const from = new Date(`${range[0]}T00:00:00`)
  const to = new Date(`${range.at(-1)}T00:00:00`)
  to.setDate(to.getDate() + 1)
  return { from: from.toISOString(), to: to.toISOString() }
}
