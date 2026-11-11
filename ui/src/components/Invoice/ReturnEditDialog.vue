<template>
  <v-dialog max-width="620" :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)">
    <v-card>
      <v-card-title class="pt-5 px-6">Edit refund · {{ invoice?.branch_invoice_code }}</v-card-title>
      <v-card-text class="px-6">
        <p class="text-body-2 text-medium-emphasis mb-4">
          Set how much each returned item gives back. The till changes the return, its total
          and the cash in its drawer, and notes why.
        </p>
        <v-checkbox
          v-model="fullPrice"
          color="primary"
          data-test="full-price"
          density="compact"
          hide-details
          label="Refund full price, ignoring any deal charge-back"
        />
        <div v-for="row in rows" :key="row.position" class="refund-row" data-test="refund-row">
          <div class="refund-name">
            <div class="text-body-1">{{ row.name }}</div>
            <div class="text-caption text-medium-emphasis">
              {{ row.code }} · was {{ formatMoney(row.credit) }} · at most {{ formatMoney(row.max) }}
            </div>
          </div>
          <v-text-field
            v-model="entered[row.position]"
            class="refund-input"
            density="compact"
            :disabled="fullPrice"
            :error-messages="problem(row)"
            label="Gives back"
            prefix="$"
            variant="outlined"
          />
        </div>
        <v-textarea
          v-model="reason"
          auto-grow
          class="mt-2"
          label="Reason (written on the return)"
          rows="2"
          variant="outlined"
        />
        <div class="refund-total" data-test="refund-total">
          Total {{ formatMoney(edit.oldTotal) }} → <strong>{{ formatMoney(edit.newTotal) }}</strong>
          <span v-if="edit.moved < 0" class="text-medium-emphasis">
            · {{ formatMoney(-edit.moved) }} more back to the customer</span>
          <span v-else-if="edit.moved > 0" class="text-medium-emphasis">
            · {{ formatMoney(edit.moved) }} less back</span>
        </div>
      </v-card-text>
      <v-card-actions class="px-6 pb-4">
        <v-spacer />
        <v-btn text="Cancel" variant="text" @click="emit('update:modelValue', false)" />
        <v-btn
          color="primary"
          :disabled="!canSend"
          :loading="sending"
          text="Send to till"
          variant="flat"
          @click="send"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import { useBranchCommands } from '@/composables/useBranchCommands'
  import { useNotify } from '@/composables/useNotify'
  import { formatMoney } from '@/utils/money'
  import {
    creditProblem,
    fullPriceEntries,
    parseCredit,
    refundEdit,
    refundEditPayload,
    refundEditSentMessage,
    refundRows,
  } from '@/utils/returnEdit'

  const props = defineProps({
    modelValue: { type: Boolean, default: false },
    invoice: { type: Object, default: null },
    items: { type: Array, default: () => [] },
    branchName: { type: String, default: '' },
  })
  const emit = defineEmits(['update:modelValue', 'sent'])

  const notify = useNotify()
  const { sendCommand } = useBranchCommands()
  const rows = computed(() => refundRows(props.items))
  const entered = ref<Record<number, string>>({})
  const reason = ref('')
  const sending = ref(false)
  const fullPrice = ref(false)
  let typed: Record<number, string> = {} // what was typed, kept while full price is on

  watch(
    () => [props.modelValue, rows.value] as const,
    ([open]) => {
      if (!open) {return}
      entered.value = Object.fromEntries(rows.value.map(r => [r.position, (r.credit / 100).toFixed(2)]))
      reason.value = ''
      fullPrice.value = false
    },
    { immediate: true },
  )

  watch(fullPrice, on => {
    if (on) {
      typed = { ...entered.value }
      entered.value = fullPriceEntries(rows.value)
    } else {
      entered.value = typed
    }
  })

  function problem (row: any) {
    return creditProblem(row, parseCredit(entered.value[row.position]))
  }
  const credits = computed(() => {
    const out: Record<number, number> = {}
    for (const row of rows.value) {
      const cents = parseCredit(entered.value[row.position])
      if (cents !== null) {out[row.position] = cents}
    }
    return out
  })
  const edit = computed(() => refundEdit(props.invoice, rows.value, credits.value))
  const canSend = computed(() => edit.value.lines.length > 0 && rows.value.every(r => !problem(r)))

  async function send () {
    if (!canSend.value || sending.value) {return}
    sending.value = true
    try {
      await sendCommand(props.invoice!.branch_id, 'edit_return_invoice', refundEditPayload(props.invoice, edit.value.lines, reason.value))
      notify.success(refundEditSentMessage(props.invoice, props.branchName))
      emit('sent')
      emit('update:modelValue', false)
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      sending.value = false
    }
  }
</script>

<style scoped>
.refund-row { display: grid; grid-template-columns: 1fr 160px; gap: 12px; align-items: start; margin-bottom: 8px; }
.refund-name { padding-top: 4px; }
.refund-total { font-size: 1rem; margin-top: 4px; }
</style>
