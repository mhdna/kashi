import { computed, ref } from 'vue'
import { API_BASE } from '@/config'
import { authFetch } from './useApi'

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Request failed with status ${res.status}`)
  }
  return res.json().catch(() => null)
}

// What each command type is called on screen. The type strings are the
// ones kashi-pos's executor knows (commands.go there).
export const COMMAND_TYPE_LABELS = {
  update_settings: 'Update settings',
  remote_return_invoice: 'Return invoice',
  set_branch_user_pin: 'Set PIN',
  print_shift_report: 'Print close shift',
  reprint_invoice: 'Reprint invoice',
  sync_now: 'Sync now',
  transfer_decision: 'Transfer decision',
}

// Shared across the nav drawer and the Commands page, so both show the same
// queue and a cancel/retry on the page turns the nav dot at once.
const summary = ref([])
const summaryLoaded = ref(false)

// Green when every command has been run, yellow while any is still queued
// for a till. 'loading' (a grey placeholder dot in the nav) until the first
// summary arrives, rather than a green one it hasn't checked.
const commandsStatus = computed(() => {
  if (!summaryLoaded.value) {return 'loading'}
  return summary.value.some(b => b.pending > 0) ? 'warning' : 'healthy'
})

// The remote command channel: kashi queues a command, each target till
// fetches it (instantly over its WebSocket, or on its next sync), runs it
// through its own local function and reports success or failure back. See
// api/branch_command.go.
export function useBranchCommands () {
  async function refreshCommandSummary () {
    const data = await requestJSON(`${API_BASE}/branch_commands/summary`)
    summary.value = data?.branches ?? []
    summaryLoaded.value = true
    return summary.value
  }

  // registerId 0 sends it to every active register of the branch. Commands
  // about one sale or shift ignore it: kashi sends them to the till that
  // has that sale or shift.
  async function sendCommand (branchId, type, payload = {}, registerId = 0) {
    const data = await requestJSON(`${API_BASE}/branches/${branchId}/commands`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type, payload, register_id: registerId }),
    })
    refreshCommandSummary().catch(() => {})
    return data?.commands ?? []
  }

  async function cancelCommand (id) {
    const data = await requestJSON(`${API_BASE}/branch_commands/${id}/cancel`, { method: 'POST' })
    refreshCommandSummary().catch(() => {})
    return data?.command ?? null
  }

  async function retryCommand (id) {
    const data = await requestJSON(`${API_BASE}/branch_commands/${id}/retry`, { method: 'POST' })
    refreshCommandSummary().catch(() => {})
    return data?.command ?? null
  }

  // Reprints a closed branch shift's end-of-shift receipt on the till that
  // closed it. shift is a branch_shifts row from kashi.
  function printBranchShift (shift) {
    return sendCommand(shift.branch_id, 'print_shift_report', { shift_id: shift.id })
  }

  return {
    summary,
    commandsStatus,
    refreshCommandSummary,
    sendCommand,
    cancelCommand,
    retryCommand,
    printBranchShift,
  }
}
