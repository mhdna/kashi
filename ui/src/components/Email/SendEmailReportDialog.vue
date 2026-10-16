<!--
  Emails an invoice report (api/email_report.go): one day, a date range or
  one closed shift, for any mix of registers across branches (none = all).
  Opens on whatever the page is showing (the Day Details page passes its day
  and branch); the preview line says what will go out before Send. Sender,
  reply-to, default recipients and the CSV's columns are on Settings > Email.
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

        <div class="d-flex align-center flex-wrap ga-3 mb-4">
          <v-btn-toggle
            v-model="scope"
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
          <v-spacer />
          <v-btn-toggle
            v-model="format"
            color="primary"
            density="compact"
            divided
            mandatory
            variant="outlined"
          >
            <v-btn icon="mdi-file-delimited-outline" title="CSV" value="csv" />
            <v-btn icon="mdi-microsoft-excel" title="Excel workbook" value="xlsx" />
          </v-btn-toggle>
        </div>

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

        <!-- Same rows in every mode so the dialog keeps its height: a shift
             already names its branch, so Shift takes Registers' place. The
             picks show as one line ("Centro · CT1, +2 more") so many of
             them don't grow it either. -->
        <v-select
          v-if="scope !== 'shift'"
          v-model="registerIds"
          class="mb-4"
          clearable
          density="compact"
          hide-details
          :items="registerOptions"
          label="Registers"
          multiple
          persistent-placeholder
          placeholder="All registers"
          variant="outlined"
        >
          <template #selection="{ item, index }">
            <span v-if="index === 0" class="text-truncate">{{ item.title }}</span>
            <span v-else-if="index === 1" class="text-medium-emphasis text-no-wrap ms-1">
              , +{{ registerIds.length - 1 }} more
            </span>
          </template>
        </v-select>
        <v-select
          v-else
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

        <!-- Type an address and press Enter (or comma) for each person;
             past recipients are suggested. -->
        <v-combobox
          v-model="recipients"
          chips
          closable-chips
          :delimiters="[',', ';', ' ']"
          density="compact"
          :hint="invalidRecipients.length > 0 ? `Not an email address: ${invalidRecipients.join(', ')}` : 'Press Enter after each address'"
          :items="pastRecipients"
          label="Send to"
          multiple
          :no-data-text="'No one emailed yet'"
          persistent-hint
          variant="outlined"
        />

        <v-textarea
          v-model="note"
          class="mt-4"
          counter="2000"
          density="compact"
          label="Note (optional)"
          maxlength="2000"
          no-resize
          persistent-counter
          placeholder="Shown at the top of the email, above the totals"
          rows="2"
          variant="outlined"
        />

        <div class="preview mt-4 text-body-2">
          <template v-if="previewError">
            <div class="text-error text-truncate" :title="previewError">{{ previewError }}</div>
          </template>
          <template v-else-if="preview">
            <div class="font-weight-medium text-truncate" :title="preview.subject">{{ preview.subject }}</div>
            <div class="text-medium-emphasis text-truncate">
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
  const {
    getEmailSettings,
    previewEmailReport,
    sendEmailReport,
    listEmailReportShifts,
    listEmailReportRecipients,
    listEmailReportRegisters,
  } = useEmailReports()

  function ymd (date) {
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  }
  const today = ymd(new Date())

  const scope = ref('day')
  const from = ref(today)
  const to = ref(today)
  const shiftId = ref(null)
  const recipients = ref([])
  const note = ref('')
  const format = ref('csv')
  const pastRecipients = ref([])
  const EMAIL_RE = /^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/
  const invalidRecipients = computed(() => recipients.value.filter(r => !EMAIL_RE.test(String(r).trim())))
  function splitRecipients (text) {
    return String(text || '').split(/[,;\n]/).map(r => r.trim()).filter(Boolean)
  }
  const configured = ref(null)
  const canSeeHistory = computed(() => auth.canViewPage('system', 'emails'))

  const registers = ref([])
  const registerIds = ref([])
  const registerOptions = computed(() => registers.value.map(r => ({
    title: `${r.branch_name} · ${r.code}`,
    value: r.id,
  })))

  // Fresh each time it opens, on whatever the page is showing.
  watch(open, async isOpen => {
    if (!isOpen) return
    scope.value = 'day'
    from.value = props.day || today
    to.value = from.value
    registerIds.value = []
    shiftId.value = null
    note.value = ''
    try {
      const [data, past, list] = await Promise.all([
        getEmailSettings(),
        listEmailReportRecipients().catch(() => ({ recipients: [], last: [] })),
        listEmailReportRegisters(),
      ])
      registers.value = list
      // Opened on one branch's day: start with that branch's registers.
      if (props.branchId) {
        registerIds.value = list.filter(r => r.branch_id === props.branchId).map(r => r.id)
      }
      configured.value = Boolean(data?.configured)
      format.value = data?.settings?.email_attachment_format || 'csv'
      pastRecipients.value = past.recipients
      // Settings' default recipients, else whoever got the last report.
      const defaults = splitRecipients(data?.settings?.email_default_recipients)
      recipients.value = defaults.length > 0 ? defaults : [...past.last]
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
  watch([open, scope, from], async () => {
    if (!open.value || scope.value !== 'shift' || !from.value) return
    loadingShifts.value = true
    try {
      shifts.value = await listEmailReportShifts(from.value, from.value, null)
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
    register_ids: scope.value === 'shift' ? '' : registerIds.value.join(','),
    shift_id: scope.value === 'shift' ? shiftId.value : null,
    format: format.value,
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
        if (requested === JSON.stringify(params.value)) loadingPreview.value = false
      }
    }, 250)
  }, { deep: true })

  // ---- send ----
  const sending = ref(false)
  const canSend = computed(() =>
    configured.value !== false
    && recipients.value.length > 0
    && invalidRecipients.value.length === 0
    && preview.value?.invoice_count > 0,
  )
  async function send () {
    sending.value = true
    try {
      const report = await sendEmailReport(params.value, recipients.value.map(r => String(r).trim()).join(', '), note.value)
      notify.success(`Sent to ${report?.recipients}.`)
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
  /* Always two lines tall, so a longer label or an error never resizes. */
  height: 3em;
}
</style>
