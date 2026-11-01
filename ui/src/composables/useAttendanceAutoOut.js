import { computed, ref } from 'vue'
import { useReports } from '@/composables/useReports'
import { parseMinutes } from '@/utils/attendanceSessions'

// "Open a missing IN / close a missing OUT at this hour" for the attendance
// screens and reports. Two hours ('HH:MM', '' = off) shared by all of them.
// The saved defaults live in the report settings (defaults.auto_in /
// defaults.auto_out), the same blob "Save as default" on Reports writes, so
// they are the same for everyone; the fields can still be changed on screen
// without saving.
const autoInTime = ref('')
const autoOutTime = ref('')
let seeded = false

// Which assumed times to mark on screen: 'off' | 'in' | 'out' | 'both'. A
// view option; its saved default is defaults.highlight_auto, like the hours.
const highlightAuto = ref('off')
const HIGHLIGHT_OPTIONS = [
  { title: 'No highlight', value: 'off' },
  { title: 'Highlight auto in', value: 'in' },
  { title: 'Highlight auto out', value: 'out' },
  { title: 'Highlight both', value: 'both' },
]
// Is a session's assumed side ('in' / 'out' / undefined) to be marked?
export const isHighlighted = auto => Boolean(auto) && (highlightAuto.value === 'both' || highlightAuto.value === auto)

function applyDefaults (defaults) {
  autoInTime.value = defaults?.auto_in ?? ''
  autoOutTime.value = defaults?.auto_out ?? ''
  highlightAuto.value = ['in', 'out', 'both'].includes(defaults?.highlight_auto) ? defaults.highlight_auto : 'off'
  seeded = true
}

export function useAttendanceAutoOut () {
  const { getReportSettings, saveReportSettings } = useReports()
  const toMinutes = time => (time ? parseMinutes(time) : null)

  // Seed from the saved defaults the first time a screen needs them.
  async function ensureAutoDefaults () {
    if (seeded) {return}
    seeded = true
    applyDefaults((await getReportSettings().catch(() => null))?.defaults)
  }

  // Store the two hours as the defaults, keeping the rest of the settings.
  async function saveAutoDefaults () {
    const current = (await getReportSettings()) ?? {}
    const defaults = { ...current.defaults, auto_in: autoInTime.value || '', auto_out: autoOutTime.value || '', highlight_auto: highlightAuto.value }
    await saveReportSettings({ ...current, defaults })
    return defaults
  }

  return {
    autoInTime,
    autoOutTime,
    highlightAuto,
    HIGHLIGHT_OPTIONS,
    autoInMinutes: computed(() => toMinutes(autoInTime.value)),
    autoOutMinutes: computed(() => toMinutes(autoOutTime.value)),
    applyAutoDefaults: applyDefaults,
    ensureAutoDefaults,
    saveAutoDefaults,
  }
}
