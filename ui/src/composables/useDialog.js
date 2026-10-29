import { reactive } from 'vue'

// The app's one question dialog (components/AppDialog.vue, mounted in
// App.vue), in place of the browser's confirm() and prompt(): await
// confirmDialog(...) for yes/no, promptDialog(...) for a typed answer.
export const dialogState = reactive({
  open: false,
  kind: 'confirm', // 'confirm' or 'prompt'
  title: '',
  text: '',
  label: '',
  value: '',
  type: 'text',
  prefix: '',
  hint: '',
  okText: 'OK',
  danger: false,
  resolve: null,
})

function show (kind, options) {
  // A question still open is answered "cancel" before the next one shows.
  dialogState.resolve?.(kind === 'confirm' ? false : null)
  return new Promise(resolve => {
    Object.assign(dialogState, {
      kind,
      title: '',
      text: '',
      label: '',
      type: 'text',
      prefix: '',
      hint: '',
      okText: 'OK',
      danger: false,
      ...options,
      value: String(options.value ?? ''),
      open: true,
      resolve,
    })
  })
}

// confirmDialog resolves true for OK, false for Cancel or a click outside.
// danger: true colours OK red, for a delete.
export function confirmDialog (options) {
  return show('confirm', typeof options === 'string' ? { text: options } : options)
}

// promptDialog resolves the typed text, or null for Cancel.
export function promptDialog (options) {
  return show('prompt', options)
}

export function closeDialog (answer) {
  const resolve = dialogState.resolve
  dialogState.resolve = null
  dialogState.open = false
  resolve?.(answer)
}
