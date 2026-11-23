<template>
  <v-dialog v-model="open" max-width="640" scrollable>
    <v-card data-no-discard-guard>
      <v-card-title class="px-6 pt-5 pb-2">{{ t('Filter invoices') }}</v-card-title>
      <v-card-text class="px-6">
        <div class="text-subtitle-2 mb-2">{{ t('Product on the invoice') }}</div>
        <v-row density="compact">
          <v-col cols="12" sm="6">
            <v-select
              v-model="draft.productField"
              data-test="filter-product-field"
              density="compact"
              hide-details
              :items="productFieldOptions()"
              :label="t('Search in')"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" sm="6">
            <v-select
              v-model="draft.productMatch"
              data-test="filter-product-match"
              density="compact"
              hide-details
              :items="productMatchOptions()"
              :label="t('Match')"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12">
            <SuggestField
              v-model="draft.product"
              autofocus
              data-test="filter-product"
              field="product"
              icon="mdi-tag-outline"
              :label="t('Code, name or barcode (* is a wildcard)')"
              :product-field="draft.productField"
              :suggest="suggest"
              @enter="apply"
            />
          </v-col>
        </v-row>

        <div class="text-subtitle-2 mt-5 mb-2">{{ t('Where and what') }}</div>
        <v-row density="compact">
          <v-col cols="12" sm="6">
            <v-select
              v-model="draft.branchId"
              clearable
              data-test="filter-branch"
              density="compact"
              hide-details
              item-title="name"
              item-value="id"
              :items="branches"
              :label="t('Branch')"
              persistent-placeholder
              :placeholder="t('All branches')"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" sm="6">
            <v-select
              v-model="draft.registerId"
              clearable
              data-test="filter-register"
              density="compact"
              hide-details
              item-title="label"
              item-value="id"
              :items="registerChoices"
              :label="t('Register')"
              persistent-placeholder
              :placeholder="t('All registers')"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" sm="6">
            <v-select
              v-model="draft.kind"
              clearable
              data-test="filter-kind"
              density="compact"
              hide-details
              :items="kindOptions()"
              :label="t('Type')"
              persistent-placeholder
              :placeholder="t('All types')"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" sm="6">
            <v-select
              v-model="draft.status"
              clearable
              data-test="filter-status"
              density="compact"
              hide-details
              :items="statusOptions()"
              :label="t('Status')"
              persistent-placeholder
              :placeholder="t('Any status')"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12">
            <SuggestField
              v-model="draft.client"
              data-test="filter-client"
              field="client"
              icon="mdi-account-search"
              :label="t('Client name or phone (* is a wildcard)')"
              :suggest="suggest"
              @enter="apply"
            />
          </v-col>
          <v-col cols="12">
            <SuggestField
              v-model="draft.salesperson"
              :branch-id="draft.branchId"
              data-test="filter-salesperson"
              field="salesperson"
              icon="mdi-account-tie"
              :label="t('Salesperson (* is a wildcard)')"
              :suggest="suggest"
              @enter="apply"
            />
          </v-col>
        </v-row>

        <div class="text-subtitle-2 mt-5 mb-2">{{ t('When and how much') }}</div>
        <v-row density="compact">
          <v-col cols="6">
            <v-text-field
              v-model="draft.dateFrom"
              data-test="filter-date-from"
              density="compact"
              hide-details
              :label="t('From')"
              :max="draft.dateTo || undefined"
              type="date"
              variant="outlined"
            />
          </v-col>
          <v-col cols="6">
            <v-text-field
              v-model="draft.dateTo"
              data-test="filter-date-to"
              density="compact"
              hide-details
              :label="t('To')"
              :min="draft.dateFrom || undefined"
              type="date"
              variant="outlined"
            />
          </v-col>
          <v-col cols="6">
            <v-text-field
              v-model="draft.totalMin"
              data-test="filter-total-min"
              density="compact"
              hide-details
              inputmode="decimal"
              :label="t('Total from')"
              variant="outlined"
              @keydown.enter="apply"
            />
          </v-col>
          <v-col cols="6">
            <v-text-field
              v-model="draft.totalMax"
              data-test="filter-total-max"
              density="compact"
              hide-details
              inputmode="decimal"
              :label="t('Total to')"
              variant="outlined"
              @keydown.enter="apply"
            />
          </v-col>
        </v-row>
        <div class="text-caption text-medium-emphasis mt-2">{{ t("A return's total is negative.") }}</div>
        <div v-if="error" class="text-body-2 text-error mt-3" data-test="filter-error">{{ error }}</div>
      </v-card-text>
      <v-card-actions class="px-6 pb-4">
        <v-btn data-test="filter-clear" :text="t('Clear all')" variant="text" @click="clearAll" />
        <v-spacer />
        <v-btn :text="t('Cancel')" variant="text" @click="open = false" />
        <v-btn
          color="primary"
          data-test="filter-apply"
          :disabled="Boolean(error)"
          :text="t('Apply')"
          variant="flat"
          @click="apply"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import { t } from '@/i18n'
  import SuggestField from '@/components/Invoice/SuggestField.vue'
  import {
    emptyFilters,
    invoiceFilterError,
    kindOptions,
    productFieldOptions,
    productMatchOptions,
    statusOptions,
  } from '@/utils/invoiceFilters'

  const props = defineProps({
    filters: { type: Object, required: true },
    branches: { type: Array, default: () => [] },
    // { id, branch_id, label } for every register.
    registers: { type: Array, default: () => [] },
    // Completes the product, client and salesperson boxes (no completions when absent).
    suggestFn: { type: Function, default: async () => [] },
  })
  const suggest = args => props.suggestFn(args)
  const emit = defineEmits(['apply'])
  const open = defineModel({ type: Boolean, default: false })

  // The dialog edits a copy; the list only changes on Apply.
  const draft = ref(emptyFilters())
  watch(open, isOpen => {
    if (isOpen) draft.value = { ...emptyFilters(), ...props.filters }
  }, { immediate: true })

  const registerChoices = computed(() =>
    props.registers.filter(r => !draft.value.branchId || r.branch_id === draft.value.branchId))
  // A register of another branch can't stay picked once the branch changes.
  watch(() => draft.value.branchId, () => {
    if (draft.value.registerId && !registerChoices.value.some(r => r.id === draft.value.registerId)) {
      draft.value.registerId = null
    }
  })

  const error = computed(() => invoiceFilterError(draft.value))

  function apply () {
    if (error.value) return
    emit('apply', { ...draft.value, product: draft.value.product ?? '', client: draft.value.client ?? '', salesperson: draft.value.salesperson ?? '' })
    open.value = false
  }

  function clearAll () {
    draft.value = emptyFilters()
  }
</script>
