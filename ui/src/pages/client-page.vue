<script setup>
  import { ref, computed, onMounted } from 'vue'
  import { useRoute } from 'vue-router'
  import VChart from '@/components/EChart.vue'
  import ClientRankTag from '@/components/ClientRankTag.vue'
  import { API_BASE } from '@/config'
  import { authFetch } from '@/composables/useApi'
  import { formatMoney } from '@/utils/money'
  import { useNotify } from '@/composables/useNotify'

  const route = useRoute()
  const clientId = Number(route.params.id)
  const notify = useNotify()

  // ---------------------------------------------------------------------------
  // Data
  // ---------------------------------------------------------------------------
  const profile   = ref(null)
  const loading   = ref(true)

  async function fetchProfile () {
    loading.value = true
    try {
      const res = await authFetch(`${API_BASE}/clients/${clientId}/profile`)
      if (!res.ok) throw new Error(`${res.status}`)
      profile.value = await res.json()
    } catch (e) {
      notify.error(e?.message ?? 'Failed to load profile')
    } finally {
      loading.value = false
    }
  }

  onMounted(fetchProfile)

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------
  const money = n => '$ ' + formatMoney(Number(n || 0))

  function formatDate (isoString) {
    if (!isoString) return ''
    const d = new Date(isoString)
    return isNaN(d) ? '' : d.toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' })
  }

  // ---------------------------------------------------------------------------
  // Derived display values
  // ---------------------------------------------------------------------------
  const client  = computed(() => profile.value?.client   ?? null)
  const initials = computed(() =>
    (client.value?.name ?? '')
      .split(' ')
      .map(w => w[0] ?? '')
      .join('')
      .slice(0, 2)
      .toUpperCase()
  )

  const tier = computed(() => {
    const pts = Number(client.value?.totalLoyaltyPoints ?? 0)
    if (pts >= 500) return 'Gold Tier'
    if (pts >= 200) return 'Silver Tier'
    if (pts >= 1)   return 'Bronze Tier'
    return 'None'
  })

  const meta = computed(() => {
    if (!profile.value) return []
    return [
      { label: 'Client ID',     value: client.value?.uid    ?? '—' },
      { label: 'Phone',         value: client.value?.phone  ?? '—' },
      { label: 'Home branch',   value: profile.value.homeBranch  || '—' },
      { label: 'First purchase', value: formatDate(profile.value.firstPurchaseAt) || '—' },
    ]
  })

  const statCards = computed(() => {
    if (!profile.value) return []
    return [
      { title: 'Money Spent',      value: money(profile.value.totalSpentCents) },
      { title: 'Items Bought',     value: Number(profile.value.itemsBought).toLocaleString() },
      { title: 'Points Remaining', value: Number(profile.value.pointsRemaining).toLocaleString() },
      { title: 'Points Redeemed',  value: Number(profile.value.pointsRedeemed).toLocaleString() },
    ]
  })

  // Loyalty tier progress
  const TIERS = [
    { name: 'None',   min: 0,   next: 'Bronze',   nextMin: 1   },
    { name: 'Bronze', min: 1,   next: 'Silver',   nextMin: 200 },
    { name: 'Silver', min: 200, next: 'Gold',     nextMin: 500 },
    { name: 'Gold',   min: 500, next: 'Platinum', nextMin: 1000 },
  ]

  const currentTierDef = computed(() => {
    const pts = Number(client.value?.totalLoyaltyPoints ?? 0)
    const found = [...TIERS].reverse().find(t => pts >= t.min)
    return found ?? TIERS[0]
  })

  const tierProgress = computed(() => {
    const pts  = Number(client.value?.totalLoyaltyPoints ?? 0)
    const t    = currentTierDef.value
    if (t.name === 'Gold') return 100
    const range = t.nextMin - t.min
    return Math.min(100, Math.round(((pts - t.min) / range) * 100))
  })

  const ptsToNext = computed(() => {
    const pts = Number(client.value?.totalLoyaltyPoints ?? 0)
    const t   = currentTierDef.value
    if (t.name === 'Gold') return 0
    return Math.max(0, t.nextMin - pts)
  })

  // ---------------------------------------------------------------------------
  // Payment mix donut (ECharts)
  // ---------------------------------------------------------------------------
  const DONUT_COLORS = ['#5c6bc0', '#8bc34a', '#78909c', '#ffa726', '#ec407a', '#26c6da']

  const paymentOption = computed(() => {
    const mix = (profile.value?.paymentMix ?? [])
    if (!mix.length) return {}
    const data = mix.map((row, i) => ({
      value: row.totalCents,
      name:  row.name,
      itemStyle: { color: DONUT_COLORS[i % DONUT_COLORS.length] },
    }))
    return {
      backgroundColor: 'transparent',
      tooltip: { trigger: 'item', appendTo: 'body', formatter: '{b}: {d}%' },
      legend: { bottom: 0, left: 'center' },
      series: [{
        name: 'Payment mix',
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['50%', '44%'],
        avoidLabelOverlap: false,
        itemStyle: { borderColor: 'rgba(var(--v-theme-surface), 1)', borderWidth: 2 },
        label: { show: false },
        labelLine: { show: false },
        data,
      }],
    }
  })

  const hasPaymentData = computed(() => (profile.value?.paymentMix ?? []).length > 0)

  // ---------------------------------------------------------------------------
  // Recent purchases
  // ---------------------------------------------------------------------------
  const purchaseHeaders = [
    { title: 'Date',    key: 'date'    },
    { title: 'Invoice', key: 'invoice' },
    { title: 'Branch',  key: 'branch'  },
    { title: 'Items',   key: 'items',  align: 'end' },
    { title: 'Payment', key: 'payment' },
    { title: 'Total',   key: 'total',  align: 'end' },
  ]

  const purchases = computed(() =>
    (profile.value?.recentInvoices ?? []).map(inv => ({
      date:    formatDate(inv.occurredAt),
      invoice: inv.invoiceCode,
      branch:  inv.branchName,
      items:   inv.itemCount,
      payment: inv.paymentMethods,
      total:   inv.grandTotal,
    }))
  )

  const paymentColor = {
    'Card':          'indigo',
    'Cash':          'teal',
    'Store Credit':  'deep-purple',
    'Loyalty Points':'amber-darken-2',
  }
</script>

<template>
  <div class="client-page pa-4">
    <v-progress-linear v-if="loading" indeterminate class="mb-4" />

    <template v-else-if="profile">
      <!-- Row 1: client header (left) + stat cards (right) -->
      <div class="profile-top-row mb-4">
        <!-- Client header card -->
        <v-sheet class="client-header pa-6" color="surface-light" rounded="lg">
          <div class="d-flex ga-6 align-center">
            <v-avatar class="text-white flex-shrink-0" color="blue-darken-4" size="112">
              <span class="text-h3">{{ initials }}</span>
            </v-avatar>

            <div class="flex-grow-1">
              <div class="d-flex flex-wrap align-center ga-3 mb-3">
                <span class="text-h5 font-weight-bold">{{ client.name }}</span>
                <ClientRankTag :rank="tier" />
              </div>
              <div class="meta-grid">
                <template v-for="row in meta" :key="row.label">
                  <span class="text-body-2 text-medium-emphasis">{{ row.label }}</span>
                  <span class="text-body-2">{{ row.value }}</span>
                </template>
              </div>
            </div>
          </div>
        </v-sheet>

        <!-- Stat cards (2×2 grid, no sparklines) -->
        <div class="stat-grid">
          <v-card
            v-for="card in statCards"
            :key="card.title"
            class="stat-card pa-4"
            rounded="lg"
            variant="outlined"
          >
            <div class="text-body-2 text-medium-emphasis text-uppercase mb-2">{{ card.title }}</div>
            <div class="text-h5 font-weight-black">{{ card.value }}</div>
          </v-card>
        </div>
      </div>

      <!-- Row 2: payment mix + recent purchases -->
      <v-row>
        <v-col v-if="hasPaymentData" cols="12" md="5">
          <v-card class="fill-height" flat>
            <v-card-title class="card-title">Payment Mix</v-card-title>
            <VChart autoresize class="donut" :option="paymentOption" />
          </v-card>
        </v-col>

        <v-col cols="12" :md="hasPaymentData ? 7 : 12">
          <v-card flat>
            <v-card-title class="card-title">Recent Purchases</v-card-title>
            <v-data-table
              density="compact"
              :headers="purchaseHeaders"
              :items="purchases"
              :items-per-page="8"
            >
              <template #item.payment="{ value }">
                <v-chip
                  v-for="m in (Array.isArray(value) ? value : [value])"
                  :key="m"
                  :color="paymentColor[m]"
                  class="mr-1"
                  label
                  size="small"
                  variant="tonal"
                >{{ m }}</v-chip>
              </template>
              <template #item.total="{ value }">
                {{ value === 0 ? '—' : money(value) }}
              </template>
            </v-data-table>
          </v-card>
        </v-col>
      </v-row>

      <!-- Row 3: loyalty -->
      <v-row>
        <v-col cols="12" md="5">
          <v-card class="pa-4" flat>
            <v-card-title class="card-title px-0">Loyalty</v-card-title>

            <div class="d-flex align-baseline justify-space-between mb-1">
              <span class="text-body-2 text-medium-emphasis">{{ tier }}</span>
              <span class="text-body-2 text-medium-emphasis">{{ currentTierDef.next }} Tier</span>
            </div>
            <v-progress-linear
              color="amber-darken-2"
              height="10"
              :model-value="tierProgress"
              rounded
            />
            <div class="text-caption text-medium-emphasis mt-1">
              <span v-if="ptsToNext > 0">
                {{ ptsToNext.toLocaleString() }} pts to {{ currentTierDef.next }} Tier
              </span>
              <span v-else class="text-green">Max tier reached</span>
            </div>

            <v-divider class="my-4" />

            <v-row dense>
              <v-col cols="6">
                <div class="text-h6 font-weight-bold">{{ money(profile.totalSpentCents) }}</div>
                <div class="text-caption text-medium-emphasis">Lifetime spend</div>
              </v-col>
              <v-col cols="6">
                <div class="text-h6 font-weight-bold">{{ Number(profile.pointsRemaining).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis">Points balance</div>
              </v-col>
              <v-col cols="6">
                <div class="text-h6 font-weight-bold">{{ Number(profile.itemsBought).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis">Items bought</div>
              </v-col>
              <v-col cols="6">
                <div class="text-h6 font-weight-bold">{{ Number(profile.pointsRedeemed).toLocaleString() }}</div>
                <div class="text-caption text-medium-emphasis">Points redeemed</div>
              </v-col>
            </v-row>
          </v-card>
        </v-col>
      </v-row>
    </template>
  </div>
</template>

<style scoped>
  /* Header + stat cards side by side */
  .profile-top-row {
    display: flex;
    gap: 16px;
    align-items: stretch;
  }

  .client-header {
    flex: 3;
    min-width: 0;
  }

  .stat-grid {
    flex: 2;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    min-width: 0;
  }

  .stat-card {
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .meta-grid {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 4px 24px;
  }

  .donut {
    height: 320px;
    width: 100%;
  }

  /* Match the card-title spacing used elsewhere on the page */
  .card-title {
    font-size: 1.1rem;
    font-weight: 600;
    padding-bottom: 4px;
  }
</style>
