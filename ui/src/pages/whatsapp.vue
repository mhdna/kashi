<template>
  <div class="wa d-flex flex-column fill-height">
    <div class="d-flex align-center flex-wrap flex-shrink-0 ga-3 pa-4 pb-2">
      <v-icon color="success" icon="mdi-whatsapp" />
      <div class="text-h6 font-weight-medium">WhatsApp &amp; Reviews</div>
      <v-chip
        v-if="settings"
        :color="settings.whatsapp_enabled ? 'success' : undefined"
        size="small"
        :text="settings.whatsapp_enabled ? 'Sending after sales' : 'Off in Settings'"
        to="/settings?tab=whatsapp"
        variant="tonal"
      />
      <v-spacer />
      <!-- Hidden, not removed, on Rules: the header keeps its height so the
           tabs don't jump when switching. -->
      <div class="d-flex align-center flex-wrap ga-3" :class="{ 'wa__filters--hidden': tab === 'rules' || tab === 'health' }">
        <v-select
          v-model="preset"
          density="compact"
          hide-details
          :items="PRESETS"
          style="max-width: 170px"
          variant="outlined"
        />
        <template v-if="preset === 'custom'">
          <v-text-field
            v-model="customFrom"
            density="compact"
            hide-details
            label="From"
            style="max-width: 160px"
            type="date"
            variant="outlined"
          />
          <v-text-field
            v-model="customTo"
            density="compact"
            hide-details
            label="To"
            style="max-width: 160px"
            type="date"
            variant="outlined"
          />
        </template>
        <v-select
          v-model="branchId"
          density="compact"
          hide-details
          :items="branchOptions"
          style="min-width: 170px; max-width: 240px"
          variant="outlined"
        />
        <v-btn icon="mdi-refresh" title="Refresh" variant="text" @click="refreshKey++" />
      </div>
    </div>

    <v-tabs v-model="tab" class="flex-shrink-0 px-4" color="primary" density="compact">
      <v-tab prepend-icon="mdi-view-dashboard" text="Dashboard" value="overview" />
      <v-tab prepend-icon="mdi-format-list-bulleted" text="Messages" value="messages" />
      <v-tab prepend-icon="mdi-reply" text="Replies" value="replies" />
      <v-tab prepend-icon="mdi-filter-variant" text="Rules" value="rules" />
      <v-tab prepend-icon="mdi-heart-pulse" value="health">
        Health
        <v-badge
          v-if="healthAlert"
          class="ms-2"
          color="error"
          dot
          inline
        />
      </v-tab>
    </v-tabs>
    <v-divider class="flex-shrink-0" />

    <div class="wa__body">
      <!-- One dashboard: the messages, then what clients thought of the visit. -->
      <div v-if="tab === 'overview'">
        <WhatsAppOverview
          :branch-id="branchId"
          :from="range.from"
          :meta="sender === 'meta'"
          :refresh-key="refreshKey"
          :to="range.to"
          @show-messages="showMessages"
          @show-replies="tab = 'replies'"
        />
        <div class="wa__section-title px-4 pt-2">Reviews</div>
        <ReviewsDashboard
          :branch-id="branchId"
          :from="range.from"
          :refresh-key="refreshKey"
          :to="range.to"
        />
      </div>
      <WhatsAppMessages
        v-else-if="tab === 'messages'"
        v-model:search="messageSearch"
        v-model:status="messageStatus"
        :branch-id="branchId"
        :from="range.from"
        :meta="sender === 'meta'"
        :refresh-key="refreshKey"
        :to="range.to"
      />
      <WhatsAppReplies
        v-else-if="tab === 'replies'"
        :branch-id="branchId"
        :from="range.from"
        :refresh-key="refreshKey"
        :to="range.to"
      />
      <WhatsAppHealth
        v-else-if="tab === 'health'"
        :refresh-key="refreshKey"
        @alert="healthAlert = $event"
      />
      <WhatsAppRules
        v-else
        :branches="branches"
        :settings="settings"
      />
    </div>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import ReviewsDashboard from '@/components/Reviews/ReviewsDashboard.vue'
  import WhatsAppHealth from '@/components/WhatsApp/WhatsAppHealth.vue'
  import WhatsAppMessages from '@/components/WhatsApp/WhatsAppMessages.vue'
  import WhatsAppOverview from '@/components/WhatsApp/WhatsAppOverview.vue'
  import WhatsAppReplies from '@/components/WhatsApp/WhatsAppReplies.vue'
  import WhatsAppRules from '@/components/WhatsApp/WhatsAppRules.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { useWhatsAppSettings } from '@/composables/useWhatsAppSettings'
  import { addDays, ymd } from '@/utils/whatsapp'

  // Post-sale WhatsApp messages: what went out, to whom and what it cost
  // (Overview, Messages), and the rules deciding it (Rules). The Cloud API
  // credentials and default template stay on Settings > WhatsApp.

  // The open tab, kept in the address (?tab=) so a link or a reload opens it;
  // the old Reviews page lands on ?tab=reviews.
  const TABS = ['overview', 'messages', 'replies', 'rules', 'health']
  const route = useRoute()
  const router = useRouter()
  // Reviews are part of the dashboard now; an old ?tab=reviews link lands there.
  const tab = ref(TABS.includes(route.query.tab) ? route.query.tab : 'overview')
  watch(tab, value => router.replace({ query: { ...route.query, tab: value } }))
  // A number at risk or paused: a red dot on the Health tab.
  const healthAlert = ref(false)
  const { getHealth, getSender } = useWhatsApp()

  const PRESETS = [
    { title: 'Today', value: 'today' },
    { title: 'Last 7 days', value: '7' },
    { title: 'Last 30 days', value: '30' },
    { title: 'Last 90 days', value: '90' },
    { title: 'This month', value: 'month' },
    { title: 'Last month', value: 'last-month' },
    { title: 'This year', value: 'year' },
    { title: 'Custom', value: 'custom' },
  ]
  const preset = ref('30')
  const customFrom = ref(ymd(addDays(new Date(), -29)))
  const customTo = ref(ymd(new Date()))

  const range = computed(() => {
    const today = new Date()
    switch (preset.value) {
      case 'today': {
        return { from: ymd(today), to: ymd(today) }
      }
      case 'month': {
        return { from: ymd(new Date(today.getFullYear(), today.getMonth(), 1)), to: ymd(today) }
      }
      case 'last-month': {
        return {
          from: ymd(new Date(today.getFullYear(), today.getMonth() - 1, 1)),
          to: ymd(new Date(today.getFullYear(), today.getMonth(), 0)),
        }
      }
      case 'year': {
        return { from: ymd(new Date(today.getFullYear(), 0, 1)), to: ymd(today) }
      }
      case 'custom': {
        const from = customFrom.value || ymd(today)
        const to = customTo.value || from
        return from <= to ? { from, to } : { from: to, to: from }
      }
      default: {
        return { from: ymd(addDays(today, -(Number(preset.value) - 1))), to: ymd(today) }
      }
    }
  })

  const { branches, fetchBranches } = useBranches()
  const branchId = ref(0)
  const branchOptions = computed(() => [
    { title: 'All branches', value: 0 },
    ...branches.value.map(b => ({ title: b.name, value: b.id })),
  ])

  const settings = ref(null)
  const { getWhatsAppSettings } = useWhatsAppSettings()

  // Who sends (set in Settings > WhatsApp): Meta's costs only show with Meta.
  const sender = ref('')

  // Bumped to make the open tab reload: Refresh, or a message logged live.
  const refreshKey = ref(0)

  const messageStatus = ref('')
  const messageSearch = ref('')
  function showMessages ({ status = '', search = '' } = {}) {
    messageStatus.value = status
    messageSearch.value = search
    tab.value = 'messages'
  }

  // Every message kashi decides on is pushed as whatsapp_message; a burst
  // after a busy minute reloads once.
  const { ensureConnected, onMessage } = useAdminSocket()
  let unsubscribe = null
  const LIVE_TYPES = new Set(['whatsapp_message', 'review_click', 'google_reviews'])
  let liveTimer = 0
  let pollTimer = 0
  onMounted(async () => {
    ensureConnected()
    getHealth()
      .then(data => {
        healthAlert.value = (data.numbers ?? []).some(n => n.health?.paused_at || ['warning', 'high'].includes(n.health?.level))
      })
      .catch(() => {})
    // whatsapp_message: a message logged or its ticks moved; review_click: a
    // vote, a reply or a review link opened; google_reviews: new Google ones.
    unsubscribe = onMessage(message => {
      if (!LIVE_TYPES.has(message.type)) return
      clearTimeout(liveTimer)
      liveTimer = setTimeout(() => refreshKey.value++, 1500)
    })
    // And every minute while the page is in view, in case the live
    // connection dropped.
    pollTimer = setInterval(() => {
      if (document.visibilityState === 'visible') refreshKey.value++
    }, 60_000)
    try {
      await fetchBranches()
    } catch { /* the branch filter just stays at "All branches" */ }
    try {
      settings.value = await getWhatsAppSettings()
    } catch { /* no status chip */ }
    try {
      sender.value = (await getSender()).settings?.whatsapp_sender || 'bridge'
    } catch { /* no switch */ }
  })
  onUnmounted(() => {
    unsubscribe?.()
    clearTimeout(liveTimer)
    clearInterval(pollTimer)
  })
</script>

<style scoped>
.wa {
  min-height: 0;
}
.wa__section-title {
  font-size: 1rem;
  font-weight: 600;
}
.wa__filters--hidden {
  visibility: hidden;
}
.wa__body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
}
</style>
