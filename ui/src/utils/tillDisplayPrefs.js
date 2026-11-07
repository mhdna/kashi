// A till's screen prefs as kashi's branch settings edit them (kashi-pos
// display_prefs.go validates them key by key when they arrive). Save sends
// them to every register of the branch; "One register only" to one.

// What a till shows when kashi never set a pref.
export const DISPLAY_DEFAULTS = Object.freeze({
  quick_access_rows: 3,
  quick_access_font_scale: 100,
  quick_access_show_code: true,
  quick_access_show_name: true,
  quick_access_show_remove: true,
  animations_enabled: false,
  calculator_total_currency: 'USD',
  theme: 'light',
  // Returns and exchanges print their receipt (with receipts on). Off, an
  // exchange at the till and a return sent from kashi don't print; either
  // can still be printed later from the invoice ("Print on till").
  print_return_receipts: true,
})

// The form for a branch's saved prefs: the defaults, then what was saved.
export function displayFormFrom (saved) {
  const form = { ...DISPLAY_DEFAULTS }
  if (saved && typeof saved === 'object' && !Array.isArray(saved)) {
    for (const [key, value] of Object.entries(saved)) {
      if (value !== undefined) {
        form[key] = value
      }
    }
  }
  return form
}
