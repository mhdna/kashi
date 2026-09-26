import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

// What the signed-in user's role allows on the current page. The section comes
// from the route's `meta.section` (router/index.js), so a page just asks
// "can I edit?" and never has to know which section it belongs to:
//
//   const { canEdit } = usePageAccess()
//   <v-btn v-if="canEdit" ...>Add</v-btn>
//
// Read-only users still get the page; only the add/edit/delete controls go.
// This is a courtesy -- the API rejects writes without edit access anyway
// (api/access.go).
export function usePageAccess () {
  const auth = useAuthStore()
  const route = useRoute()

  const canEdit = computed(() => {
    const section = route.meta.section
    return section ? auth.canEdit(section) : true
  })

  return { canEdit, isAdmin: computed(() => auth.isAdmin) }
}
