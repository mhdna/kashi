// What the Branches page says after the Shared switch flips. Off keeps a
// branch (a test branch) out of reports, and its own clients, the ones no
// other branch has, come off every other till.
export function hiddenBranchMessage (name, hidden, clients) {
  const n = Number(clients) || 0
  const who = n === 1 ? '1 client' : `${n} clients`
  if (hidden) {
    return n > 0
      ? `${name} is out of reports; its ${who} will leave the other tills`
      : `${name} is out of reports and keeps its clients to itself`
  }
  return n > 0
    ? `${name} is back in reports; its ${who} will return to the other tills`
    : `${name} is back in reports`
}
