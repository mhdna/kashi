import { computed, ref, watch } from 'vue'
import { parseMinutes } from '@/utils/attendanceSessions'

// "Close a missing OUT at this hour" for the attendance screens and reports.
// One setting shared by all of them, kept in this browser. '' = off,
// otherwise 'HH:MM'.
const KEY = 'attendance.autoOut'
const time = ref(localStorage.getItem(KEY) ?? '')
watch(time, value => localStorage.setItem(KEY, value ?? ''))

export function useAttendanceAutoOut () {
  // Minutes since midnight for buildTimesheet, null when off.
  const minutes = computed(() => (time.value ? parseMinutes(time.value) : null))
  return { autoOutTime: time, autoOutMinutes: minutes }
}
