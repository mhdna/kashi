<template>
  <div class="mb-6">
    <div class="d-flex align-center ga-2 mb-2">
      <span class="text-subtitle-1 font-weight-medium">Sync</span>
      <v-chip
        v-if="stuckCount > 0"
        color="error"
        size="small"
        variant="tonal"
      >
        {{ stuckCount }} stuck
      </v-chip>
      <v-chip
        v-if="openConflicts > 0"
        color="warning"
        size="small"
        to="/branch-conflicts"
        variant="tonal"
      >
        {{ openConflicts }} open conflict{{ openConflicts === 1 ? '' : 's' }}
      </v-chip>
    </div>
    <p class="text-body-2 text-medium-emphasis mb-3">
      What each till has received from kashi: discounts, prices, products and the rest. A till stuck on one
      kind of data keeps selling on its old copy.
    </p>

    <v-alert v-if="error" class="mb-3" density="compact" type="error" variant="tonal">{{ error }}</v-alert>
    <p v-else-if="!loaded" class="text-body-2 text-medium-emphasis">Loading…</p>
    <p v-else-if="registers.length === 0" class="text-body-2 text-medium-emphasis">No tills here.</p>

    <v-expansion-panels v-model="open" multiple variant="accordion">
      <v-expansion-panel v-for="r in registers" :key="r.register_id" :value="r.register_id">
        <v-expansion-panel-title>
          <div class="d-flex flex-wrap align-center ga-2">
            <span class="font-weight-medium">{{ r.branch_name }} · {{ r.register_code }}</span>
            <span class="text-caption text-medium-emphasis">
              {{ r.app_version ? `v${r.app_version} · ` : '' }}seen {{ r.last_seen_at ? ago(r.last_seen_at) : 'never' }}
            </span>
            <v-chip :color="STATES[registerState(r)].color" size="x-small" variant="tonal">
              {{ registerSummary(r) }}
            </v-chip>
            <v-chip v-if="r.outbox_failed > 0" color="error" size="x-small" variant="tonal">
              {{ r.outbox_failed }} failed to send up
            </v-chip>
            <v-chip v-if="r.outbox_pending > 0" color="warning" size="x-small" variant="tonal">
              {{ r.outbox_pending }} waiting to send up
            </v-chip>
          </div>
        </v-expansion-panel-title>
        <v-expansion-panel-text>
          <p v-if="r.entities.length === 0" class="text-body-2 text-medium-emphasis">
            This till hasn't pulled since kashi started recording it.
          </p>
          <v-table v-else density="compact">
            <thead>
              <tr>
                <th>Data</th>
                <th>Status</th>
                <th>Last pull</th>
                <th>Got</th>
                <th>Latest change in kashi</th>
                <th>Detail</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in sortedEntities(r)" :key="e.entity">
                <td>{{ LABELS[e.entity] ?? e.entity }}</td>
                <td>
                  <v-chip :color="STATES[e.state].color" size="x-small" variant="tonal">
                    {{ STATES[e.state].title }}
                  </v-chip>
                </td>
                <td>{{ e.last_pull_at ? ago(e.last_pull_at) : '—' }}</td>
                <td>{{ e.items_sent > 0 ? e.items_sent : '—' }}</td>
                <td>{{ e.latest_change_at ? ago(e.latest_change_at) : '—' }}</td>
                <td class="sync-detail">{{ detail(e) }}</td>
              </tr>
            </tbody>
          </v-table>
        </v-expansion-panel-text>
      </v-expansion-panel>
    </v-expansion-panels>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { API_BASE } from '@/config'

  // What each till has of each kind of data (api/branch_sync_status.go).
  const props = defineProps({
    // null is every branch.
    branchId: { type: Number, default: null },
  })

  const STATES = {
    synced: { title: 'Synced', color: 'success', rank: 0 },
    sent: { title: 'Sent, not confirmed', color: undefined, rank: 1 },
    behind: { title: 'Behind', color: 'warning', rank: 2 },
    retrying: { title: 'Retrying', color: 'warning', rank: 3 },
    stuck: { title: 'Stuck', color: 'error', rank: 4 },
  }

  const LABELS = {
    discount_lists: 'Discount lists',
    price_lists: 'Price lists',
    invoice_discounts: 'Invoice discounts',
    products: 'Products',
    clients: 'Clients',
    coupons: 'Coupons',
    coupon_types: 'Coupon types',
    currencies: 'Currencies',
    cashbox_accounts: 'Payment methods',
    branch_targets: 'Targets',
    salespersons: 'Salespersons',
    branch_users: 'Users',
    expense_categories: 'Expense categories',
    loan_categories: 'Loan categories',
    branch_stock: 'Stock',
    branches: 'Branches',
    quick_access: 'Quick access',
    calendar_events: 'Calendar events',
    transfer_requests: 'Transfer requests',
    incoming_transfer_requests: 'Incoming transfers',
    branch_invoice_history: 'Invoice history',
    branch_expense_history: 'Expense history',
    branch_loan_history: 'Loan history',
    branch_shift_history: 'Shift history',
  }

  const registers = ref([])
  const conflicts = ref([])
  const loaded = ref(false)
  const error = ref('')
  const open = ref([])

  function worstState (r) {
    return r.entities.reduce((worst, e) => (STATES[e.state].rank > STATES[worst].rank ? e.state : worst), 'synced')
  }
  const registerState = worstState

  function registerSummary (r) {
    if (r.entities.length === 0) return 'No sync data yet'
    const stuck = r.entities.filter(e => e.state === 'stuck').length
    if (stuck > 0) return `${stuck} stuck`
    return STATES[worstState(r)].title
  }

  // Worst first: what needs attention is at the top.
  function sortedEntities (r) {
    return r.entities.toSorted((a, b) => STATES[b.state].rank - STATES[a.state].rank
      || (LABELS[a.entity] ?? a.entity).localeCompare(LABELS[b.entity] ?? b.entity))
  }

  const stuckCount = computed(() => registers.value.reduce(
    (n, r) => n + r.entities.filter(e => e.state === 'stuck').length, 0))
  const openConflicts = computed(() => conflicts.value
    .filter(c => props.branchId == null || c.branch_id === props.branchId)
    .reduce((n, c) => n + c.open, 0))

  function ago (value) {
    const seconds = Math.max(0, Math.round((Date.now() - new Date(value)) / 1000))
    if (seconds < 60) return 'just now'
    if (seconds < 3600) return `${Math.round(seconds / 60)} min ago`
    if (seconds < 86_400) return `${Math.round(seconds / 3600)} h ago`
    return `${Math.round(seconds / 86_400)} d ago`
  }

  function detail (e) {
    if (e.error) return e.error
    if (e.state === 'stuck' || e.state === 'retrying') {
      const since = e.stuck_since ? `since ${ago(e.stuck_since)}` : ''
      return `Asked for the same data ${e.retries + 1} times ${since}`.trim()
    }
    if (e.state === 'sent') return 'Confirmed on its next pull'
    if (e.state === 'behind') return 'Changed since its last pull'
    return e.rows_applied > 0 ? `${e.rows_applied.toLocaleString()} rows applied` : ''
  }

  async function getJSON (url) {
    const res = await authFetch(url)
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || `Request failed with status ${res.status}`)
    return data
  }

  // Only the newest answer counts when the branch changes mid-request.
  let latest = 0
  async function load () {
    const mine = ++latest
    const query = props.branchId == null ? '' : `?branch_id=${props.branchId}`
    try {
      const [status, summary] = await Promise.all([
        getJSON(`${API_BASE}/branch_sync_status${query}`),
        getJSON(`${API_BASE}/branch_sync_conflicts/summary`),
      ])
      if (mine !== latest) return
      registers.value = status.registers ?? []
      conflicts.value = summary.branches ?? []
      error.value = ''
      // A till with a problem opens by itself; one the user opened stays open.
      const troubled = registers.value.filter(r => STATES[worstState(r)].rank >= 3).map(r => r.register_id)
      open.value = [...new Set([...open.value, ...troubled])]
    } catch (error_) {
      if (mine === latest) error.value = error_.message
    } finally {
      if (mine === latest) loaded.value = true
    }
  }

  watch(() => props.branchId, () => {
    open.value = []
    load()
  })

  let poll = null
  onMounted(() => {
    load()
    poll = setInterval(load, 15_000)
  })
  onBeforeUnmount(() => clearInterval(poll))

  defineExpose({ reload: load })
</script>

<style scoped>
.sync-detail {
  max-width: 360px;
  overflow-wrap: anywhere;
  font-size: 0.8rem;
}
</style>
