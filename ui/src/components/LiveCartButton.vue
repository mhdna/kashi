<!--
  App-bar shortcut: see the sale open on each till of a branch, as it is
  rung up, and change it past the branch's own rules: turn every deal off,
  or give a line a price, a discount or no deal (api/live_cart.go). It only
  changes that one sale; the till applies it and locks those lines.
-->
<template>
  <v-btn
    v-if="auth.canEdit('pricing')"
    icon="mdi-cart"
    title="Live cart: the sale open on a till"
    variant="text"
    @click="open = true"
  />

  <v-dialog v-model="open" max-width="980" scrollable>
    <v-card>
      <v-card-title class="d-flex align-center ga-2">
        Live cart
        <v-spacer />
        <v-chip
          :color="onlineOnly ? 'success' : undefined"
          filter
          :model-value="onlineOnly"
          :text="`Online${offlineCount ? ` (${registers.length - offlineCount}/${registers.length})` : ''}`"
          :variant="onlineOnly ? 'tonal' : 'outlined'"
          @click="onlineOnly = !onlineOnly"
        />
        <v-select
          v-model="branchId"
          density="compact"
          hide-details
          :items="branchItems"
          label="Branch"
          style="max-width: 240px"
          variant="outlined"
        />
      </v-card-title>

      <v-card-text>
        <div v-if="loading && registers.length === 0" class="text-center pa-6">
          <v-progress-circular indeterminate />
        </div>
        <v-alert
          v-else-if="registers.length === 0"
          density="compact"
          type="info"
          variant="tonal"
        >
          This branch has no active till.
        </v-alert>
        <v-alert
          v-else-if="shownRegisters.length === 0"
          density="compact"
          type="info"
          variant="tonal"
        >
          None of this branch's {{ registers.length }} tills is reporting right now.
          <a class="text-decoration-underline cursor-pointer" @click="onlineOnly = false">Show them all</a>
        </v-alert>

        <v-card
          v-for="reg in shownRegisters"
          :key="reg.register_id"
          class="mb-4"
          variant="outlined"
        >
          <v-card-item>
            <v-card-title class="d-flex align-center ga-2 text-subtitle-1">
              <v-icon icon="mdi-cash-register" size="20" />
              {{ reg.register_name }}
              <v-chip
                :color="reg.online ? 'success' : 'grey'"
                label
                size="x-small"
                :text="reg.online ? 'online' : 'not reporting'"
              />
              <v-spacer />
              <span v-if="cartOf(reg)?.invoiceCode" class="text-caption text-medium-emphasis">
                {{ cartOf(reg).invoiceCode }}
              </span>
            </v-card-title>
            <v-card-subtitle v-if="hasSale(reg)">
              {{ cartOf(reg).client || 'No client yet' }} · {{ cartOf(reg).salesperson || 'No salesperson yet' }}
            </v-card-subtitle>
          </v-card-item>

          <v-card-text>
            <div v-if="!reg.online" class="text-medium-emphasis">
              This till hasn't reported lately; it may be off or offline.
            </div>
            <div v-else-if="!hasSale(reg)" class="text-medium-emphasis">
              No sale open.
            </div>
            <template v-else>
              <v-alert
                v-if="cartOf(reg).mode !== 'sale'"
                class="mb-3"
                density="compact"
                type="info"
                variant="tonal"
              >
                The till is on {{ cartOf(reg).mode === 'exchange' ? 'an exchange' : 'a settlement' }}: only a plain sale can be changed.
              </v-alert>

              <v-switch
                v-if="editable(reg)"
                v-model="draftOf(reg).allDealsOff"
                color="error"
                density="compact"
                hide-details
                inset
                label="Turn every deal off"
                @update:model-value="touch(reg)"
              />

              <v-table class="live-cart-table" density="compact">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th class="text-end">Qty</th>
                    <th class="text-end">List</th>
                    <th class="text-end">On the till</th>
                    <template v-if="editable(reg)">
                      <th style="width: 120px">Price</th>
                      <th style="width: 100px">Discount</th>
                      <th class="text-center">No deal</th>
                    </template>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(line, i) in cartOf(reg).lines" :key="`${line.barcode}:${line.saleOption}:${i}`">
                    <td>
                      <div class="d-flex align-center ga-1">
                        <v-icon
                          v-if="line.kashi"
                          color="primary"
                          icon="mdi-lock-outline"
                          size="14"
                          title="Set from kashi"
                        />
                        <span class="font-weight-medium">{{ line.name }}</span>
                      </div>
                      <div class="text-caption text-medium-emphasis">
                        {{ line.code }}<span v-if="line.variant"> · {{ line.variant }}</span><span v-if="line.saleOption"> · {{ line.saleOption }}</span>
                      </div>
                    </td>
                    <td class="text-end">{{ line.qty }}</td>
                    <td class="text-end text-no-wrap">
                      {{ money(line.listPriceCents) }}
                      <span v-if="line.listDiscount" class="text-error"> −{{ line.listDiscount }}%</span>
                    </td>
                    <td class="text-end text-no-wrap">
                      {{ money(line.priceCents) }}
                      <span v-if="line.discount" class="text-error"> −{{ line.discount }}%</span>
                      <v-chip
                        v-if="line.deal"
                        class="ms-1"
                        color="success"
                        label
                        size="x-small"
                        :text="line.deal"
                      />
                      <v-chip
                        v-else-if="line.dealOff"
                        class="ms-1 text-decoration-line-through"
                        label
                        size="x-small"
                        :text="line.dealOff"
                        variant="outlined"
                      />
                    </td>
                    <template v-if="editable(reg)">
                      <template v-if="line.unknown || !line.barcode || line.qty <= 0">
                        <td class="text-caption text-medium-emphasis" colspan="3">
                          {{ line.unknown ? 'Unknown items can’t be set one by one' : '—' }}
                        </td>
                      </template>
                      <template v-else>
                        <td>
                          <v-text-field
                            v-model="lineDraft(reg, line).price"
                            density="compact"
                            hide-details
                            min="0.01"
                            :placeholder="(line.listPriceCents / 100).toFixed(2)"
                            prefix="$"
                            step="0.01"
                            type="number"
                            variant="outlined"
                            @update:model-value="touch(reg)"
                          />
                        </td>
                        <td>
                          <v-text-field
                            v-model="lineDraft(reg, line).discount"
                            density="compact"
                            hide-details
                            max="100"
                            min="0"
                            :placeholder="String(line.listDiscount || 0)"
                            suffix="%"
                            type="number"
                            variant="outlined"
                            @update:model-value="touch(reg)"
                          />
                        </td>
                        <td class="text-center">
                          <v-checkbox-btn
                            v-model="lineDraft(reg, line).noDeal"
                            class="d-inline-flex"
                            :disabled="draftOf(reg).allDealsOff"
                            @update:model-value="touch(reg)"
                          />
                        </td>
                      </template>
                    </template>
                  </tr>
                </tbody>
              </v-table>

              <div class="d-flex align-center mt-3 ga-2 flex-wrap">
                <div>
                  <div v-if="cartOf(reg).invoiceDiscountCents" class="text-caption">
                    {{ cartOf(reg).invoiceDiscountName || 'Invoice discount' }}: −{{ money(cartOf(reg).invoiceDiscountCents) }}
                  </div>
                  <div class="text-h6">
                    Total {{ cartOf(reg).totalCents == null ? '—' : money(cartOf(reg).totalCents) }}
                  </div>
                </div>
                <v-spacer />
                <span v-if="statusText(reg)" class="text-caption" :class="statusClass(reg)">
                  {{ statusText(reg) }}
                </span>
                <template v-if="editable(reg)">
                  <v-btn
                    :disabled="!hasOverrides(reg) && !draftOf(reg).dirty"
                    :loading="saving === reg.register_id"
                    text="Undo all"
                    variant="text"
                    @click="undoAll(reg)"
                  />
                  <v-btn
                    color="primary"
                    :disabled="!draftOf(reg).dirty"
                    :loading="saving === reg.register_id"
                    text="Apply to the till"
                    variant="flat"
                    @click="apply(reg)"
                  />
                </template>
              </div>
            </template>
          </v-card-text>
        </v-card>
      </v-card-text>

      <v-card-actions>
        <span class="text-caption text-medium-emphasis ms-2">
          Changes go to this one sale, past the branch's price and deal settings.
        </span>
        <v-spacer />
        <v-btn text="Close" variant="text" @click="open = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup>
  import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useBranches } from '@/composables/useBranches'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'

  const auth = useAuthStore()
  const notify = useNotify()
  const { branches, fetchBranches } = useBranches()

  const LAST_BRANCH_KEY = 'liveCart.branchId'
  const POLL_MS = 2000

  const open = ref(false)
  const branchId = ref(null)
  const registers = ref([])
  // Only the tills reporting now, unless the filter is turned off; the
  // choice is remembered.
  const ONLINE_ONLY_KEY = 'liveCart.onlineOnly'
  const onlineOnly = ref(true)
  try {
    onlineOnly.value = localStorage.getItem(ONLINE_ONLY_KEY) !== '0'
  } catch {}
  watch(onlineOnly, value => {
    try {
      localStorage.setItem(ONLINE_ONLY_KEY, value ? '1' : '0')
    } catch {}
  })
  const offlineCount = computed(() => registers.value.filter(r => !r.online).length)
  const shownRegisters = computed(() => onlineOnly.value ? registers.value.filter(r => r.online) : registers.value)
  const loading = ref(false)
  const saving = ref(null)
  // What is typed per register, for the sale it was typed on:
  // { saleRef, allDealsOff, lines: { [barcode]: { price, discount, noDeal } }, dirty }
  const drafts = reactive({})
  let timer = null

  const branchItems = computed(() => branches.value
    .filter(b => b.is_active)
    .map(b => ({ title: b.name, value: b.id })))

  function money (cents) {
    return `$${(Number(cents || 0) / 100).toFixed(2)}`
  }

  const cartOf = reg => reg.cart
  const hasSale = reg => Boolean(reg.online && reg.cart?.lines?.length)
  const editable = reg => hasSale(reg) && reg.cart.mode === 'sale'
  const hasOverrides = reg => Boolean(reg.cart?.overrides)

  // The draft follows the till's own overrides until someone types, and
  // starts over when the till moves to another sale.
  function syncDraft (reg) {
    const cart = reg.cart
    if (!cart) return
    const d = drafts[reg.register_id]
    if (d && d.saleRef === cart.saleRef && (d.dirty || reg.last_command?.status === 'pending')) return
    const o = cart.overrides || {}
    const lines = {}
    for (const l of o.lines || []) {
      lines[l.barcode] = {
        price: l.price_cents == null ? '' : (l.price_cents / 100).toFixed(2),
        discount: l.discount == null ? '' : String(l.discount),
        noDeal: Boolean(l.no_deal),
      }
    }
    drafts[reg.register_id] = { saleRef: cart.saleRef, allDealsOff: Boolean(o.all_deals_off), lines, dirty: false }
  }

  function draftOf (reg) {
    if (!drafts[reg.register_id]) syncDraft(reg)
    return drafts[reg.register_id]
  }

  function lineDraft (reg, line) {
    const d = draftOf(reg)
    d.lines[line.barcode] ??= { price: '', discount: '', noDeal: false }
    return d.lines[line.barcode]
  }

  function touch (reg) {
    draftOf(reg).dirty = true
  }

  async function load () {
    if (!branchId.value) return
    loading.value = true
    try {
      const res = await authFetch(`${API_BASE}/live_carts/${branchId.value}`)
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || `Request failed with status ${res.status}`)
      registers.value = data.registers ?? []
      for (const reg of registers.value) syncDraft(reg)
    } catch (error) {
      stop()
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  function start () {
    stop()
    load()
    timer = setInterval(load, POLL_MS)
  }

  function stop () {
    if (timer) clearInterval(timer)
    timer = null
  }

  function buildLines (reg) {
    const d = draftOf(reg)
    const out = []
    const seen = new Set()
    for (const line of reg.cart.lines) {
      if (line.unknown || !line.barcode || line.qty <= 0 || seen.has(line.barcode)) continue
      seen.add(line.barcode)
      const l = d.lines[line.barcode]
      if (!l) continue
      const entry = { barcode: line.barcode }
      if (String(l.price).trim() !== '') {
        const cents = Math.round(Number(l.price) * 100)
        if (!(cents > 0)) throw new Error(`${line.name}: enter a price above 0.`)
        entry.price_cents = cents
      }
      if (String(l.discount).trim() !== '') {
        const discount = Number(l.discount)
        if (!Number.isInteger(discount) || discount < 0 || discount > 100) {
          throw new Error(`${line.name}: enter a discount from 0 to 100.`)
        }
        entry.discount = discount
      }
      if (l.noDeal && !d.allDealsOff) entry.no_deal = true
      if (entry.price_cents != null || entry.discount != null || entry.no_deal) out.push(entry)
    }
    return out
  }

  async function send (reg, body) {
    saving.value = reg.register_id
    try {
      const res = await authFetch(`${API_BASE}/live_carts/${branchId.value}/override`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ register_id: reg.register_id, sale_ref: reg.cart.saleRef, ...body }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || `Request failed with status ${res.status}`)
      draftOf(reg).dirty = false
      notify.success(`Sent to ${reg.register_name}.`)
      await load()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = null
    }
  }

  function apply (reg) {
    let lines
    try {
      lines = buildLines(reg)
    } catch (error) {
      notify.error(error.message)
      return
    }
    send(reg, { all_deals_off: draftOf(reg).allDealsOff, lines })
  }

  function undoAll (reg) {
    drafts[reg.register_id] = { saleRef: reg.cart.saleRef, allDealsOff: false, lines: {}, dirty: false }
    send(reg, { all_deals_off: false, lines: [] })
  }

  function statusText (reg) {
    const c = reg.last_command
    if (!c) return ''
    if (c.status === 'pending') return 'Waiting for the till…'
    if (c.status === 'failed') return `The till refused: ${c.error || 'unknown error'}`
    return ''
  }

  function statusClass (reg) {
    return reg.last_command?.status === 'failed' ? 'text-error' : 'text-medium-emphasis'
  }

  watch(open, async value => {
    if (!value) {
      stop()
      return
    }
    registers.value = []
    try {
      await fetchBranches()
    } catch (error) {
      notify.error(error.message)
    }
    let last = null
    try {
      last = Number(localStorage.getItem(LAST_BRANCH_KEY)) || null
    } catch {}
    const next = branchItems.value.some(b => b.value === last) ? last : (branchItems.value[0]?.value ?? null)
    if (next === branchId.value) start()
    else branchId.value = next
  })

  watch(branchId, value => {
    try {
      if (value) localStorage.setItem(LAST_BRANCH_KEY, String(value))
    } catch {}
    registers.value = []
    for (const key of Object.keys(drafts)) delete drafts[key]
    if (open.value) start()
  })

  onBeforeUnmount(stop)
</script>

<style scoped>
.live-cart-table td {
  vertical-align: middle;
}
</style>
