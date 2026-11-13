import { confirmDialog } from '@/composables/useDialog'

// Every v-dialog closes on a click outside or Escape, which used to throw
// away whatever was typed in it. This asks first, for every dialog at once:
// each field of a dialog is noted when the user touches the dialog, and if
// one has changed by the time a click outside (or Escape) would close it,
// "Discard your changes?" comes up. Unchanged dialogs close as before.
//
// It works from the page, ahead of Vuetify: window capture listeners run
// before Vuetify's own (click-outside listens on document, Escape on window).
//
// Only edits count, so:
// - Fields are compared one by one, and only those that were there when the
//   user touched the dialog: rows that arrive with live data (a till's cart,
//   a loaded list) aren't changes.
// - Tabs and other plain buttons aren't fields; only toggle groups, chip
//   groups, date pickers and inputs are.
// - A view control that isn't an edit (a branch picker, a search box, a
//   filter) is left out with data-discard-ignore on it or on what holds it.
// - A dialog that saves as you go, or is all filters, opts out whole with
//   data-no-discard-guard on anything inside it. The command palette
//   (Ctrl/Cmd+K) is skipped too: its search box is all it has, and Vuetify
//   renders its dialog, so there is nowhere to put the attribute.

// One field each: a Vuetify input (text, select, switch, checkbox...), a
// toggle group, a chip group, a date picker, or a bare native input.
const FIELD = '.v-input, .v-btn-toggle, .v-chip-group, .v-date-picker, input, textarea, select'
// What shows a field's value, inside it.
const SHOWN = '.v-btn--active, [aria-pressed="true"], .v-field .v-chip, .v-chip--selected, .v-select__selection, .v-date-picker-month__day--selected'

export function fieldsOf (content) {
  return [...content.querySelectorAll(FIELD)].filter(el =>
    !el.closest('[data-discard-ignore]')
    && !el.parentElement?.closest(FIELD), // the input inside a .v-input is that field, not another
  )
}

// A field's state as text, to compare.
export function fieldState (field) {
  const parts = []
  const inputs = field.matches('input, textarea, select') ? [field] : field.querySelectorAll('input, textarea, select')
  for (const el of inputs) {
    parts.push(el.type === 'checkbox' || el.type === 'radio' ? String(el.checked) : el.value)
  }
  for (const el of field.querySelectorAll(SHOWN)) {
    parts.push(el.textContent.trim() || el.className)
  }
  return parts.join('\u0000')
}

// Per dialog, each field as it was when the user first touched the dialog
// with that field in it.
const baselines = new WeakMap()

export function rememberFields (content) {
  let noted = baselines.get(content)
  if (!noted) {
    noted = new WeakMap()
    baselines.set(content, noted)
  }
  for (const field of fieldsOf(content)) {
    if (!noted.has(field)) {noted.set(field, fieldState(field))}
  }
}

export function isDirty (content) {
  const noted = baselines.get(content)
  if (!noted) {return false}
  return fieldsOf(content).some(field => noted.has(field) && noted.get(field) !== fieldState(field))
}

function forget (content) {
  baselines.delete(content)
}

// The overlay on top: Vuetify raises each newly opened one's z-index.
function topOverlay () {
  let top = null
  for (const el of document.querySelectorAll('.v-overlay--active')) {
    if (!top || Number(el.style.zIndex) >= Number(top.style.zIndex)) {top = el}
  }
  return top
}

// A dialog's own content box (not a nested dialog's).
function contentOf (overlay) {
  return [...overlay.children].find(el => el.classList.contains('v-overlay__content')) ?? null
}

function guardedContent (overlay) {
  if (!overlay?.classList.contains('v-dialog') || overlay.classList.contains('app-question-dialog') || overlay.classList.contains('v-command-palette')) {return null}
  const content = contentOf(overlay)
  if (!content || content.querySelector('[data-no-discard-guard]')) {return null}
  return content
}

let closing = false

async function askThenClose (content) {
  const discard = await confirmDialog({
    title: 'Discard your changes?',
    text: 'What you changed in this window hasn\'t been saved.',
    okText: 'Discard',
    danger: true,
  })
  if (!discard) {return}
  // Vuetify hands the top spot back to the dialog a tick after the question
  // closes; then its own Escape handling closes it.
  setTimeout(() => {
    if (!content.isConnected) {return}
    forget(content)
    closing = true
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    closing = false
  }, 50)
}

export function installDialogGuard () {
  // Before a pointer or key changes anything, note how the dialog's fields
  // look. Not on focus: a field with autofocus would note them before a
  // dialog's defaults have loaded, and those would count as changes.
  const remember = e => {
    const content = e.target instanceof Element && e.target.closest('.v-dialog > .v-overlay__content')
    if (content) {rememberFields(content)}
  }
  window.addEventListener('pointerdown', remember, true)
  window.addEventListener('keydown', remember, true)

  // A dialog keeps its content element when it closes (Vuetify hides it), so
  // forget what was noted once it closes, however it closed: opened again,
  // maybe for another record, it starts afresh.
  new MutationObserver(records => {
    for (const { target } of records) {
      if (target.classList.contains('v-dialog') && !target.classList.contains('v-overlay--active')) {
        const content = contentOf(target)
        if (content) {forget(content)}
      }
    }
  }).observe(document.body, { attributes: true, attributeFilter: ['class'], subtree: true })

  // A click on the scrim of the dialog on top.
  window.addEventListener('click', e => {
    if (!(e.target instanceof Element) || !e.target.classList.contains('v-overlay__scrim')) {return}
    const overlay = e.target.parentElement
    if (overlay !== topOverlay()) {return}
    const content = guardedContent(overlay)
    if (!content || !isDirty(content)) {return}
    e.stopPropagation()
    askThenClose(content)
  }, true)

  window.addEventListener('keydown', e => {
    if (closing || e.key !== 'Escape') {return}
    const content = guardedContent(topOverlay())
    if (!content || !isDirty(content)) {return}
    e.stopPropagation()
    e.preventDefault()
    askThenClose(content)
  }, true)
}
