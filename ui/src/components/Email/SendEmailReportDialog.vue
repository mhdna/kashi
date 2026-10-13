<!--
  Emails an invoice report (api/email_report.go): one day, a date range or
  one closed shift, all branches or one. Opens on whatever the page is
  showing (the Day Details page passes its day and branch); the preview line
  says what will go out before Send. Sender, reply-to, default recipients
  and the CSV's columns are on Settings > Email.
-->
<template>
  <v-dialog v-model="open" max-width="520">
    <v-card>
      <v-card-title class="d-flex align-center ga-2 pt-4">
        <v-icon icon="mdi-email-outline" />
        Email invoice report
      </v-card-title>

      <v-card-text class="pt-2">
        <v-alert
          v-if="configured === false"
          class="mb-4"
          density="compact"
          type="warning"
          variant="tonal"
        >
          Email isn't set up yet: add the Resend API key on Settings &gt; Email.
        </v-alert>

        <v-btn-toggle
          v-model="scope"
          class="mb-4"
          color="primary"
          density="compact"
          divided
          mandatory
          variant="outlined"
        >
          <v-btn text="Day" value="day" />
          <v-btn text="Date range" value="range" />
          <v-btn text="Shift" value="shift" />
        </v-btn-toggle>

        <div class="d-flex ga-3 mb-4">
          <v-text-field
            v-model="from"
            density="compact"
            hide-details
            :label="scope === 'range' ? 'From' : 'Day'"
            :max="today"
            type="date"
            variant="outlined"
          />
          <v-text-field
            v-if="scope === 'range'"
            v-model="to"
            density="compact"
            hide-details
            label="To"
            :max="today"
            type="date"
            variant="outlined"
          />
        </div>

        <v-select
          v-model="selectedBranchId"
          class="mb-4"
          density="compact"
          hide-details
          :items="branchOptions"
          label="Branch"
          variant="outlined"
        />

        <v-select
          v-if="scope === 'shift'"
          v-model="shiftId"
          class="mb-4"
          density="compact"
          hide-details
          :items="shiftOptions"
          label="Shift"
          :loading="loadingShifts"
          :no-data-text="loadingShifts ? 'Loading…' : 'No shifts closed that day'"
          variant="outlined"
        />

        <v-text-field
          v-model="recipients"
          density="compact"
          hint="Comma-separated for several people"
          label="Send to"
          persistent-hint
          variant="outlined"
        />

        <div class="preview mt-4 text-body-2">
          <template v-if="previewError">
            <span class="text-error">{{ previewError }}</span>
          </template>
          <template v-else-if="preview">
            <div class="font-weight-medium">{{ preview.subject }}</div>
            <div class="text-medium-emphasis">
              {{ preview.invoice_count.toLocaleString() }} invoice{{ preview.invoice_count === 1 ? '' : 's' }}
              · net {{ money(preview.total_cents) }} · {{ preview.filename }}
            </div>
          </template>
          <span v-else class="text-medium-emphasis">{{ loadingPreview ? 'Checking…' : '' }}</span>
        </div>
      </v-card-text>

      <v-card-actions class="px-4 pb-4">
        <v-btn
          v-if="canSeeHistory"
          size="small"
          text="Sent emails"
          to="/emails"
          variant="text"
        />
        <v-spacer />
        <v-btn text="Cancel" variant="text" @click="open = false" />
        <v-btn
          color="primary"
          :disabled="!canSend"
          :loading="sending"
          prepend-icon="mdi-send-outline"
          text="Send"
          variant="flat"
          @click="send"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import { useBranches } from '@/composables/useBranches'
  import { useEmailReports } from '@/composables/useEmailReports'
  import { useNotify } from '@/composables/useNotify'
  import { useAuthStore } from '@/stores/auth'

  const props = defineProps({
    // YYYY-MM-DD the dialog opens on; today when unset.
    day: { type: String, default: '' },
    branchId: { type: Number, default: null },
  })
  const emit = defineEmits(['sent'])
  const open = defineModel({ type: Boolean, default: false })

  const notify = useNotify()
  const auth = useAuthStore()
  const { branches, fetchBranches } = useBranches()
  const { getEmailSettings, previewEmailReport, sendEmailReport, listEmailReportShifts } = useEmailReports()

  function ymd (date) {
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  }
  const today = ymd(new Date())

  const scope = ref('day')
  const from = ref(today)
  const to = ref(today)
  const selectedBranchId = ref(null)
  const shiftId = ref(null)
  const recipients = ref('')
  const configured = ref(null)
  const canSeeHistory = computed(() => auth.canViewPage('system', 'emails'))

  const branchOptions = computed(() => [
    { title: 'All branches', value: null },
    ...branches.value.map(b => ({ title: b.name, value: b.id })),
  ])

  // Fresh each time it opens, on whatever the page is showing.
  watch(open, async isOpen => {
    if (!isOpen) return
    scope.value = 'day'
    from.value = props.day || today
    to.value = from.value
    selectedBranchId.value = props.branchId
    shiftId.value = null
    fetchBranches().catch(() => {})
    try {
      const data = await getEmailSettings()
      configured.value = Boolean(data?.configured)
      recipients.value = data?.settings?.email_default_recipients ?? ''
    } catch (error) {
      notify.error(error.message)
    }
  })

  // ---- shifts ----
  const shifts = ref([])
  const loadingShifts = ref(false)
  function clock (value) {
    return value ? new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '?'
  }
  const shiftOptions = computed(() => shifts.value.map(s => ({
    title: [s.branch_name, s.register_code, s.closing_person_name, `${clock(s.opened_at)}–${clock(s.closed_at || s.occurred_at)}`]
      .filter(Boolean)
      .join(' · '),
    value: s.id,
  })))
  watch([open, scope, from, selectedBranchId], async () => {
    if (!open.value || scope.value !== 'shift' || !from.value) return
    loadingShifts.value = true
    try {
      shifts.value = await listEmailReportShifts(from.value, from.value, selectedBranchId.value)
      if (!shifts.value.some(s => s.id === shiftId.value)) {
        shiftId.value = shifts.value[0]?.id ?? null
      }
    } catch (error) {
      shifts.value = []
      notify.error(error.message)
    } finally {
      loadingShifts.value = false
    }
  })

  // ---- preview ----
  const params = computed(() => ({
    scope: scope.value,
    from: from.value,
    to: scope.value === 'range' ? to.value : '',
    branch_id: scope.value === 'shift' ? null : selectedBranchId.value,
    shift_id: scope.value === 'shift' ? shiftId.value : null,
  }))
  const preview = ref(null)
  const previewError = ref('')
  const loadingPreview = ref(false)
  let previewTimer = null
  watch([open, params], () => {
    clearTimeout(previewTimer)
    preview.value = null
    previewError.value = ''
    if (!open.value || !from.value || (scope.value === 'shift' && !shiftId.value)) return
    const requested = JSON.stringify(params.value)
    loadingPreview.value = true
    previewTimer = setTimeout(async () => {
      try {
        const data = await previewEmailReport(params.value)
        if (requested === JSON.stringify(params.value)) preview.value = data
      } catch (error) {
        if (requested === JSON.stringify(params.value)) previewError.value = error.message
      } finally {
        loadingPreview.value = false
      }
    }, 250)
  }, { deep: true })

  // ---- send ----
  const sending = ref(false)
  const canSend = computed(() =>
    configured.value !== false
    && recipients.value.trim() !== ''
    && preview.value?.invoice_count > 0,
  )
  async function send () {
    sending.value = true
    try {
      const report = await sendEmailReport(params.value, recipients.value)
      notify.success(`Sent to ${report?.recipients || recipients.value}.`)
      emit('sent', report)
      open.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      sending.value = false
    }
  }

  function money (cents) {
    const n = Number(cents || 0) / 100
    const abs = Math.abs(n).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
    return `${n < 0 ? '−' : ''}$${abs}`
  }
</script>

<style scoped>
.preview {
  min-height: 2.6em;
}
</style>
