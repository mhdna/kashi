import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { appendItems, navItems } from '@/data/navItems'
import { useAuthStore } from '@/stores/auth'

// The nav menus (drawer, command palette, home links) filtered to the pages
// the signed-in user's role can open. A page's section comes from its route's
// `meta.section` (router/index.js); pages without one are open to everyone.
export function useAllowedNav () {
  const auth = useAuthStore()
  const router = useRouter()

  function allowed (item) {
    if (!item.to) {return true}
    const { section, adminOnly } = router.resolve(item.to).meta
    if (adminOnly && !auth.isAdmin) {return false}
    return !section || auth.canView(section)
  }

  // A group with no visible links disappears too.
  const allowedNavItems = computed(() =>
    navItems
      .map(item => (item.children ? { ...item, children: item.children.filter(child => allowed(child)) } : item))
      .filter(item => (item.children ? item.children.length > 0 : allowed(item))),
  )
  const allowedAppendItems = computed(() => appendItems.filter(item => allowed(item)))

  return { allowedNavItems, allowedAppendItems, allowed }
}
