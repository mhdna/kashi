<template>
  <div class="pa-4 pt-3">
    <div class="d-flex align-center flex-wrap ga-3 mb-3">
      <p class="text-body-2 text-medium-emphasis wh-intro mb-0">
        Warning signs before WhatsApp bans a number, from the last 24 hours: disconnects, messages
        WhatsApp refused, failed sends and sudden jumps in volume. A number that reaches
        <strong>70</strong> pauses itself and keeps its messages queued until you resume it.
      </p>
      <v-spacer />
      <v-btn
        :loading="loading"
        prepend-icon="mdi-refresh"
        text="Refresh"
        variant="tonal"
        @click="load"
      />
    </div>

    <v-alert
      v-if="bridgeError"
      class="mb-3"
      density="compact"
      type="warning"
      variant="tonal"
    >
      {{ bridgeError }}
    </v-alert>

    <div v-if="!loading && numbers.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">
      No numbers yet. Add one in <router-link to="/settings?tab=whatsapp">Settings</router-link>.
    </div>

    <v-row dense>
      <v-col
        v-for="n in numbers"
        :key="n.id"
        cols="12"
        lg="4"
        md="6"
      >
        <v-card class="wh-card" :class="`wh-card--${level(n)}`" variant="flat">
          <div class="d-flex align-start ga-3">
            <div class="flex-grow-1 min-w-0">
              <div class="text-subtitle-1 font-weight-bold text-truncate">
                {{ n.label || formatPhone(n.phone) || n.instance }}
                <v-chip v-if="n.is_default" class="ms-1" size="x-small" variant="tonal">Default</v-chip>
              </div>
              <div class="text-caption text-medium-emphasis">
                {{ formatPhone(n.phone) || '—' }} · {{ stateText(n.state) }}
                <template v-if="n.queued"> · {{ n.queued }} queued</template>
              </div>
            </div>
            <div class="text-end">
              <div class="wh-risk" :class="`text-${levelColor(n)}`">{{ n.health ? n.health.risk : '—' }}</div>
              <div class="text-caption text-medium-emphasis">risk</div>
            </div>
          </div>

          <v-progress-linear
            v-if="n.health"
            class="my-3"
            :color="levelColor(n)"
            height="6"
            :model-value="n.health.risk"
            rounded
          />

          <v-alert
            v-if="n.health?.paused_at"
            class="mb-3"
            density="compact"
            icon="mdi-pause"
            type="error"
            variant="tonal"
          >
            <div class="font-weight-bold">Paused {{ when(n.health.paused_at) }}</div>
            <div class="text-body-2">{{ n.health.paused_reason }}</div>
            <div class="text-caption mt-1">
              Open WhatsApp on that phone and check for a warning before resuming. Its branches'
              messages wait in the queue meanwhile.
            </div>
            <v-btn
              class="mt-2"
              color="error"
              :loading="resuming === n.id"
              prepend-icon="mdi-play"
              size="small"
              text="Resume sending"
              variant="flat"
              @click="resume(n)"
            />
          </v-alert>

          <template v-if="n.health">
            <div v-if="n.health.reasons.length > 0" class="mb-2">
              <div v-for="r in n.health.reasons" :key="r" class="d-flex align-start ga-2 text-body-2 py-1">
                <v-icon class="mt-1" :color="levelColor(n)" icon="mdi-alert" size="16" />
                <span>{{ r }}</span>
              </div>
            </div>
            <div v-else class="d-flex align-center ga-2 text-body-2 text-success mb-2">
              <v-icon icon="mdi-shield-check" size="18" />
              <span>No warning signs</span>
            </div>

            <div class="wh-stats text-body-2">
              <span>Sent today</span>
              <strong>{{ n.health.sent_24h }} <span class="text-medium-emphasis font-weight-regular">(usual {{ n.health.sent_daily_avg }})</span></strong>
              <span>Disconnects</span><strong>{{ n.health.disconnects_24h }}</strong>
              <span>Refused by WhatsApp</span><strong>{{ n.health.rejected_24h }}</strong>
              <span>Failed sends</span><strong>{{ n.health.failed_24h }}</strong>
              <span title="Messages of the last week not delivered after 2 days: blocked, or the phone is off">Stuck on one tick</span><strong>{{ n.health.one_tick ?? 0 }}</strong>
            </div>
            <div v-if="n.health.last_problem_at" class="text-caption text-medium-emphasis mt-2">
              Last problem: {{ n.health.last_problem }}, {{ when(n.health.last_problem_at) }}
            </div>
          </template>
          <div v-else class="text-body-2 text-medium-emphasis">
            The bridge doesn't report health for this number (it needs bridge 2.4.0).
          </div>

          <div class="text-caption text-medium-emphasis mt-3">
            Sends for: {{ n.branches.length > 0 ? n.branches.join(', ') : 'no branch' }}
          </div>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup>
  import { onMounted, onUnmounted, ref, watch } from 'vue'
  import { useNotify } from '@/composables/useNotify'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { formatPhone } from '@/utils/whatsapp'

  // Each number's warning signs (api/whatsapp_health.go, from the bridge),
  // refreshed every minute while open.

  const props = defineProps({
    refreshKey: { type: Number, default: 0 },
  })
  const emit = defineEmits(['alert'])

  const notify = useNotify()
  const { getHealth, resumeNumber } = useWhatsApp()

  const numbers = ref([])
  const bridgeError = ref('')
  const loading = ref(false)
  const resuming = ref(0)
  let timer

  const level = n => (n.health?.paused_at ? 'high' : n.health?.level ?? 'unknown')
  const levelColor = n => ({ high: 'error', warning: 'warning', ok: 'success' })[level(n)] ?? 'grey'
  const stateText = state => ({ open: 'linked', close: 'not linked', connecting: 'connecting', missing: 'missing on the bridge' })[state] ?? state

  function when (at) {
    const minutes = Math.round((Date.now() - new Date(at).getTime()) / 60_000)
    if (minutes < 1) return 'just now'
    if (minutes < 60) return `${minutes} min ago`
    if (minutes < 24 * 60) return `${Math.round(minutes / 60)} h ago`
    return new Date(at).toLocaleString()
  }

  async function load () {
    loading.value = true
    try {
      const data = await getHealth()
      numbers.value = data.numbers ?? []
      bridgeError.value = data.bridge_error ?? ''
      emit('alert', numbers.value.some(n => level(n) !== 'ok' && level(n) !== 'unknown'))
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  async function resume (n) {
    resuming.value = n.id
    try {
      await resumeNumber(n.id)
      notify.success('Sending again')
      await load()
    } catch (error) {
      notify.error(error.message)
    } finally {
      resuming.value = 0
    }
  }

  watch(() => props.refreshKey, load)
  onMounted(() => {
    load()
    timer = setInterval(load, 60_000)
  })
  onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.wh-intro {
  max-width: 760px;
}
.wh-card {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-left-width: 4px;
  padding: 16px;
  height: 100%;
}
.wh-card--ok {
  border-left-color: rgb(var(--v-theme-success));
}
.wh-card--warning {
  border-left-color: rgb(var(--v-theme-warning));
}
.wh-card--high {
  border-left-color: rgb(var(--v-theme-error));
}
.wh-risk {
  font-size: 2rem;
  font-weight: 700;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}
.wh-stats {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 4px 12px;
}
.min-w-0 {
  min-width: 0;
}
</style>
