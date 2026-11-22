<template>
  <v-dialog max-width="520" :model-value="modelValue" @update:model-value="close">
    <v-card>
      <v-card-title class="px-6 pt-5">Open for exchange</v-card-title>
      <v-card-subtitle class="px-6">{{ invoice?.branch_invoice_code }}, {{ branchName }}</v-card-subtitle>
      <v-card-text class="px-6">
        <template v-if="phase === 'edit'">
          <p class="text-body-2 mb-4">
            The till lets this sale be exchanged past the branch's exchange window, until the end of the last day.
          </p>
          <v-radio-group v-model="mode" hide-details>
            <v-radio data-test="mode-today" label="Only today" value="today" />
            <v-radio data-test="mode-days" label="For a number of days" value="days" />
          </v-radio-group>
          <v-text-field
            v-if="mode === 'days'"
            v-model.number="days"
            class="mt-2"
            data-test="days"
            density="compact"
            :hint="'Counting today, up to ' + MAX_OPENING_DAYS"
            label="Days"
            :max="MAX_OPENING_DAYS"
            min="1"
            persistent-hint
            type="number"
            variant="outlined"
          />
          <p class="text-body-2 mt-4" data-test="until">
            <template v-if="untilDate">Open until the end of <strong>{{ formatOpeningDate(untilDate) }}</strong></template>
            <template v-else>Enter between 1 and {{ MAX_OPENING_DAYS }} days.</template>
          </p>
          <v-textarea v-model="reason" class="mt-2" label="Reason (optional)" rows="2" />
        </template>
        <PreparingState
          v-else-if="phase === 'waiting'"
          kind="receipt"
          min-height="180px"
          :title="`Waiting for ${branchName} to open the sale`"
        />
        <p v-else-if="phase === 'result'" class="text-body-2" :class="result?.status === 'failed' ? 'text-error' : ''" data-test="result">
          {{ resultText }}
        </p>
      </v-card-text>
      <v-card-actions class="px-6 pb-4">
        <v-spacer />
        <template v-if="phase === 'edit'">
          <v-btn text="Cancel" @click="close" />
          <v-btn
            color="primary"
            data-test="send-opening"
            :disabled="!untilDate"
            :loading="sending"
            text="Open for exchange"
            @click="send"
          />
        </template>
        <template v-else-if="phase === 'result' && result?.status === 'failed'">
          <v-btn text="Close" @click="close" />
          <v-btn color="primary" text="Edit and send again" @click="phase = 'edit'" />
        </template>
        <v-btn v-else text="Close" @click="close" />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import PreparingState from '@/components/PreparingState.vue'
  import { useBranchCommands } from '@/composables/useBranchCommands'
  import { useNotify } from '@/composables/useNotify'
  import { usePaymentEntry } from '@/composables/usePaymentEntry'
  import { formatOpeningDate, MAX_OPENING_DAYS, openingPayload, openingUntilDate } from '@/utils/exchangeOpening'

  const props = defineProps({
    modelValue: { type: Boolean, default: false },
    // A branch_invoices row (a sale).
    invoice: { type: Object, default: null },
    branchName: { type: String, default: '' },
  })
  const emit = defineEmits(['update:modelValue'])

  const { sendCommand } = useBranchCommands()
  const { waitForCommand } = usePaymentEntry()
  const notify = useNotify()

  const phase = ref('edit')
  const mode = ref('today')
  const days = ref(7)
  const reason = ref('')
  const sending = ref(false)
  const result = ref(null)
  const sentUntil = ref('')
  let session = 0

  const untilDate = computed(() => openingUntilDate(mode.value, days.value))
  const resultText = computed(() => {
    const until = formatOpeningDate(sentUntil.value)
    if (result.value?.status === 'success') return `Open for exchange on the till until the end of ${until}.`
    if (result.value?.status === 'failed') return result.value.error || 'The till reported a failure with no message.'
    return `Still waiting: the till may be offline. It opens the sale until ${until} the next time it's online.`
  })

  watch(() => props.modelValue, open => {
    if (!open) return
    session++
    phase.value = 'edit'
    mode.value = 'today'
    days.value = 7
    reason.value = ''
    result.value = null
  }, { immediate: true })

  async function send () {
    sending.value = true
    const mine = session
    try {
      const payload = openingPayload(props.invoice, mode.value, days.value, reason.value)
      sentUntil.value = payload.until_date
      const commands = await sendCommand(props.invoice.branch_id, 'open_invoice_exchange', payload)
      phase.value = 'waiting'
      result.value = await waitForCommand(props.invoice.branch_id, commands[0]?.id, () => mine !== session)
      if (result.value === null) return
      phase.value = 'result'
    } catch (error) {
      notify.error(error.message)
    } finally {
      sending.value = false
    }
  }

  function close () {
    session++
    emit('update:modelValue', false)
  }
</script>
