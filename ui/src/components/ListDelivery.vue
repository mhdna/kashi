<template>
  <div v-if="registers.length > 0" class="px-4 py-2">
    <div class="d-flex flex-wrap align-center ga-2">
      <span class="text-body-2 font-weight-medium">{{ summary }}</span>
      <span v-if="changedAt" class="text-caption text-medium-emphasis">last changed {{ ago(changedAt) }}</span>
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
            {{ r.branch_name }} · {{ r.register_code }}: {{ STATES[r.state].title }}
          </v-chip>
        </template>
      </v-tooltip>
    </div>
    <p v-if="firstError" class="text-caption text-error mt-1 mb-0">{{ firstError }}</p>
  </div>
  <p v-else-if="loaded" class="px-4 py-2 text-body-2 text-medium-emphasis mb-0">
    No branch uses this list yet, so no till gets it.
  </p>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { API_BASE } from '@/config'

  // Whether this list's latest change has reached each till (api/list_delivery.go).
  const props = defineProps({
    kind: { type: String, required: true },
    listId: { type: Number, required: true },
  })

  const STATES = {
    reached: { title: 'Reached', color: 'success' },
    sending: { title: 'Sending…', color: 'primary' },
    waiting: { title: 'Waiting for the till', color: 'warning' },
    offline: { title: 'Offline', color: undefined },
    stuck: { title: 'Stuck', color: 'error' },
  }

  const registers = ref([])
  const changedAt = ref(null)
  const loaded = ref(false)

  // An offline till can't be waited on, so it doesn't hold the progress back.
  const online = computed(() => registers.value.filter(r => r.state !== 'offline'))
  const reachedCount = computed(() => registers.value.filter(r => r.state === 'reached').length)
  const stuckCount = computed(() => registers.value.filter(r => r.state === 'stuck').length)
  const offlineCount = computed(() => registers.value.length - online.value.length)
  const pendingCount = computed(() => online.value.filter(r => r.state !== 'reached').length)
  const progress = computed(() => online.value.length === 0 ? 0 : 100 * (online.value.length - pendingCount.value) / online.value.length)
  const firstError = computed(() => registers.value.find(r => r.error)?.error ?? '')

  const summary = computed(() => {
    const parts = [`Reached ${reachedCount.value} of ${registers.value.length} till${registers.value.length === 1 ? '' : 's'}`]
    if (stuckCount.value > 0) parts.push(`${stuckCount.value} stuck`)
    if (offlineCount.value > 0) parts.push(`${offlineCount.value} offline`)
    return parts.join(' · ')
  })

  function ago (value) {
    const seconds = Math.max(0, Math.round((Date.now() - new Date(value)) / 1000))
    if (seconds < 60) return 'just now'
    if (seconds < 3600) return `${Math.round(seconds / 60)} min ago`
    if (seconds < 86_400) return `${Math.round(seconds / 3600)} h ago`
    return `${Math.round(seconds / 86_400)} d ago`
  }

  function detail (r) {
    if (r.state === 'reached') return `Has this version${r.reached_at ? `, since ${ago(r.reached_at)}` : ''}`
    if (r.state === 'stuck') return r.error || 'It keeps asking for the same data, so applying it is failing'
    if (r.state === 'offline') return `Last seen ${r.last_seen_at ? ago(r.last_seen_at) : 'never'}. It gets this when it comes back.`
    if (r.state === 'sending') return `Pulled it ${r.last_pull_at ? ago(r.last_pull_at) : ''}; it confirms when applied, or on its next pull`
    return 'Online, and it hasn\'t asked for this version yet'
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
