import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { appendItems, navItems, navSections } from '@/data/navItems'
import { useAuthStore } from '@/stores/auth'

// The nav menus (drawer, command palette, home links) filtered to the pages
// the signed-in user's role can open. A page's section and page key come from
// its route's `meta` (router/index.js); pages without a section are open to
// everyone.
export function useAllowedNav () {
  const auth = useAuthStore()
  const router = useRouter()

  function allowed (item) {
    if (!item.to) {return true}
    const { meta } = router.resolve(item.to)
    if (meta.adminOnly && !auth.isAdmin) {return false}
    return auth.canViewRoute(meta)
  }

  // A group with no visible links disappears too.
  const allowedNavItems = computed(() =>
    navItems
      .map(item => (item.children ? { ...item, children: item.children.filter(child => allowed(child)) } : item))
      .filter(item => (item.children ? item.children.length > 0 : allowed(item))),
  )
  const allowedAppendItems = computed(() => appendItems.filter(item => allowed(item)))

  // The rail's sections, each with only the pages the user can open; a
  // section with none disappears.
  const allowedSections = computed(() =>
    navSections
      .map(section => ({
        ...section,
        items: section.items
          .map(item => (item.children ? { ...item, children: item.children.filter(child => allowed(child)) } : item))
          .filter(item => (item.children ? item.children.length > 0 : allowed(item))),
      }))
      .filter(section => section.items.length > 0),
  )

  return { allowedNavItems, allowedAppendItems, allowedSections, allowed }
}
