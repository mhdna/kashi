<template>
  <v-navigation-drawer
    class="nav-drawer"
    density="compact"
    location="left"
    :model-value="props.mobile ? props.modelValue : true"
    :permanent="!props.mobile"
    :temporary="props.mobile"
    :width="drawerWidth"
    @update:model-value="props.mobile && $emit('update:modelValue', $event)"
  >
    <div class="nav-shell">
      <!-- The rail: one icon per section, always there. -->
      <nav aria-label="Sections" class="nav-rail" data-test="nav-rail">
        <v-tooltip
          v-for="section in sections"
          :key="section.key"
          :disabled="railLabel(section) === section.title"
          location="end"
          :text="section.title"
        >
          <template #activator="{ props: tip }">
            <button
              v-bind="tip"
              :aria-current="section.key === openKey ? 'true' : undefined"
              :aria-label="section.title"
              class="nav-rail-btn"
              :class="{ 'nav-rail-btn--open': section.key === openKey }"
              :data-test="`rail-${section.key}`"
              type="button"
              @click="openSection(section.key)"
            >
              <v-icon :icon="section.icon" size="22" />
              <span class="nav-rail-label" :data-test="`rail-label-${section.key}`">{{ railLabel(section) }}</span>
              <span v-if="needsAttention(section)" class="nav-rail-dot" :data-test="`rail-dot-${section.key}`" />
            </button>
          </template>
        </v-tooltip>
      </nav>

      <!-- The open section's pages. -->
      <div v-if="!props.rail && openSectionInfo" class="nav-pages" data-test="nav-pages">
        <div class="nav-pages-title" data-test="nav-section-title">{{ openSectionInfo.title }}</div>
        <v-list density="compact" nav>
          <template v-for="item in openSectionInfo.items" :key="item.title">
            <template v-if="item.children">
              <v-list-subheader class="nav-heading">{{ item.title }}</v-list-subheader>
              <NavLink
                v-for="child in item.children"
                :key="child.to"
                :alert-counts="alertCounts"
                :item="child"
                :security-count="openSecurityAlerts"
                :status="statusFor(child.append)"
              />
            </template>
            <NavLink
              v-else
              :alert-counts="alertCounts"
              :item="item"
              :security-count="openSecurityAlerts"
              :status="statusFor(item.append)"
            />
          </template>
        </v-list>
      </div>
    </div>
  </v-navigation-drawer>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import NavLink from '@/components/NavLink.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useAllowedNav } from '@/composables/useAllowedNav'
  import { useBranchCommands } from '@/composables/useBranchCommands'
  import { useBranchSyncConflicts } from '@/composables/useBranchSyncConflicts'
  import { useLowStockAlerts } from '@/composables/useLowStockAlerts'
  import { useSecurityAlerts } from '@/composables/useSecurityAlerts'
  import { useStockHealth } from '@/composables/useStockHealth'
  import { useWhatsAppFailures } from '@/composables/useWhatsAppFailures'
  import { navPages, railLabel, sectionForPath } from '@/data/navItems'
  import { useAuthStore } from '@/stores/auth'

  const props = defineProps({
    modelValue: { type: Boolean, required: true },
    rail: { type: Boolean, default: false },
    mobile: { type: Boolean, default: false },
  })

  // Links to pages the user's role can't open are hidden (see useAllowedNav).
  const auth = useAuthStore()
  const { allowedSections: sections, allowed } = useAllowedNav()

  // The open section follows the page; a click on the rail opens another.
  const route = useRoute()
  const openKey = ref(sectionForPath(sections.value, route.path) ?? sections.value[0]?.key ?? '')
  watch(() => route.path, path => {
    const key = sectionForPath(sections.value, path)
    if (key) {openKey.value = key}
  })
  const openSectionInfo = computed(() => sections.value.find(s => s.key === openKey.value) ?? sections.value[0] ?? null)
  const emit = defineEmits(['update:modelValue', 'update:rail'])
  function openSection (key) {
    openKey.value = key
    // A collapsed drawer opens to show the section's pages.
    if (props.rail) {emit('update:rail', false)}
  }

  const RAIL_WIDTH = 84
  const drawerWidth = computed(() => (props.rail ? RAIL_WIDTH : RAIL_WIDTH + 236))

  // Real counts backing the Alerts nav badges, shown separately: red for
  // out-of-stock, orange for merely low (see pages/alerts.vue, api/low_stock.go).
  const { fetchSummary } = useLowStockAlerts()
  const outOfStockCount = ref(0)
  const lowStockCount = ref(0)
  // Until the first summary lands the Alerts row shows a placeholder dot.
  const alertsLoaded = ref(false)

  onMounted(async () => {
    if (!auth.canView('inventory')) {
      alertsLoaded.value = true
      return
    }
    try {
      const summary = await fetchSummary()
      outOfStockCount.value = summary?.out_of_stock_count ?? 0
      lowStockCount.value = summary?.low_stock_count ?? 0
    } catch {
      outOfStockCount.value = 0
      lowStockCount.value = 0
    } finally {
      alertsLoaded.value = true
    }
  })

  // Real status backing the Stock Health nav dot -- green/orange/red for
  // healthy/warning/critical (see pages/stock-health.vue, api/stock_health.go).
  // 'loading' (a grey placeholder dot) until the check comes back; null (no
  // dot) if it fails.
  const { fetchStockHealth } = useStockHealth()
  const healthStatus = ref('loading')

  onMounted(async () => {
    if (!auth.canView('inventory')) {
      healthStatus.value = null
      return
    }
    try {
      const health = await fetchStockHealth()
      healthStatus.value = health?.status ?? null
    } catch {
      healthStatus.value = null
    }
  })

  // Commands (green: every queued command ran, yellow: some still waiting
  // for a till) and Conflicts (green / yellow open / red oversold) dots --
  // see pages/branch-commands.vue and pages/branch-conflicts.vue. Polled,
  // plus a nudge when kashi reports a new conflict.
  const { commandsStatus, refreshCommandSummary } = useBranchCommands()
  const { conflictsStatus, refreshConflictSummary } = useBranchSyncConflicts()
  // WhatsApp sale messages that failed lately (useWhatsAppFailures).
  const { whatsAppFailureStatus, refreshWhatsAppFailures } = useWhatsAppFailures()
  const canSeeWhatsApp = () => allowed({ to: '/whatsapp' })
  function refreshWhatsAppStatus () {
    if (!canSeeWhatsApp()) {return}
    refreshWhatsAppFailures().catch(() => {})
  }
  const { onMessage: onAdminMessage, ensureConnected: ensureAdminSocket } = useAdminSocket()

  // Security nav badge: open (unacknowledged) alerts, refetched when kashi
  // reports a new one. Only for those who can open the Security page.
  const { openSecurityAlerts, refreshOpenSecurityAlerts } = useSecurityAlerts()
  const canSeeSecurity = () => allowed({ to: '/security' })
  function refreshSecurityAlerts () {
    if (!canSeeSecurity()) {return}
    refreshOpenSecurityAlerts().catch(() => {})
  }
  // The 30s branch-status poll also refetches the badge, in case a
  // security_alert message was missed while the socket was down.
  function refreshStatus () {
    refreshBranchStatus()
    refreshSecurityAlerts()
    refreshWhatsAppStatus()
  }

  function refreshBranchStatus () {
    if (!auth.canView('branches')) {return}
    refreshCommandSummary().catch(() => {})
    refreshConflictSummary().catch(() => {})
  }

  let branchStatusPoll = null
  let conflictNudge = null
  let whatsAppNudge = null
  let offAdminMessage = null
  onMounted(() => {
    refreshStatus()
    branchStatusPoll = setInterval(refreshStatus, 30_000)
    if (canSeeSecurity() || canSeeWhatsApp()) {ensureAdminSocket()}
    // A sync can stage hundreds of conflicts at once; one refetch per burst.
    offAdminMessage = onAdminMessage((msg) => {
      if (msg?.type === 'security_alert') {
        refreshSecurityAlerts()
        return
      }
      if (msg?.type === 'whatsapp_message') {
        if (!whatsAppNudge) {
          whatsAppNudge = setTimeout(() => {
            whatsAppNudge = null
            refreshWhatsAppStatus()
          }, 2000)
        }
        return
      }
      if (msg?.type !== 'branch_sync_conflict' || conflictNudge) {return}
      conflictNudge = setTimeout(() => {
        conflictNudge = null
        refreshConflictSummary().catch(() => {})
      }, 2000)
    })
  })
  onUnmounted(() => {
    clearInterval(branchStatusPoll)
    clearTimeout(conflictNudge)
    clearTimeout(whatsAppNudge)
    offAdminMessage?.()
  })

  const alertCounts = computed(() => ({ loaded: alertsLoaded.value, out: outOfStockCount.value, low: lowStockCount.value }))

  // A section's icon gets a dot when one of its pages wants a look: stock
  // running out, a warning or critical status, an open security alert.
  function needsAttention (section) {
    return navPages(section).some((page) => {
      const status = statusFor(page.append)
      if (status === 'warning' || status === 'critical') {return true}
      if (page.append === 'alert-count') {return outOfStockCount.value > 0}
      return page.to === '/security' && openSecurityAlerts.value > 0
    })
  }

  function statusFor (append) {
    if (append === 'status') {return healthStatus.value}
    if (append === 'commands-status') {return commandsStatus.value}
    if (append === 'conflicts-status') {return conflictsStatus.value}
    if (append === 'whatsapp-status') {return canSeeWhatsApp() ? whatsAppFailureStatus.value : null}
    return null
  }
</script>

<style scoped>
.nav-shell {
  display: flex;
  height: 100%;
}
.nav-rail {
  flex: 0 0 84px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 0;
  overflow-y: auto;
  border-inline-end: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.nav-rail-btn {
  position: relative;
  width: 76px;
  min-height: 54px;
  padding: 6px 2px 5px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: rgba(var(--v-theme-on-surface), 0.7);
  cursor: pointer;
}
.nav-rail-btn:hover {
  color: rgb(var(--v-theme-on-surface));
  background: rgba(var(--v-theme-on-surface), 0.06);
}
/* The open section: its icon in the primary colour with a bar beside it. */
.nav-rail-btn--open {
  color: rgb(var(--v-theme-primary));
}
.nav-rail-btn--open::before {
  content: '';
  position: absolute;
  left: -4px;
  top: 8px;
  bottom: 8px;
  width: 3px;
  border-radius: 2px;
  background: rgb(var(--v-theme-primary));
}
.nav-rail-label {
  max-width: 100%;
  overflow: hidden;
  font-size: 0.6875rem;
  font-weight: 500;
  line-height: 1.15;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nav-rail-dot {
  position: absolute;
  top: 6px;
  right: 20px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgb(var(--v-theme-error));
}
.nav-pages {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
}
.nav-pages-title {
  padding: 14px 16px 4px;
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.nav-heading {
  min-height: 28px;
  font-size: 0.75rem;
}
</style>
