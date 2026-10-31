import { confirmDialog } from '@/composables/useDialog'

// Every v-dialog closes on a click outside or Escape, which used to throw
// away whatever was typed in it. This asks first, for every dialog at once:
// a dialog's fields are read when the user first touches it, and if they've
// changed by the time a click outside (or Escape) would close it, "Discard
// your changes?" comes up. Unchanged dialogs close as before.
//
// It works from the page, ahead of Vuetify: window capture listeners run
// before Vuetify's own (click-outside listens on document, Escape on window).
// A dialog with fields that change on their own and aren't edits can opt
// out with data-no-discard-guard on anything inside it. The command palette
// (Ctrl/Cmd+K) is skipped too: its search box is all it has, and Vuetify
// renders its dialog, so there is nowhere to put the attribute.

// The fields' state as text, to compare: inputs, switches, toggled buttons,
// chosen chips and dates.
function snapshot (content) {
  const parts = []
  for (const el of content.querySelectorAll('input, textarea, select')) {
    parts.push(el.type === 'checkbox' || el.type === 'radio' ? String(el.checked) : el.value)
  }
  for (const el of content.querySelectorAll('.v-btn--active, [aria-pressed="true"], .v-field .v-chip, .v-select__selection, .v-date-picker-month__day--selected')) {
    parts.push(el.textContent.trim() || el.className)
  }
  return parts.join('\u0000')
}

// The fields as they were when the user first touched each dialog.
const baselines = new WeakMap()

// The overlay on top: Vuetify raises each newly opened one's z-index.
function topOverlay () {
  let top = null
  for (const el of document.querySelectorAll('.v-overlay--active')) {
    if (!top || Number(el.style.zIndex) >= Number(top.style.zIndex)) {top = el}
  }
  return top
}

function guardedContent (overlay) {
  if (!overlay?.classList.contains('v-dialog') || overlay.classList.contains('app-question-dialog') || overlay.classList.contains('v-command-palette')) {return null}
  const content = overlay.querySelector(':scope > .v-overlay__content')
  if (!content || content.querySelector('[data-no-discard-guard]')) {return null}
  return content
}

function isDirty (content) {
  const baseline = baselines.get(content)
  return baseline !== undefined && baseline !== snapshot(content)
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
    baselines.delete(content)
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
    if (content && !baselines.has(content)) {baselines.set(content, snapshot(content))}
  }
  window.addEventListener('pointerdown', remember, true)
  window.addEventListener('keydown', remember, true)

  // A dialog keeps its content element when it closes (Vuetify hides it), so
  // forget what was noted once it closes, however it closed: opened again,
  // maybe for another record, it starts afresh.
  new MutationObserver(records => {
    for (const { target } of records) {
      if (target.classList.contains('v-dialog') && !target.classList.contains('v-overlay--active')) {
        const content = target.querySelector(':scope > .v-overlay__content')
        if (content) {
          baselines.delete(content)
        }
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
