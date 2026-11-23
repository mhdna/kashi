<template>
  <div v-if="registers.length > 0" class="px-4 py-2">
    <div class="d-flex flex-wrap align-center ga-2">
      <span class="text-body-2 font-weight-medium">{{ summary }}</span>
      <span v-if="changedAt" class="text-caption text-medium-emphasis">{{ t('last changed {when}', { when: ago(changedAt) }) }}</span>
    </div>
    <v-progress-linear
      v-if="pendingCount > 0"
      class="my-1"
      :color="stuckCount > 0 ? 'error' : 'primary'"
      :model-value="progress"
    />
    <div class="d-flex flex-wrap ga-2 mt-1">
      <v-tooltip v-for="r in registers" :key="r.register_id" location="bottom" :text="detail(r)">
        <template #activator="{ props: tip }">
          <v-chip
            v-bind="tip"
            :color="STATES[r.state].color"
            size="small"
            variant="tonal"
          >
            <v-icon v-if="r.verified" class="me-1" icon="mdi-check-circle" size="16" />
            {{ r.branch_name }}, {{ r.register_code }}: {{ chipTitle(r) }}
          </v-chip>
        </template>
      </v-tooltip>
    </div>
    <p v-if="firstError" class="text-caption text-error mt-1 mb-0">{{ firstError }}</p>
  </div>
  <p v-else-if="loaded" class="px-4 py-2 text-body-2 text-medium-emphasis mb-0">
    {{ t('No branch uses this list yet, so no till gets it.') }}
  </p>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { API_BASE } from '@/config'
  import { t } from '@/i18n'

  // Whether this list's latest change has reached each till (api/list_delivery.go).
  const props = defineProps({
    kind: { type: String, required: true },
    listId: { type: Number, required: true },
  })

  const STATES = {
    reached: { title: t('Reached'), color: 'success' },
    sending: { title: t('Sending…'), color: 'primary' },
    waiting: { title: t('Waiting for the till'), color: 'warning' },
    offline: { title: t('Offline'), color: undefined },
    stuck: { title: t('Stuck'), color: 'error' },
    different: { title: t('Different'), color: 'error' },
  }

  const registers = ref([])
  const changedAt = ref(null)
  const expectedRows = ref(0)
  const loaded = ref(false)

  // An offline till can't be waited on, so it doesn't hold the progress back.
  const online = computed(() => registers.value.filter(r => r.state !== 'offline'))
  const reachedCount = computed(() => registers.value.filter(r => r.state === 'reached').length)
  // Stuck or holding something else: both are a till selling on the wrong list.
  const stuckCount = computed(() => registers.value.filter(r => r.state === 'stuck' || r.state === 'different').length)
  const offlineCount = computed(() => registers.value.length - online.value.length)
  const pendingCount = computed(() => online.value.filter(r => r.state !== 'reached').length)
  const progress = computed(() => online.value.length === 0 ? 0 : 100 * (online.value.length - pendingCount.value) / online.value.length)
  const firstError = computed(() => registers.value.find(r => r.error)?.error ?? '')

  const summary = computed(() => {
    const reached = reachedCount.value
    const total = registers.value.length
    const parts = [total === 1
      ? t('Reached {reached} of {total} till', { reached, total })
      : t('Reached {reached} of {total} tills', { reached, total })]
    if (stuckCount.value > 0) parts.push(t('{n} with a problem', { n: stuckCount.value }))
    if (offlineCount.value > 0) parts.push(t('{n} offline', { n: offlineCount.value }))
    return parts.join(', ')
  })

  function ago (value) {
    const seconds = Math.max(0, Math.round((Date.now() - new Date(value)) / 1000))
    if (seconds < 60) return t('just now')
    if (seconds < 3600) return t('{n} min ago', { n: Math.round(seconds / 60) })
    if (seconds < 86_400) return t('{n} h ago', { n: Math.round(seconds / 3600) })
    return t('{n} d ago', { n: Math.round(seconds / 86_400) })
  }

  function chipTitle (r) {
    // An older till can't be checked: it should have it, going by its sync.
    if (r.state === 'reached' && !r.verified) return t('Reached (not checked)')
    return STATES[r.state].title
  }

  function detail (r) {
    if (r.state === 'reached' && r.verified) return r.reached_at
      ? t('Checked: the till holds exactly this list, as of {when}', { when: ago(r.reached_at) })
      : t('Checked: the till holds exactly this list')
    if (r.state === 'reached') return t('Going by its sync it should have this. It needs a newer version of the app to check what it really holds.')
    if (r.state === 'different') return t("The till says it's up to date but holds something else ({rows} rows, kashi has {expected}). To send it again, edit this list and save it.", { rows: r.till_rows ?? '?', expected: expectedRows.value })
    if (r.state === 'stuck') return r.error || t('It keeps asking for the same data, so applying it is failing')
    if (r.state === 'offline') return t('Last seen {when}. It gets this when it comes back.', { when: r.last_seen_at ? ago(r.last_seen_at) : t('never') })
    if (r.state === 'sending') return t('Pulled it {when}; it confirms when applied, or on its next pull', { when: r.last_pull_at ? ago(r.last_pull_at) : '' })
    return t("Online, and it hasn't asked for this version yet")
  }

  // Only the newest answer counts when another list is picked mid-request.
  let latest = 0
  let timer = null
  async function load () {
    const mine = ++latest
    clearTimeout(timer)
    const path = props.kind === 'price' ? 'price_lists' : 'discount_lists'
    try {
      const res = await authFetch(`${API_BASE}/${path}/${props.listId}/delivery`)
      const data = await res.json().catch(() => ({}))
      if (mine !== latest) return
      if (res.ok) {
        registers.value = data.registers ?? []
        changedAt.value = data.changed_at
        expectedRows.value = data.expected_rows ?? 0
      }
    } catch { /* informational: the next round tries again */ } finally {
      if (mine === latest) {
        loaded.value = true
        // Quick while a till is on its way to it; a slow check otherwise.
        timer = setTimeout(load, pendingCount.value > 0 ? 3000 : 30_000)
      }
    }
  }

  watch(() => props.listId, () => {
    registers.value = []
    loaded.value = false
    load()
  })
  onMounted(load)
  onBeforeUnmount(() => {
    latest++
    clearTimeout(timer)
  })

  defineExpose({ reload: load })
</script>
