<template>
  <!-- A skeleton until the first refresh settles, so startup doesn't flash
       a green cloud and "syncing..." before the real state is known. -->
  <v-menu
    v-if="loaded"
    v-model="menu"
    :close-on-content-click="false"
    location="bottom center"
    :offset="8"
  >
    <template #activator="{ props }">
      <v-card
        v-bind="props"
        class="d-flex align-center me-2"
        :class="[mobile ? 'icon-pill' : 'py-1 px-2', pill && !mobile ? `activity-pill activity-pill--${pill.tint}` : '']"
        rounded="xl"
        style="cursor: pointer"
        :variant="pill && !mobile ? 'flat' : 'tonal'"
      >
        <v-icon
          v-if="mobile || !pill"
          :class="{ 'me-2': !mobile }"
          :color="currentActivity ? currentActivity.color : wsIconColor"
          :icon="currentActivity ? currentActivity.icon : 'mdi-cloud'"
          :size="mobile ? 20 : undefined"
        />
        <!-- One message at a time, like the Syncing pill: a ring spinner in
             its colour, its icon, the branch in bold and the short detail,
             and +N for the others still showing. Three side by side were
             each cut to "Test Branch: shift b...". -->
        <div v-if="!mobile" class="text-body-2 ticker">
          <Transition name="swap">
            <span v-if="pill" :key="`activity-${pill.id}`" class="activity-line" data-test="activity-line">
              <v-progress-circular :color="pill.color" indeterminate size="16" width="2" />
              <v-icon :color="pill.color" :icon="pill.icon" size="14" />
              <span class="activity-line__text">
                <strong v-if="pill.branch">{{ pill.branch }}</strong>
                {{ pill.detail }}
              </span>
              <span v-if="pill.more > 0" class="activity-line__more" data-test="activity-more">+{{ pill.more }}</span>
            </span>
          </Transition>
          <span class="status" :class="{ 'status--dim': pill }">{{ statusText }}</span>
        </div>
      </v-card>
    </template>

    <v-card class="menu-card" max-width="400" min-width="320">
      <v-card-title class="flex-shrink-0 text-subtitle-1">Branch sync</v-card-title>
      <v-divider />

      <v-card-text v-if="serverDown" class="text-medium-emphasis">
        {{ online ? 'Can\'t reach the server, we\'ll be back soon.' : NO_INTERNET_MESSAGE }}
      </v-card-text>
      <v-list v-else-if="recentInvoices.length > 0" class="menu-scroll" density="compact" lines="two">
        <v-list-item
          v-for="invoice in recentInvoices"
          :key="invoice.id"
          :prepend-icon="invoiceIcon(invoice)"
          :subtitle="`${branchName(invoice.branch_id)} — ${relativeTime(invoice.received_at)}`"
          :title="invoiceTitle(invoice)"
          @click="openInvoice(invoice)"
        />
      </v-list>
      <v-card-text v-else-if="loaded" class="text-medium-emphasis">
        No branch data has synced yet.
      </v-card-text>
      <v-card-text v-else class="text-medium-emphasis">
        Loading...
      </v-card-text>
    </v-card>
  </v-menu>
  <PillSkeleton v-else class="me-2" :width="96" />
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { useDisplay } from 'vuetify'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { NO_INTERNET_MESSAGE } from '@/composables/useApi'
  import { describeBranchActivity, isBranchActivityMessage } from '@/composables/useBranchActivityMessage'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchInvoices } from '@/composables/useBranchInvoices'
  import { useNetworkStatus } from '@/composables/useNetworkStatus'
  import { useAuthStore } from '@/stores/auth'
  import { useSettingsStore } from '@/stores/settings'
  import { activityPill } from '@/utils/activityPill'

  const { branches, fetchBranches } = useBranches()
  const { listRecentBranchInvoices } = useBranchInvoices()
  const { status: wsStatus, ensureConnected, onMessage } = useAdminSocket()
  const settingsStore = useSettingsStore()
  const auth = useAuthStore()
  const { mobile } = useDisplay()

  const menu = ref(false)
  const loaded = ref(false)
  const recentInvoices = ref([])
  // Wall-clock time of the last branch activity of ANY kind seen on the
  // socket -- invoices, but also visitor counts, attendance, shift closes,
  // none of which touch recentInvoices. "synced ... ago" is measured against
  // whichever is more recent, this or the newest invoice.
  const lastActivityAt = ref(0)
  // Ticks once a minute purely to force relativeTime()/statusText to
  // re-evaluate -- received_at/last_seen_at themselves only change on refresh.
  const now = ref(Date.now())

  // Live push is purely an optimization on top of the poll below -- if the
  // socket never connects or drops, the 30s poll still keeps this card
  // correct, just slower. wsStatus only drives the icon color.
  let unsubscribe = null

  let refreshTimer = null
  let clockTimer = null

  // No live socket means we can't confirm branch data is flowing, so the
  // card reports "not syncing" rather than a stale "synced ... ago". Only a
  // confirmed 'closed' counts -- 'connecting' is the normal pre-handshake
  // state right after a page load and should read as "syncing...", not an
  // outage.
  const { online } = useNetworkStatus()
  const serverDown = computed(() => !online.value || wsStatus.value === 'closed')
  const wsIconColor = computed(() => (serverDown.value ? undefined : 'success'))

  // 'appbar' mode: branch activity (sales/returns/expenses) rides on the
  // card's line. It's a sliding window of the most recent MAX_VISIBLE
  // messages, newest on the right -- a fresh message shows immediately and
  // bumps the oldest one out rather than waiting in a queue, so a burst
  // scrolls straight through. Each message also carries its own
  // activityMessageSeconds lifetime, so once activity stops the window
  // drains over that long and the card falls back to "synced ... ago". In
  // 'notification' mode (the default) BranchActivityToast owns this instead
  // and activeItems is never fed -- see the onMessage handler.
  const MAX_VISIBLE = 3
  const activeItems = ref([]) // [{ id, text, color, trendIcon, icon, groupKey, count }], oldest first

  // The newest visible activity -- drives the pill's leading icon.
  const currentActivity = computed(() => activeItems.value.at(-1) ?? null)
  const pill = computed(() => activityPill(activeItems.value))
  const itemTimers = new Map()
  let itemSeq = 0

  function dropItem (id) {
    clearTimeout(itemTimers.get(id))
    itemTimers.delete(id)
    const i = activeItems.value.findIndex(it => it.id === id)
    if (i !== -1) activeItems.value.splice(i, 1)
  }

  function pushActivity (message) {
    const described = describeBranchActivity(message, branchName)
    if (!described) return
    // Same group already showing (e.g. a burst of one branch's sync
    // conflicts): drop it and carry its count into the new, newest item.
    let count = 1
    const same = described.groupKey && activeItems.value.find(it => it.groupKey === described.groupKey)
    if (same) {
      count = same.count + 1
      dropItem(same.id)
    }
    const item = {
      id: ++itemSeq,
      text: count > 1 ? described.groupText(count) : (described.shortText ?? described.text),
      color: described.color,
      trendIcon: described.trendIcon,
      icon: described.icon,
      groupKey: described.groupKey,
      count,
    }
    activeItems.value.push(item)
    itemTimers.set(item.id, setTimeout(() => dropItem(item.id), settingsStore.activityMessageSeconds * 1000))
    // Newer message arrived while the window was full -- evict the oldest
    // now, no waiting.
    while (activeItems.value.length > MAX_VISIBLE) {
      dropItem(activeItems.value[0].id)
    }
  }

  // How many recent invoices the dropdown lists; .menu-scroll scrolls them.
  // The API caps page_size at 100.
  const RECENT_INVOICE_COUNT = 30

  async function refresh () {
    try {
      const [invoices] = await Promise.all([
        listRecentBranchInvoices(RECENT_INVOICE_COUNT),
        fetchBranches(true),
      ])
      recentInvoices.value = invoices
    } catch {
      // Sync status is informational -- a failed refresh just keeps
      // showing the last known state rather than surfacing an error.
    } finally {
      loaded.value = true
      now.value = Date.now()
    }
  }

  function branchName (branchId) {
    return branches.value.find(b => b.id === branchId)?.name ?? `Branch #${branchId}`
  }

  // A sale in the list opens on the Invoices page, searched and with its
  // details showing.
  const router = useRouter()
  function openInvoice (invoice) {
    menu.value = false
    router.push({ path: '/invoices', query: { search: invoice.branch_invoice_code, open: invoice.id } })
  }

  function invoiceIcon (invoice) {
    if (invoice.kind === 'exchange') return 'mdi-swap-horizontal'
    if (invoice.kind === 'return') return 'mdi-transfer'
    return 'mdi-invoice'
  }

  function invoiceKindLabel (invoice) {
    if (invoice.kind === 'exchange') return 'Exchange'
    if (invoice.kind === 'return') return 'Return'
    return 'Sale'
  }

  function invoiceTitle (invoice) {
    return `${invoiceKindLabel(invoice)} ${invoice.branch_invoice_code}`
  }

  // Accepts an ISO string or epoch ms.
  function relativeTime (value) {
    const then = new Date(value).getTime()
    const diffSeconds = Math.max(0, Math.round((now.value - then) / 1000))

    if (diffSeconds < 45) return 'just now'
    const diffMinutes = Math.round(diffSeconds / 60)
    if (diffMinutes < 60) return `${diffMinutes} min${diffMinutes === 1 ? '' : 's'} ago`
    const diffHours = Math.round(diffMinutes / 60)
    if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`
    const diffDays = Math.round(diffHours / 24)
    return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`
  }

  // Compact form for the pill: "now", "5m ago", "1h ago", "2d ago". The cloud
  // icon already says it's about syncing, so no "synced" prefix.
  function shortTime (value) {
    const diffSeconds = Math.max(0, Math.round((now.value - new Date(value).getTime()) / 1000))
    if (diffSeconds < 45) return 'now'
    const diffMinutes = Math.round(diffSeconds / 60)
    if (diffMinutes < 60) return `${diffMinutes}m ago`
    const diffHours = Math.round(diffMinutes / 60)
    if (diffHours < 24) return `${diffHours}h ago`
    return `${Math.round(diffHours / 24)}d ago`
  }

  // Most recent proof of sync: newest invoice's received_at vs. the last
  // live activity ping, in ms. 0 means nothing has synced this session.
  const lastSyncMs = computed(() => {
    const latest = recentInvoices.value[0]
    const invoiceMs = latest ? new Date(latest.received_at).getTime() : 0
    return Math.max(invoiceMs, lastActivityAt.value)
  })

  const statusText = computed(() => {
    if (!online.value) return 'offline'
    if (serverDown.value) return 'not syncing'
    if (!loaded.value) return 'syncing...'
    if (!lastSyncMs.value) return 'no sync yet'
    return shortTime(lastSyncMs.value)
  })

  onMounted(() => {
    refresh()
    refreshTimer = setInterval(refresh, 30_000)
    clockTimer = setInterval(() => {
      now.value = Date.now()
    }, 30_000)
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (message.type === 'branch_invoice_created') {
        refresh()
      }
      if (isBranchActivityMessage(message)) {
        // Any activity means data is flowing right now -- keep "synced ..."
        // honest even when the branch is only sending visitor counts.
        lastActivityAt.value = Date.now()
        now.value = Date.now()
      }
      // Sync conflicts are only actionable by admins, so only they see them
      // ride on the card.
      if (message.type === 'branch_sync_conflict' && !auth.isAdmin) {return}
      if (settingsStore.activityDisplayMode === 'appbar' && isBranchActivityMessage(message)) {
        pushActivity(message)
      }
    })
  })

  onUnmounted(() => {
    clearInterval(refreshTimer)
    clearInterval(clockTimer)
    for (const timer of itemTimers.values()) clearTimeout(timer)
    unsubscribe?.()
  })
</script>

<style scoped>
/* On mobile the pill shrinks to just its icon -- fix it to a square that
   matches ConnectedBranchesCard's icon-only pill. */
.icon-pill {
  width: 34px;
  height: 34px;
  justify-content: center;
}

/* Cap the dropdown height; the list scrolls inside. */
.menu-card {
  display: flex;
  flex-direction: column;
  max-height: min(60vh, 420px);
}

.menu-scroll {
  overflow-y: auto;
}

/* The message and the status line are stacked in a single grid cell so they
   can crossfade in place. The pill hugs whichever is wider -- its status
   text, or the messages -- up to max-width; each message is capped at a
   third of that and ellipsized past it. */
.ticker {
  display: grid;
  grid-template: 1fr / 1fr;
  place-items: center;
  min-height: 24px;
  overflow: hidden;
  white-space: nowrap;
  max-width: min(44vw, 520px);
}

.ticker > * {
  grid-area: 1 / 1;
}

/* The tint follows the message, like the Syncing pill's soft info tint. */
.activity-pill--success { background: rgba(var(--v-theme-success), 0.14); border: 1px solid rgba(var(--v-theme-success), 0.28); }
.activity-pill--error { background: rgba(var(--v-theme-error), 0.14); border: 1px solid rgba(var(--v-theme-error), 0.28); }
.activity-pill--warning { background: rgba(var(--v-theme-warning), 0.14); border: 1px solid rgba(var(--v-theme-warning), 0.28); }
.activity-pill--info { background: rgba(var(--v-theme-info), 0.14); border: 1px solid rgba(var(--v-theme-info), 0.28); }
.activity-pill { transition: background-color 0.24s ease, border-color 0.24s ease; }

.activity-line {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  padding: 0 4px;
}

.activity-line__text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.activity-line__text strong {
  font-weight: 600;
}

.activity-line__more {
  flex: 0 0 auto;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0 7px;
  border-radius: 999px;
  background: rgba(var(--v-theme-on-surface), 0.1);
}

/* The newest message crossfades in over the last, in place. */
.swap-enter-active,
.swap-leave-active {
  transition: opacity 0.24s ease, transform 0.24s ease;
}

.swap-enter-from {
  opacity: 0;
  transform: translateY(6px);
}

.swap-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

@media (prefers-reduced-motion: reduce) {
  .swap-enter-active,
  .swap-leave-active { transition: none; }
}
</style>
