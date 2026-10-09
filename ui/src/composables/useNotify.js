import { isPermissionDenied, SERVER_UPDATING_MESSAGE } from '@/composables/useApi'
import { useNotifyStore } from '@/stores/notify'

export function useNotify () {
  const store = useNotifyStore()
  return {
    success: (msg, timeout = 3000) => store.push(msg, 'success', timeout),
    // A request that hit a deploy already put the full-screen "Updating GB
    // Cloud" state up (see useServerStatus); a popup repeating it is noise.
    // A role's permission denial isn't announced either (isPermissionDenied).
    error: (msg, timeout = 6000) => {
      if (String(msg ?? '').includes(SERVER_UPDATING_MESSAGE) || isPermissionDenied(msg)) {
        return
      }
      return store.push(msg, 'error', timeout)
    },
    warning: (msg, timeout = 5000) => store.push(msg, 'warning', timeout),
    info: (msg, timeout = 4000) => store.push(msg, 'info', timeout),
  }
}
