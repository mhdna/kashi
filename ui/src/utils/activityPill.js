// What the app bar's sync pill shows while branch activity is coming in: one
// message at a time (the newest), split into the branch name and its short
// detail, and how many others are still in their display time. Three messages
// side by side were each cut to "Test Branch: shift b…", so none could be read.

// The pill's tint follows the message's colour; anything outside the theme's
// colours (the visitor counter's 'orange') reads as a warning tint.
const TINTS = new Set(['success', 'error', 'warning', 'info'])

export function pillTint (color) {
  return TINTS.has(color) ? color : 'warning'
}

// splitActivityText turns "Test Branch: shift balanced" into its branch and
// detail. A text with no "branch: " prefix is all detail.
export function splitActivityText (text) {
  const s = String(text ?? '')
  const i = s.indexOf(': ')
  if (i <= 0) {return { branch: '', detail: s }}
  return { branch: s.slice(0, i), detail: s.slice(i + 2) }
}

// activityPill picks what to draw from the visible items (oldest first).
// Returns null when nothing is showing.
export function activityPill (items) {
  if (!items?.length) {return null}
  const current = items.at(-1)
  const { branch, detail } = splitActivityText(current.text)
  return {
    id: current.id,
    // A group ("Centro: 5 to fix" -> "6 to fix") keeps one key, so only its
    // text changes; a new message gets a new key and swaps in.
    key: current.groupKey || `activity-${current.id}`,
    branch,
    detail,
    icon: current.trendIcon || current.icon || 'mdi-cloud',
    color: current.color,
    tint: pillTint(current.color),
    more: items.length - 1,
  }
}
