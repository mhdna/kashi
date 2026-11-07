<template>
  <v-btn
    v-if="visible"
    class="invoice-print-button"
    :icon="text ? undefined : 'mdi-printer'"
    :loading="printing"
    :prepend-icon="text ? 'mdi-printer' : undefined"
    :size="text ? undefined : 'small'"
    :text="text ? 'Print on till' : undefined"
    :title="PRINT_ON_TILL_TITLE"
    variant="text"
    @click.stop="print"
  />
</template>

<script setup lang="ts">
  // Prints a branch invoice's receipt on the till that rang it up (a
  // reprint_invoice command, see utils/invoicePrint.js). Queuing commands
  // takes Branches edit, like the close-shift print button on Shifts.
  import { computed, ref } from 'vue'
  import { useBranchCommands } from '@/composables/useBranchCommands'
  import { useNotify } from '@/composables/useNotify'
  import { useAuthStore } from '@/stores/auth'
  import { canPrintOnTill, PRINT_ON_TILL_TITLE, printOnTillSentMessage } from '@/utils/invoicePrint'

  const props = defineProps({
    invoice: { type: Object, default: null },
    branchName: { type: String, default: '' },
    // A labelled button (the details dialog) instead of an icon (a table row).
    text: { type: Boolean, default: false },
  })

  const auth = useAuthStore()
  const notify = useNotify()
  const { printBranchInvoice } = useBranchCommands()
  const printing = ref(false)

  const visible = computed(() => auth.canEdit('branches') && canPrintOnTill(props.invoice))

  async function print () {
    if (printing.value || !visible.value) {return}
    printing.value = true
    try {
      await printBranchInvoice(props.invoice)
      notify.success(printOnTillSentMessage(props.invoice, props.branchName))
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      printing.value = false
    }
  }
</script>
