<template>
  <div class="page-root">
    <v-card class="bk-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-gift-outline" />
        <span>Referrals</span>
        <v-spacer />
        <v-btn prepend-icon="mdi-refresh" text="Reload" variant="text" @click="reload" />
      </v-card-title>
      <v-divider />

      <v-card-text>
        <p class="text-body-2 text-medium-emphasis mb-4" data-test="summary">{{ summary }}</p>

        <v-form class="referral-rules mb-6" @submit.prevent="save">
          <div class="text-subtitle-2 mb-1">Branches where referrals are on</div>
          <div class="referral-branches mb-4" data-test="branches">
            <v-switch
              v-for="b in branches"
              :key="b.id"
              v-model="form.branchIds"
              color="primary"
              :data-test="`branch-${b.id}`"
              density="compact"
              hide-details
              inset
              :label="b.name"
              :value="b.id"
            />
            <div v-if="branches.length === 0" class="text-body-2 text-medium-emphasis">No branches yet.</div>
          </div>
          <div class="referral-grid">
            <v-text-field
              v-model="form.rewardPercent"
              data-test="percent"
              density="comfortable"
              hint="Off the owner's next visit, once"
              label="Reward"
              persistent-hint
              suffix="%"
              variant="outlined"
            />
            <v-text-field
              v-model="form.maxRewardDollars"
              density="comfortable"
              hint="Empty for no cap"
              label="Most the reward takes off"
              persistent-hint
              prefix="$"
              variant="outlined"
            />
            <v-text-field
              v-model="form.minFriendSpendDollars"
              density="comfortable"
              hint="The friend's first purchase, empty for any"
              label="Minimum purchase"
              persistent-hint
              prefix="$"
              variant="outlined"
            />
            <v-text-field
              v-model="form.unlockAfterDays"
              density="comfortable"
              hint="0 for right away"
              label="Days before it can be used"
              persistent-hint
              variant="outlined"
            />
            <v-text-field
              v-model="form.expiresAfterDays"
              density="comfortable"
              hint="0 for no end"
              label="Days it lasts"
              persistent-hint
              variant="outlined"
            />
            <v-text-field
              v-model="form.maxRewardsPerMonth"
              density="comfortable"
              hint="For one owner, 0 for no limit"
              label="Rewards a month"
              persistent-hint
              variant="outlined"
            />
          </div>
          <v-checkbox
            v-model="form.cancelOnReturn"
            color="primary"
            hide-details
            label="Take the reward back if the friend's purchase is cancelled or returned (before it is used)"
          />
          <div class="d-flex align-center ga-3 mt-2">
            <v-btn
              color="primary"
              data-test="save"
              :loading="saving"
              type="submit"
              variant="flat"
            >Save</v-btn>
            <span v-if="formError" class="text-error text-body-2" data-test="form-error">{{ formError }}</span>
          </div>
        </v-form>

        <div class="d-flex flex-wrap align-center ga-3 mb-2">
          <div class="text-subtitle-1 font-weight-medium">Referrals</div>
          <v-spacer />
          <v-btn-toggle
            v-model="status"
            color="primary"
            density="compact"
            mandatory
            variant="outlined"
            @update:model-value="loadList"
          >
            <v-btn value="">All</v-btn>
            <v-btn value="available">Ready to use</v-btn>
            <v-btn value="used">Used</v-btn>
            <v-btn value="cancelled">Taken back</v-btn>
            <v-btn value="refused">No reward</v-btn>
          </v-btn-toggle>
        </div>
        <v-table data-test="list" density="comfortable">
          <thead>
            <tr>
              <th>Date</th>
              <th>Owner</th>
              <th>Code</th>
              <th>Friend</th>
              <th>Purchase</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="rows.length === 0">
              <td class="text-medium-emphasis" colspan="6">{{ loading ? 'Loading' : 'No referrals yet' }}</td>
            </tr>
            <tr v-for="row in rows" :key="row.id">
              <td>{{ formatDate(row.when) }}</td>
              <td>
                <div>{{ row.owner }}</div>
                <div class="text-caption text-medium-emphasis">{{ row.ownerPhone }}</div>
              </td>
              <td class="font-weight-medium">{{ row.code }}</td>
              <td>
                <div>{{ row.friend }}</div>
                <div class="text-caption text-medium-emphasis">{{ row.friendPhone }}</div>
              </td>
              <td>
                <div>{{ row.sale }}, ${{ formatMoney(row.saleTotal) }}</div>
                <div class="text-caption text-medium-emphasis">{{ row.branch }}</div>
              </td>
              <td>
                <div :class="statusColor(row.status) ? `text-${statusColor(row.status)}` : ''">{{ row.statusLabel }}</div>
                <div class="text-caption text-medium-emphasis">{{ row.detail }}</div>
              </td>
            </tr>
          </tbody>
        </v-table>
      </v-card-text>
    </v-card>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useBranches } from '@/composables/useBranches'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'
  import { formatMoney } from '@/utils/money'
  import { formToRequest, programSummary, programToForm, referralRow, statusColor } from '@/utils/referrals'

  const notify = useNotify()
  const { branches, fetchBranches } = useBranches()
  function branchName (id) {
    return branches.value.find(b => b.id === id)?.name ?? `Branch #${id}`
  }
  const program = ref(null)
  const form = ref(programToForm())
  const formError = ref('')
  const saving = ref(false)
  const status = ref('')
  const rows = ref([])
  const loading = ref(false)

  const summary = computed(() => programSummary(program.value, branchName))

  async function requestJSON (url, options) {
    const res = await authFetch(url, options)
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {throw new Error(data.error || `Request failed with status ${res.status}`)}
    return data
  }

  async function loadProgram () {
    const data = await requestJSON(`${API_BASE}/referral_program`)
    program.value = data.referral_program
    form.value = programToForm(data.referral_program)
  }

  async function loadList () {
    loading.value = true
    try {
      const query = status.value ? `?status=${status.value}` : ''
      const data = await requestJSON(`${API_BASE}/referrals${query}`)
      rows.value = (data.referrals ?? []).map(r => referralRow(r))
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  async function save () {
    formError.value = ''
    let body
    try {
      body = formToRequest(form.value)
    } catch (error) {
      formError.value = error.message
      return
    }
    saving.value = true
    try {
      const data = await requestJSON(`${API_BASE}/referral_program`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      program.value = data.referral_program
      form.value = programToForm(data.referral_program)
      notify.success('Referral rules saved')
    } catch (error) {
      formError.value = error.message
    } finally {
      saving.value = false
    }
  }

  function formatDate (value) {
    if (!value) {return ''}
    return new Date(value).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
  }

  async function reload () {
    fetchBranches().catch(error => notify.error(error.message))
    try {
      await loadProgram()
    } catch (error) {
      formError.value = error.message
    }
    await loadList()
  }

  onMounted(reload)
</script>

<style scoped>
.referral-branches {
  display: flex;
  flex-wrap: wrap;
  column-gap: 24px;
}
.referral-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
  margin: 16px 0 8px;
}
</style>
