// Calendar-day keys (YYYY-MM-DD) in the viewer's own zone. Not
// toISOString().slice(0, 10): that's the UTC day, which in Beirut is still
// yesterday until 02:00/03:00.

export function ymd (d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
