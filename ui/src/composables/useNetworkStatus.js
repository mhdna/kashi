import { readonly, ref } from 'vue'

// Module-level so every caller shares one listener pair and one value.
// navigator.onLine === false is reliable (the device has no network at all);
// true only means "some network", so a server outage still shows up as the
// server being down rather than the internet.
const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine !== false)

if (typeof window !== 'undefined') {
  window.addEventListener('online', () => {
    online.value = true
  })
  window.addEventListener('offline', () => {
    online.value = false
  })
}

export function useNetworkStatus () {
  return { online: readonly(online) }
}
