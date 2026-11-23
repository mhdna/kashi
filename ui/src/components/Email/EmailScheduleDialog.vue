<!--
  Creates or edits a scheduled invoice report (api/email_schedule.go): what
  to send (a period relative to each run, which registers, the format), to
  whom, and when -- every day, on some weekdays, or on one day of the month,
  at a time in the shops' zone (Beirut).
-->
<template>
  <v-dialog v-model="open" max-width="560" scrollable>
    <v-card>
      <v-card-title class="d-flex align-center ga-2 pt-4">
        <v-icon icon="mdi-calendar-clock-outline" />
        {{ schedule ? t('Edit scheduled report') : t('New scheduled report') }}
      </v-card-title>

      <v-card-text class="pt-2">
        <v-text-field
          v-model="form.name"
          class="mb-4"
          density="compact"
          hide-details
          :label="t('Name')"
          persistent-placeholder
          :placeholder="t('e.g. Centro end of day')"
          variant="outlined"
        />

        <div class="text-subtitle-2 mb-2">{{ t('What') }}</div>
        <div class="d-flex ga-3 mb-4">
          <v-select
            v-model="form.period"
            density="compact"
            hide-details
            :items="PERIODS"
            :label="t('Invoices from')"
            variant="outlined"
          />
          <v-btn-toggle
            v-model="form.format"
            color="primary"
            density="compact"
            divided
            variant="outlined"
          >
            <v-btn icon="mdi-file-delimited-outline" title="CSV" value="csv" />
            <v-btn icon="mdi-microsoft-excel" :title="t('Excel workbook')" value="xlsx" />
          </v-btn-toggle>
        </div>
        <v-select
          v-model="form.registerIds"
          class="mb-4"
          clearable
          density="compact"
          hide-details
          :items="registerOptions"
          :label="t('Registers')"
          multiple
          persistent-placeholder
          :placeholder="t('All registers')"
          variant="outlined"
        >
          <template #selection="{ item, index }">
            <span v-if="index === 0" class="text-truncate">{{ item.title }}</span>
            <span v-else-if="index === 1" class="text-medium-emphasis text-no-wrap ms-1">
              {{ t(', +{n} more', { n: form.registerIds.length - 1 }) }}
            </span>
          </template>
        </v-select>

        <div class="text-subtitle-2 mb-2">{{ t('When') }}</div>
        <div class="d-flex flex-wrap align-center ga-3 mb-3">
          <v-btn-toggle
            v-model="form.frequency"
            color="primary"
            density="compact"
            divided
            mandatory
            variant="outlined"
          >
            <v-btn :text="t('Daily')" value="daily" />
            <v-btn :text="t('Weekly')" value="weekly" />
            <v-btn :text="t('Monthly')" value="monthly" />
          </v-btn-toggle>
          <v-text-field
            v-model="form.time"
            density="compact"
            hide-details
            :label="t('At')"
            style="max-width: 130px"
            type="time"
            variant="outlined"
          />
        </div>
        <v-chip-group
          v-if="form.frequency === 'weekly'"
          v-model="form.weekdays"
          class="mb-2"
          color="primary"
          column
          multiple
        >
          <v-chip
            v-for="(day, index) in WEEKDAYS"
            :key="day"
            filter
            size="small"
            :text="t(day)"
            :value="index"
            variant="outlined"
          />
        </v-chip-group>
        <v-select
          v-if="form.frequency === 'monthly'"
          v-model="form.monthDay"
          class="mb-4"
          density="compact"
          :hint="t('A shorter month sends on its last day')"
          :items="MONTH_DAYS"
          :label="t('On day')"
          persistent-hint
          style="max-width: 200px"
          variant="outlined"
        />
        <div class="text-caption text-medium-emphasis mb-4">{{ whenSummary }} {{ t('(Beirut time)') }}</div>

        <div class="text-subtitle-2 mb-2">{{ t('Who') }}</div>
        <v-combobox
          v-model="form.recipients"
          chips
          class="mb-4"
          closable-chips
          :delimiters="[',', ';', ' ']"
          density="compact"
          :hint="invalidRecipients.length > 0 ? t('Not an email address: {list}', { list: invalidRecipients.join(', ') }) : t('Press Enter after each address')"
          :items="pastRecipients"
          :label="t('Send to')"
          multiple
          persistent-hint
          variant="outlined"
        />
        <v-combobox
          v-model="form.bcc"
          chips
          class="mb-4"
          closable-chips
          :delimiters="[',', ';', ' ']"
          density="compact"
          :hint="invalidBcc.length > 0 ? t('Not an email address: {list}', { list: invalidBcc.join(', ') }) : t('Hidden from everyone else')"
          :items="pastRecipients"
          :label="t('Bcc')"
          multiple
          persistent-hint
          variant="outlined"
        />
        <v-textarea
          v-model="form.note"
          counter="2000"
          density="compact"
          :label="t('Note (optional)')"
          maxlength="2000"
          no-resize
          rows="2"
          variant="outlined"
        />
        <v-switch
          v-model="form.skipEmpty"
          color="primary"
          density="compact"
          hide-details
          :label="t('Don\'t send when there are no invoices')"
        />
        <v-switch
          v-model="form.enabled"
          color="primary"
          density="compact"
          hide-details
          :label="t('On')"
        />
      </v-card-text>

      <v-card-actions class="px-4 pb-4">
        <v-spacer />
        <v-btn :text="t('Cancel')" variant="text" @click="open = false" />
        <v-btn
          color="primary"
          :disabled="!canSave"
          :loading="saving"
          :text="t('Save')"
          variant="flat"
          @click="save"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
  import { computed, reactive, ref, watch } from 'vue'
  import { useEmailReports } from '@/composables/useEmailReports'
  import { useNotify } from '@/composables/useNotify'
  import { t } from '@/i18n'

  const props = defineProps({
    // The schedule being edited; null creates one.
    schedule: { type: Object, default: null },
  })
  const emit = defineEmits(['saved'])
  const open = defineModel({ type: Boolean, default: false })

  const PERIODS = computed(() => [
    { title: t('Today'), value: 'today' },
    { title: t('Yesterday'), value: 'yesterday' },
    { title: t('Last 7 days'), value: 'last_7_days' },
    { title: t('This week (from Monday)'), value: 'this_week' },
    { title: t('This month'), value: 'this_month' },
    { title: t('Last month'), value: 'last_month' },
  ])
  const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
  const MONTH_DAYS = Array.from({ length: 31 }, (_, i) => i + 1)
  const EMAIL_RE = /^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/

  const notify = useNotify()
  const { saveEmailSchedule, listEmailReportRegisters, listEmailReportRecipients, getEmailSettings } = useEmailReports()

  const form = reactive({
    name: '',
    period: 'today',
    format: 'csv',
    registerIds: [],
    frequency: 'daily',
    time: '22:00',
    weekdays: [],
    monthDay: 1,
    recipients: [],
    bcc: [],
    note: '',
    skipEmpty: true,
    enabled: true,
  })

  const registers = ref([])
  const pastRecipients = ref([])
  const registerOptions = computed(() => registers.value.map(r => ({
    title: `${r.branch_name} · ${r.code}`,
    value: r.id,
  })))

  function minutesToTime (minutes) {
    return `${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}`
  }

  // Fresh each time it opens: the schedule being edited, or defaults.
  watch(open, async isOpen => {
    if (!isOpen) return
    const s = props.schedule
    Object.assign(form, s
      ? {
        name: s.name,
        period: s.period,
        format: s.format || 'csv',
        registerIds: [...(s.register_ids ?? [])],
        frequency: s.frequency,
        time: minutesToTime(s.at_minute),
        weekdays: [...(s.weekdays ?? [])],
        monthDay: s.month_day || 1,
        recipients: s.recipients.split(',').map(r => r.trim()).filter(Boolean),
        bcc: (s.bcc || '').split(',').map(r => r.trim()).filter(Boolean),
        note: s.note,
        skipEmpty: s.skip_empty,
        enabled: s.enabled,
      }
      : {
        name: '',
        period: 'today',
        format: 'csv',
        registerIds: [],
        frequency: 'daily',
        time: '22:00',
        weekdays: [1, 2, 3, 4, 5, 6],
        monthDay: 1,
        recipients: [],
        bcc: [],
        note: '',
        skipEmpty: true,
        enabled: true,
      })
    try {
      const [list, past, settings] = await Promise.all([
        listEmailReportRegisters(),
        listEmailReportRecipients().catch(() => ({ recipients: [], last: [] })),
        s ? null : getEmailSettings().catch(() => null),
      ])
      registers.value = list
      pastRecipients.value = past.recipients
      if (!s) {
        form.format = settings?.settings?.email_attachment_format || 'csv'
        const defaults = String(settings?.settings?.email_default_recipients || '').split(',').map(r => r.trim()).filter(Boolean)
        form.recipients = defaults.length > 0 ? defaults : [...past.last]
      }
    } catch (error) {
      notify.error(error.message)
    }
  })

  const invalidRecipients = computed(() => form.recipients.filter(r => !EMAIL_RE.test(String(r).trim())))

  const invalidBcc = computed(() => form.bcc.filter(r => !EMAIL_RE.test(String(r).trim())))

  const whenSummary = computed(() => {
    const at = form.time || '--:--'
    if (form.frequency === 'weekly') {
      const days = form.weekdays.toSorted((a, b) => a - b).map(d => t(WEEKDAYS[d]))
      return days.length > 0 ? t('Every {days} at {time}', { days: days.join(', '), time: at }) : t('Pick at least one day')
    }
    if (form.frequency === 'monthly') {
      return t('Every month on day {day} at {time}', { day: form.monthDay, time: at })
    }
    return t('Every day at {time}', { time: at })
  })

  const canSave = computed(() =>
    form.recipients.length > 0
    && invalidRecipients.value.length === 0
    && invalidBcc.value.length === 0
    && /^\d{2}:\d{2}$/.test(form.time)
    && (form.frequency !== 'weekly' || form.weekdays.length > 0),
  )

  const saving = ref(false)
  async function save () {
    saving.value = true
    try {
      const saved = await saveEmailSchedule(props.schedule?.id, {
        name: form.name,
        enabled: form.enabled,
        period: form.period,
        register_ids: form.registerIds,
        recipients: form.recipients.map(r => String(r).trim()).join(', '),
        bcc: form.bcc.map(r => String(r).trim()).join(', '),
        format: form.format,
        note: form.note,
        skip_empty: form.skipEmpty,
        frequency: form.frequency,
        time: form.time,
        weekdays: form.weekdays,
        month_day: form.monthDay,
      })
      emit('saved', saved)
      open.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }
</script>
