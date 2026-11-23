<template>
  <v-dialog max-width="720" :model-value="modelValue" @update:model-value="close">
    <v-card>
      <v-card-title class="px-6 pt-5">{{ t('Change settlement') }}</v-card-title>
      <v-card-subtitle class="px-6">{{ invoice?.branch_invoice_code }}, {{ branchName }}</v-card-subtitle>
      <v-card-text class="px-6">
        <div v-if="phase === 'loading'" class="d-flex justify-center pa-6">
          <v-progress-circular indeterminate />
        </div>
        <template v-else-if="phase === 'edit'">
          <p class="text-body-2 mb-4">
            {{ t("How the {amount} was paid. The till moves the money between its payment methods in the sale's own shift, and an On Account part becomes what the client owes.", { amount: formatCents(invoice?.grand_total ?? 0) }) }}
          </p>
          <PaymentEntryPanel
            v-model="entries"
            :currencies="currencies"
            :due-cents="invoice?.grand_total ?? 0"
            :methods="methods"
          />
          <v-textarea v-model="reason" class="mt-4" :label="t('Reason (optional)')" rows="2" />
        </template>
        <PreparingState
          v-else-if="phase === 'waiting'"
          kind="receipt"
          min-height="220px"
          :title="t('Waiting for {branch} to change the settlement', { branch: branchName })"
        />
        <v-alert
          v-else-if="phase === 'result'"
          :text="resultText"
          :type="result?.status === 'success' ? 'success' : result?.status === 'failed' ? 'error' : 'info'"
          variant="tonal"
        />
      </v-card-text>
      <v-card-actions class="px-6 pb-4">
        <v-spacer />
        <template v-if="phase === 'edit' || phase === 'loading'">
          <v-btn :text="t('Cancel')" @click="close" />
          <v-btn
            color="primary"
            data-test="send-settlement"
            :disabled="!status.done || phase !== 'edit'"
            :loading="sending"
            :text="t('Change settlement')"
            @click="send"
          />
        </template>
        <template v-else-if="phase === 'result' && result?.status === 'failed'">
          <v-btn :text="t('Close')" @click="close" />
          <v-btn color="primary" :text="t('Edit and send again')" @click="phase = 'edit'" />
        </template>
        <v-btn v-else :text="t('Close')" @click="close" />
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
  import { t } from '@/i18n'
  import { entriesFromSettlement, formatCents, paymentStatus, toPayload } from '@/utils/paymentEntry'
  import PaymentEntryPanel from './PaymentEntryPanel.vue'

  const props = defineProps({
    modelValue: { type: Boolean, default: false },
    // A branch_invoices row (a sale).
    invoice: { type: Object, default: null },
    // Its current split: [{ account_name, amount }].
    payments: { type: Array, default: () => [] },
    branchName: { type: String, default: '' },
  })
  const emit = defineEmits(['update:modelValue', 'changed'])

  const { sendCommand } = useBranchCommands()
  const { loadPaymentSetup, waitForCommand } = usePaymentEntry()
  const notify = useNotify()

  const phase = ref('loading')
  const methods = ref([])
  const currencies = ref([])
  const entries = ref([])
  const reason = ref('')
  const sending = ref(false)
  const result = ref(null)
  let session = 0

  const status = computed(() => paymentStatus(entries.value, props.invoice?.grand_total ?? 0, currencies.value))
  const resultText = computed(() => {
    if (result.value?.status === 'success') return t('Settlement changed.')
    if (result.value?.status === 'failed') return result.value.error || t('The till reported a failure with no message.')
    return t("Still waiting: the till may be offline. It changes the settlement the next time it's online.")
  })

  watch(() => props.modelValue, async open => {
    if (!open || !props.invoice) return
    const mine = ++session
    phase.value = 'loading'
    reason.value = ''
    result.value = null
    try {
      const setup = await loadPaymentSetup(props.invoice.branch_id)
      if (mine !== session) return
      methods.value = setup.methods
      currencies.value = setup.currencies
      entries.value = entriesFromSettlement(props.payments, setup.methods, setup.currencies)
      phase.value = 'edit'
    } catch (error) {
      notify.error(error.message)
      close()
    }
  }, { immediate: true })

  async function send () {
    sending.value = true
    const mine = session
    try {
      const commands = await sendCommand(props.invoice.branch_id, 'change_invoice_settlement', {
        client_ref: props.invoice.client_ref,
        payments: toPayload(entries.value),
        reason: reason.value,
      })
      phase.value = 'waiting'
      result.value = await waitForCommand(props.invoice.branch_id, commands[0]?.id, () => mine !== session)
      if (result.value === null) return
      phase.value = 'result'
      if (result.value.status === 'success') emit('changed')
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
