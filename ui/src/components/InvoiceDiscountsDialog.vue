<!--
  Invoice discounts (api/invoice_discounts.go): a discount on a whole sale
  once it reaches a total, "$150 or more, 15% off". The till takes it off the
  total after item discounts and deals, on top of them; when several apply,
  the one that saves the customer most wins.
-->
<template>
  <v-dialog v-model="open" max-width="760">
    <v-card>
      <v-card-title class="d-flex align-center ga-2">
        <span>{{ t('Invoice discounts') }}</span>
        <v-spacer />
        <v-btn
          v-if="canEdit"
          prepend-icon="mdi-plus"
          size="small"
          :text="t('New invoice discount')"
          variant="tonal"
          @click="openEdit(null)"
        />
      </v-card-title>
      <v-card-text>
        <p class="text-body-2 text-medium-emphasis mb-3">
          {{ t('Comes off the whole sale once its total (after item discounts and deals, before tax) reaches the minimum, on top of the item discounts. Only the best one applies.') }}
        </p>
        <div v-if="rules.length === 0" class="text-body-2 text-medium-emphasis">{{ t('No invoice discounts yet.') }}</div>
        <v-table v-else density="compact">
          <thead>
            <tr>
              <th>{{ t('Name') }}</th>
              <th>{{ t('Terms') }}</th>
              <th>{{ t('Branches') }}</th>
              <th>{{ t('Dates') }}</th>
              <th>{{ t('On') }}</th>
              <th v-if="canEdit" style="width: 96px" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="rule in rules" :key="rule.id">
              <td>{{ rule.name }}</td>
              <td>{{ ruleTerms(rule) }}</td>
              <td>{{ branchesText(rule) }}</td>
              <td class="text-no-wrap">{{ datesText(rule) }}</td>
              <td>
                <v-icon
                  :color="rule.is_active ? 'success' : 'grey'"
                  :icon="rule.is_active ? 'mdi-check-circle' : 'mdi-pause-circle'"
                  size="small"
                />
              </td>
              <td v-if="canEdit" class="text-end text-no-wrap">
                <v-btn
                  density="comfortable"
                  icon="mdi-pencil"
                  size="small"
                  variant="text"
                  @click="openEdit(rule)"
                />
                <v-btn
                  color="error"
                  density="comfortable"
                  icon="mdi-delete"
                  size="small"
                  :title="t('Delete')"
                  variant="text"
                  @click="remove(rule)"
                />
              </td>
            </tr>
          </tbody>
        </v-table>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn :text="t('Close')" variant="text" @click="open = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="editDialog" max-width="520">
    <v-card>
      <v-card-title>{{ form.id ? t('Edit invoice discount') : t('New invoice discount') }}</v-card-title>
      <v-card-text>
        <v-text-field
          v-model="form.name"
          density="compact"
          :label="t('Name (printed on the receipt)')"
          variant="outlined"
        />
        <div class="d-flex flex-wrap ga-3">
          <v-text-field
            v-model="form.minTotal"
            density="compact"
            :label="t('When the invoice is at least')"
            min="0"
            prefix="$"
            style="max-width: 220px"
            type="number"
            variant="outlined"
          />
          <v-btn-toggle
            v-model="form.kind"
            color="primary"
            density="compact"
            mandatory
            variant="outlined"
          >
            <v-btn :text="t('% off')" value="percent" />
            <v-btn :text="t('$ off')" value="amount" />
          </v-btn-toggle>
          <v-text-field
            v-if="form.kind === 'percent'"
            v-model.number="form.percent"
            density="compact"
            :label="t('Off')"
            max="100"
            min="1"
            style="max-width: 130px"
            suffix="%"
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-else
            v-model="form.amount"
            density="compact"
            :label="t('Off')"
            min="0"
            prefix="$"
            style="max-width: 150px"
            type="number"
            variant="outlined"
          />
        </div>
        <v-switch
          v-model="form.all_branches"
          color="primary"
          density="compact"
          hide-details
          :label="t('All branches')"
        />
        <v-select
          v-if="!form.all_branches"
          v-model="form.branch_ids"
          chips
          class="mt-2"
          closable-chips
          density="compact"
          item-title="name"
          item-value="id"
          :items="branches"
          :label="t('Branches')"
          multiple
          variant="outlined"
        />
        <div class="d-flex flex-wrap ga-3 mt-2">
          <v-text-field
            v-model="form.validFrom"
            clearable
            density="compact"
            :hint="t('Empty: from now')"
            :label="t('From')"
            persistent-hint
            style="max-width: 200px"
            type="date"
            variant="outlined"
          />
          <v-text-field
            v-model="form.validTo"
            clearable
            density="compact"
            :hint="t('Empty: no end')"
            :label="t('Until (included)')"
            persistent-hint
            style="max-width: 200px"
            type="date"
            variant="outlined"
          />
        </div>
        <v-switch
          v-if="form.kind === 'amount'"
          v-model="form.repeats"
          color="primary"
          data-test="repeats"
          density="compact"
          :hint="t('$50 off every $100: a $200 invoice gets $100 off. Needs a minimum to count by.')"
          :label="t('Again for every minimum reached')"
          persistent-hint
        />
        <v-switch
          v-model="form.replaces_item_discounts"
          color="primary"
          data-test="replaces-item-discounts"
          density="compact"
          :hint="t('$50 off every $100, else 20%: when it applies, items lose their list discounts and the till charges whichever saves more.')"
          :label="t('Not with item discounts')"
          persistent-hint
        />
        <v-switch
          v-model="form.cashier_picks"
          color="primary"
          data-test="cashier-picks"
          density="compact"
          :hint="t('Like a student discount: it shows under the till\'s Rewards button, and the cashier turns it on for a sale. Off, it applies by itself once the total is reached.')"
          :label="t('The cashier picks it for a sale')"
          persistent-hint
        />
        <v-switch
          v-model="form.is_active"
          color="primary"
          density="compact"
          hide-details
          :label="t('On')"
        />
        <v-alert class="mt-2" density="compact" type="info" variant="tonal">
          {{ ruleTerms(formRule) }}
        </v-alert>
        <v-alert
          v-if="form.id"
          class="mt-2"
          density="compact"
          type="warning"
          variant="tonal"
        >
          {{ t('The change reaches the tills on their next sync. Sales already made keep the discount they got.') }}
        </v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn :text="t('Cancel')" variant="text" @click="editDialog = false" />
        <v-btn
          color="primary"
          :loading="saving"
          :text="t('Save')"
          variant="flat"
          @click="save"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useBranches } from '@/composables/useBranches'
  import { confirmDialog } from '@/composables/useDialog'
  import { useNotify } from '@/composables/useNotify'
  import { t } from '@/i18n'
  import { API_BASE } from '@/config'
  import { formatMoney } from '@/utils/money'

  const open = defineModel({ type: Boolean, default: false })
  defineProps({ canEdit: { type: Boolean, default: false } })
  const notify = useNotify()

  async function requestJSON (path, options) {
    const res = await authFetch(`${API_BASE}${path}`, options)
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || t('Request failed with status {status}', { status: res.status }))
    return data
  }
  const { branches, fetchBranches } = useBranches()

  const rules = ref([])

  async function load () {
    const data = await requestJSON('/invoice_discounts')
    rules.value = data.invoice_discounts ?? []
  }

  watch(open, value => {
    if (!value) return
    load().catch(error => notify.error(error.message))
    fetchBranches().catch(error => notify.error(error.message))
  })

  function ruleTerms (rule) {
    if (!rule) return ''
    const terms = baseTerms(rule)
    return rule.replaces_item_discounts ? t('{terms}, instead of item discounts', { terms }) : terms
  }

  function offText (rule) {
    return rule.kind === 'percent'
      ? t('{percent}% off', { percent: rule.percent })
      : t('${amount} off', { amount: formatMoney(rule.amount) })
  }

  function baseTerms (rule) {
    const off = offText(rule)
    if (rule.repeats) {
      const each = t('${amount} off for every ${min} of the invoice', { amount: formatMoney(rule.amount), min: formatMoney(rule.min_total) })
      return rule.cashier_picks ? t('When the cashier picks it: {each}', { each }) : each
    }
    if (rule.cashier_picks) {
      const on = rule.min_total > 0 ? t(' on invoices of ${min} or more', { min: formatMoney(rule.min_total) }) : ''
      return t('When the cashier picks it: {off} the whole invoice{on}', { off, on })
    }
    return t('Invoices of ${min} or more get {off} the whole invoice', { min: formatMoney(rule.min_total), off })
  }

  function branchesText (rule) {
    if (rule.all_branches) return t('All')
    return rule.branch_ids
      .map(id => branches.value.find(b => b.id === id)?.name ?? t('#{id}', { id }))
      .join(', ')
  }

  // The API sends a date as {Time, Valid} (sql.NullTime).
  function when (value) {
    return value?.Valid ? value.Time : null
  }

  // A date column shows the day; the API takes the day's start (from) and
  // the next day's start (until, which is included).
  function day (value) {
    if (!value) return ''
    const d = new Date(value)
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }

  function datesText (rule) {
    const from = day(when(rule.valid_from))
    const to = untilDay(when(rule.valid_to))
    if (!from && !to) return t('Always')
    return `${from || '…'} → ${to || '…'}`
  }

  function untilDay (value) {
    if (!value) return ''
    const d = new Date(value)
    d.setDate(d.getDate() - 1)
    return day(d)
  }

  const editDialog = ref(false)
  const saving = ref(false)
  function blank () {
    return {
      id: 0, name: '', minTotal: '', kind: 'percent', percent: 10, amount: '',
      all_branches: true, branch_ids: [], validFrom: '', validTo: '', is_active: true, cashier_picks: false, repeats: false,
      replaces_item_discounts: false,
    }
  }
  const form = reactive(blank())

  const formRule = computed(() => ({
    kind: form.kind,
    min_total: Math.round(Number(form.minTotal) * 100) || 0,
    percent: form.kind === 'percent' ? Number(form.percent) || 0 : 0,
    amount: form.kind === 'amount' ? Math.round(Number(form.amount) * 100) || 0 : 0,
    cashier_picks: Boolean(form.cashier_picks),
    replaces_item_discounts: Boolean(form.replaces_item_discounts),
    // Repeating counts by the minimum, so only an amount with one repeats.
    repeats: Boolean(form.repeats) && form.kind === 'amount' && Math.round(Number(form.minTotal) * 100) > 0,
  }))

  function openEdit (rule) {
    Object.assign(form, rule
      ? {
        id: rule.id, name: rule.name, kind: rule.kind, percent: rule.percent || 10,
        minTotal: String(rule.min_total / 100), amount: rule.amount ? String(rule.amount / 100) : '',
        all_branches: rule.all_branches, branch_ids: [...rule.branch_ids],
        validFrom: day(when(rule.valid_from)),
        validTo: untilDay(when(rule.valid_to)),
        is_active: rule.is_active,
        cashier_picks: Boolean(rule.cashier_picks),
        replaces_item_discounts: Boolean(rule.replaces_item_discounts),
        repeats: Boolean(rule.repeats),
      }
      : blank())
    editDialog.value = true
  }

  // Local midnight of the day, as the till's clock reads it.
  function startOfDay (value) {
    if (!value) return null
    const [y, m, d] = value.split('-').map(Number)
    return new Date(y, m - 1, d).toISOString()
  }

  function startOfNextDay (value) {
    if (!value) return null
    const [y, m, d] = value.split('-').map(Number)
    return new Date(y, m - 1, d + 1).toISOString()
  }

  async function save () {
    saving.value = true
    try {
      const body = {
        name: form.name.trim(), ...formRule.value,
        all_branches: form.all_branches, branch_ids: form.all_branches ? [] : form.branch_ids,
        valid_from: startOfDay(form.validFrom), valid_to: startOfNextDay(form.validTo),
        is_active: form.is_active,
      }
      await requestJSON(form.id ? `/invoice_discounts/${form.id}` : '/invoice_discounts', {
        method: form.id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      editDialog.value = false
      await load()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  async function remove (rule) {
    if (!(await confirmDialog({ text: t('Delete "{name}"? The tills stop giving it on their next sync.', { name: rule.name }), okText: t('Delete'), danger: true }))) return
    try {
      await requestJSON(`/invoice_discounts/${rule.id}`, { method: 'DELETE' })
      await load()
    } catch (error) {
      notify.error(error.message)
    }
  }
</script>
