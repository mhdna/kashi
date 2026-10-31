<template>
  <v-dialog max-width="560" :model-value="modelValue" scrollable @update:model-value="emit('update:modelValue', $event)">
    <v-card v-if="item" rounded="lg">
      <!-- The header carries the row's severity colour, so an urgent row
           reads as one before anything in it is read. -->
      <v-card-title class="detail-head px-4 py-3" :class="severity && `detail-head--${severity}`">
        <div class="d-flex align-center ga-2">
          <div class="min-w-0">
            <div class="text-subtitle-1 font-weight-medium">{{ title }}</div>
            <div class="text-caption text-medium-emphasis text-truncate">
              {{ [item.branch_name, register].filter(Boolean).join(' · ') }}
            </div>
          </div>
          <v-spacer />
          <div
            v-if="hasAmount"
            class="text-subtitle-1 font-weight-medium text-no-wrap"
            :class="amountClass(item.amount)"
          >
            {{ formatAmount(item.amount, item.currency_code) }}
          </div>
          <v-btn
            aria-label="Close"
            icon="mdi-close"
            size="small"
            variant="text"
            @click="emit('update:modelValue', false)"
          />
        </div>
      </v-card-title>
      <v-divider />

      <v-card-text class="pa-4">
        <div v-if="loading" class="d-flex justify-center pa-4">
          <PreparingState kind="data" min-height="160px" title="Loading details" />
        </div>
        <v-alert v-else-if="error" density="compact" type="warning" variant="tonal">{{ error }}</v-alert>

        <template v-else-if="data">
          <!-- Shift closed: what the till expected, what was counted, and the
               difference, per payment method; then the receipt photo. -->
          <template v-if="data.shift">
            <div class="facts">
              <Fact label="Closed by" :value="data.shift.closing_person_name" />
              <Fact label="Opened" :value="dateTime(nullTime(data.shift.opened_at))" />
              <Fact label="Closed" :value="dateTime(nullTime(data.shift.closed_at))" />
              <Fact label="Opening float" :value="`${formatAmount(data.shift.opening_float_usd)} · ${formatLBP(data.shift.opening_float_lbp)}`" />
            </div>
            <v-table class="mt-3" density="compact">
              <thead>
                <tr>
                  <th />
                  <th class="text-end">Expected</th>
                  <th class="text-end">Counted</th>
                  <th class="text-end">Variance</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="method in shiftMethods" :key="method.key">
                  <td>{{ method.label }}</td>
                  <td class="text-end">{{ method.format(data.shift[`expected_${method.key}`]) }}</td>
                  <td class="text-end">{{ method.format(data.shift[`counted_${method.key}`]) }}</td>
                  <td class="text-end font-weight-medium" :class="amountClass(data.shift[`variance_${method.key}`])">
                    {{ data.shift[`variance_${method.key}`] ? method.format(data.shift[`variance_${method.key}`]) : '—' }}
                  </td>
                </tr>
              </tbody>
            </v-table>
            <div class="section-label">Close-shift receipt</div>
            <PhotoThumb
              v-if="data.receipt_image_id"
              :src="thumbs[`shift_receipt-${data.receipt_image_id}`]"
              @click="openPhoto('shift_receipt', data.receipt_image_id, 'Close-shift receipt')"
            />
            <div v-else class="text-body-2 text-medium-emphasis">The till sent no receipt photo.</div>
          </template>

          <!-- Invoice: who sold it, the lines and how it was paid. -->
          <template v-else-if="data.invoice && data.items">
            <div class="facts">
              <Fact label="Invoice" :value="data.invoice.branch_invoice_code" />
              <Fact label="Kind" :value="capitalize(data.invoice.kind)" />
              <Fact label="Salesperson" :value="data.invoice.salesperson_name" />
              <Fact label="Sold at" :value="dateTime(data.invoice.occurred_at)" />
              <Fact v-if="data.invoice.discount" label="Discount" :value="`${data.invoice.discount}%`" />
              <Fact v-if="data.invoice.loyalty_points_delta" label="Loyalty points" :value="data.invoice.loyalty_points_delta" />
              <Fact v-if="data.invoice.cancelled_at?.Valid" label="Cancelled" :value="dateTime(data.invoice.cancelled_at.Time)" />
            </div>
            <div class="section-label">Items</div>
            <v-table density="compact">
              <tbody>
                <tr v-for="line in data.items" :key="line.id">
                  <td>
                    {{ line.product_name || 'Unknown product' }}
                    <div v-if="line.product_code" class="text-caption text-medium-emphasis">{{ line.product_code }}</div>
                  </td>
                  <td class="text-end text-no-wrap">{{ line.quantity }} × {{ formatAmount(line.unit_price) }}</td>
                  <td class="text-end text-no-wrap font-weight-medium">{{ formatAmount(line.line_total) }}</td>
                </tr>
                <tr v-if="data.items.length === 0">
                  <td class="text-medium-emphasis">No lines reported.</td>
                </tr>
              </tbody>
            </v-table>
            <Payments :payments="data.payments" />
          </template>

          <!-- Settlement changed: the sale it re-settled and its new split. -->
          <template v-else-if="data.settlement">
            <div class="facts">
              <Fact label="Invoice" :value="data.invoice?.branch_invoice_code || data.settlement.sale_client_ref" />
              <Fact v-if="data.invoice" label="Salesperson" :value="data.invoice.salesperson_name" />
              <Fact v-if="data.invoice" label="Sold at" :value="dateTime(data.invoice.occurred_at)" />
              <Fact label="Changed at" :value="dateTime(data.settlement.occurred_at)" />
              <Fact label="Status" :value="data.settlement.status" />
            </div>
            <Payments label="New payments" :payments="data.payments" />
          </template>

          <!-- Expense: what it was for, paid from where, and its photos. -->
          <template v-else-if="data.expense">
            <div class="facts">
              <Fact label="Description" :value="data.expense.description" />
              <Fact label="Paid from" :value="data.expense.branch_cashbox_account_name?.String" />
              <Fact label="Spent at" :value="dateTime(data.expense.occurred_at)" />
            </div>
            <div class="section-label">Photos</div>
            <div v-if="data.image_ids?.length" class="d-flex flex-wrap ga-2">
              <PhotoThumb
                v-for="imageId in data.image_ids"
                :key="imageId"
                :src="thumbs[`expense_image-${imageId}`]"
                @click="openPhoto('expense_image', imageId, 'Expense photo')"
              />
            </div>
            <div v-else class="text-body-2 text-medium-emphasis">No photo attached.</div>
          </template>

          <template v-else-if="data.loan">
            <div class="facts">
              <Fact label="Description" :value="data.loan.description" />
              <Fact label="Account" :value="data.loan.branch_cashbox_account_name?.String" />
              <Fact label="Taken at" :value="dateTime(data.loan.occurred_at)" />
            </div>
          </template>

          <template v-else-if="data.event">
            <div class="facts">
              <Fact label="Name" :value="data.event.salesperson_name" />
              <Fact label="Punch" :value="data.event.type" />
              <Fact label="Date" :value="data.event.event_date" />
              <Fact label="Time" :value="data.event.event_time" />
              <Fact label="Status" :value="data.event.status" />
              <Fact v-if="data.event.changed_by" label="Changed by" :value="`${data.event.changed_by}${data.event.change_note ? ` · ${data.event.change_note}` : ''}`" />
            </div>
          </template>

          <!-- Attendance changed: a cashier's request, or a manager's
               decision on one. -->
          <template v-else-if="data.change">
            <div class="facts">
              <Fact label="Name" :value="data.change.salesperson_name" />
              <Fact label="What" :value="data.change.kind === 'complaint' ? 'Change requested' : 'Request reviewed'" />
              <Fact label="Status" :value="capitalize(data.change.status)" />
              <Fact label="Attendance date" :value="data.change.attendance_date" />
              <Fact label="Change" :value="changeSummary(data.change)" />
              <Fact label="Note" :value="data.change.note" />
              <Fact label="By" :value="data.change.actor" />
              <Fact label="At" :value="dateTime(data.change.occurred_at)" />
            </div>
          </template>
        </template>

        <div class="text-caption text-medium-emphasis mt-4">Synced {{ dateTime(item.synced_at) }}</div>
      </v-card-text>

      <template v-if="canResolve && (item.pinned || item.resolved)">
        <v-divider />
        <v-card-actions class="px-4">
          <template v-if="item.resolved">
            <v-icon color="success" icon="mdi-check-circle" size="18" />
            <span class="text-caption text-medium-emphasis">
              Resolved{{ item.resolved_by ? ` by ${item.resolved_by}` : '' }}.
            </span>
            <v-spacer />
            <v-btn
              :loading="resolving"
              text="Mark unresolved"
              variant="text"
              @click="emit('unresolve', item)"
            />
          </template>
          <template v-else>
            <span class="text-caption text-medium-emphasis">Pinned on Home until resolved.</span>
            <v-spacer />
            <v-btn
              color="error"
              :loading="resolving"
              prepend-icon="mdi-check-circle-outline"
              text="Mark resolved"
              variant="tonal"
              @click="emit('resolve', item)"
            />
          </template>
        </v-card-actions>
      </template>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, defineComponent, h, onUnmounted, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { PHOTO_FETCH } from '@/composables/usePhotoActions'
  import { API_BASE } from '@/config'
  import { formatAmount, formatLBP } from '@/utils/money'

  // Everything about one Recent activity row, from /home/activities/detail.
  const props = defineProps({
    modelValue: { type: Boolean, default: false },
    // The feed row: activity, id, branch_name, amount, synced_at...
    item: { type: Object, default: null },
    title: { type: String, default: '' },
    // 'high' | 'medium' | 'low' | null, as the feed tints the row.
    severity: { type: String, default: null },
    canResolve: { type: Boolean, default: false },
    resolving: { type: Boolean, default: false },
  })
  const emit = defineEmits(['update:modelValue', 'open-photo', 'resolve', 'unresolve'])

  const loading = ref(false)
  const error = ref('')
  const data = ref(null)
  const thumbs = ref({})

  const register = computed(() => data.value?.register || '')
  const hasAmount = computed(() =>
    props.item?.amount != null && props.item.amount !== 0 && !String(props.item.activity).startsWith('Attendance'))

  // The tills report the dollar drawers in cents but LBP in whole pounds,
  // as printed on the close-shift receipt.
  const shiftMethods = [
    { key: 'usd', label: 'USD', format: cents => formatAmount(cents) },
    { key: 'lbp', label: 'LBP', format: formatLBP },
    { key: 'visa', label: 'Visa', format: cents => formatAmount(cents) },
    { key: 'whish', label: 'Whish', format: cents => formatAmount(cents) },
  ]

  // The photo endpoints, by the kind name the home page's photo viewer uses.
  const PHOTO_FILES = {
    shift_receipt: id => `/branch_shift_receipt_images/${id}/file`,
    expense_image: id => `/branch_expense_images/${id}/file`,
  }

  let loadSeq = 0
  async function load () {
    const item = props.item
    if (!props.modelValue || !item) return
    const seq = ++loadSeq
    loading.value = true
    error.value = ''
    data.value = null
    try {
      const params = new URLSearchParams({ activity: item.activity, id: String(item.id) })
      const res = await authFetch(`${API_BASE}/home/activities/detail?${params}`)
      const body = await res.json().catch(() => ({}))
      if (seq !== loadSeq) return
      if (!res.ok) throw new Error(body.error || `Couldn't load the details (${res.status})`)
      data.value = body
      loadThumbs(body)
    } catch (error_) {
      if (seq === loadSeq) error.value = error_.message
    } finally {
      if (seq === loadSeq) loading.value = false
    }
  }
  watch(() => [props.modelValue, props.item?.activity, props.item?.id], load, { immediate: true })

  async function loadThumb (kind, id) {
    const key = `${kind}-${id}`
    if (thumbs.value[key]) return
    const res = await authFetch(`${API_BASE}${PHOTO_FILES[kind](id)}?w=240`, PHOTO_FETCH).catch(() => null)
    if (!res?.ok) return
    thumbs.value = { ...thumbs.value, [key]: URL.createObjectURL(await res.blob()) }
  }
  function loadThumbs (body) {
    if (body.receipt_image_id) loadThumb('shift_receipt', body.receipt_image_id)
    for (const id of body.image_ids ?? []) loadThumb('expense_image', id)
  }

  onUnmounted(() => {
    for (const url of Object.values(thumbs.value)) URL.revokeObjectURL(url)
  })

  // Opened in the home page's own viewer, which can rotate and replace it.
  function openPhoto (kind, id, title) {
    emit('open-photo', { kind, id, title, has_image: true })
  }

  function amountClass (amount) {
    if (amount > 0) return 'text-success'
    if (amount < 0) return 'text-error'
    return ''
  }
  function nullTime (value) {
    return value?.Valid ? value.Time : null
  }
  function dateTime (value) {
    return value ? new Date(value).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' }) : ''
  }
  function capitalize (value) {
    return value ? value.charAt(0).toUpperCase() + value.slice(1) : ''
  }
  function changeSummary (change) {
    if (change.action === 'add') return `Add ${change.requested_type} ${change.requested_time}`
    if (change.action === 'delete') return `Delete ${change.original_time}`
    if (change.original_time || change.requested_time) {
      return `${change.original_time || '—'} → ${[change.requested_type, change.requested_time].filter(Boolean).join(' ')}`
    }
    return ''
  }

  // One labelled value; left out when there's nothing to show.
  const Fact = defineComponent({
    props: { label: { type: String, required: true }, value: { type: [String, Number], default: '' } },
    setup (factProps) {
      return () => (factProps.value === '' || factProps.value == null)
        ? null
        : h('div', { class: 'fact' }, [
          h('div', { class: 'text-caption text-medium-emphasis' }, factProps.label),
          h('div', { class: 'text-body-2' }, String(factProps.value)),
        ])
    },
  })

  const Payments = defineComponent({
    props: { payments: { type: Array, default: () => [] }, label: { type: String, default: 'Payments' } },
    setup (paymentProps) {
      return () => [
        h('div', { class: 'section-label' }, paymentProps.label),
        paymentProps.payments.length === 0
          ? h('div', { class: 'text-body-2 text-medium-emphasis' }, 'No payments reported.')
          : h('div', { class: 'd-flex flex-wrap ga-2' }, paymentProps.payments.map(p =>
            h('div', { class: 'payment', key: p.id }, [
              h('span', { class: 'text-medium-emphasis' }, p.account_name),
              h('span', { class: 'font-weight-medium ms-2' }, formatAmount(p.amount)),
            ]),
          )),
      ]
    },
  })

  const PhotoThumb = defineComponent({
    props: { src: { type: String, default: '' } },
    emits: ['click'],
    setup (thumbProps, { emit: thumbEmit }) {
      return () => h('button', { class: 'photo-thumb', type: 'button', onClick: () => thumbEmit('click') },
                     thumbProps.src ? [h('img', { src: thumbProps.src, alt: '' })] : [])
    },
  })
</script>

<style scoped>
.detail-head--high {
  background-color: rgba(var(--v-theme-error), 0.24);
}
.detail-head--medium {
  background-color: rgba(255, 109, 0, 0.2);
}
.detail-head--low {
  background-color: rgba(255, 193, 7, 0.16);
}
.min-w-0 {
  min-width: 0;
}
.facts {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px 16px;
}
.facts :deep(.fact) {
  min-width: 0;
  overflow-wrap: anywhere;
}
:deep(.section-label) {
  margin: 16px 0 6px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
}
:deep(.payment) {
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 0.875rem;
  background: rgba(var(--v-theme-on-surface), 0.05);
}
:deep(.photo-thumb) {
  width: 120px;
  height: 120px;
  padding: 0;
  overflow: hidden;
  border-radius: 8px;
  background: rgba(var(--v-theme-on-surface), 0.06);
  cursor: zoom-in;
}
:deep(.photo-thumb img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
