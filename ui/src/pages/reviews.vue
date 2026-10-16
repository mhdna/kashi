<template>
  <div class="reviews d-flex flex-column fill-height">
    <div class="d-flex align-center flex-wrap flex-shrink-0 ga-3 pa-4 pb-2">
      <v-icon color="amber-darken-2" icon="mdi-star" />
      <div class="text-h6 font-weight-medium">Reviews</div>
      <v-spacer />
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
    <v-divider class="flex-shrink-0" />

    <div class="reviews__body">
      <ReviewsDashboard
        :branch-id="branchId"
        :from="range.from"
        :refresh-key="refreshKey"
        :to="range.to"
      />
    </div>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import ReviewsDashboard from '@/components/Reviews/ReviewsDashboard.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'
  import { addDays, ymd } from '@/utils/whatsapp'

  // Google reviews: how the review links in the post-sale WhatsApp message do
  // (sent, read, opened) and what Google shows for each branch's place.

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

  // Bumped to reload: Refresh, or something changing live.
  const refreshKey = ref(0)

  // A message sent, a review link opened, or Google's numbers moving; a
  // burst reloads once.
  const LIVE_TYPES = new Set(['whatsapp_message', 'review_click', 'google_reviews'])
  const { ensureConnected, onMessage } = useAdminSocket()
  let unsubscribe = null
  let liveTimer = 0
  onMounted(async () => {
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (!LIVE_TYPES.has(message.type)) return
      clearTimeout(liveTimer)
      liveTimer = setTimeout(() => refreshKey.value++, 1500)
    })
    try {
      await fetchBranches()
    } catch { /* the branch filter just stays at "All branches" */ }
  })
  onUnmounted(() => {
    unsubscribe?.()
    clearTimeout(liveTimer)
  })
</script>

<style scoped>
.reviews {
  min-height: 0;
}
.reviews__body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
}
</style>
