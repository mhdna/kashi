<!--
  The deals a discount list row or rule can give instead of a discount
  (api/deals.go): "Buy 2 get 1 free", "3 for $50", "Buy 3 or more, 20% off
  all". A deal is only its terms here; putting it on products is done on the
  list, by hand or by a rule.
-->
<template>
  <v-dialog v-model="open" max-width="720">
    <v-card>
      <v-card-title class="d-flex align-center ga-2">
        <span>Deals</span>
        <v-spacer />
        <v-btn
          v-if="canEdit"
          prepend-icon="mdi-plus"
          size="small"
          text="New deal"
          variant="tonal"
          @click="openEdit(null)"
        />
      </v-card-title>
      <v-card-text>
        <p class="text-body-2 text-medium-emphasis mb-3">
          Every unit in a cart on the same deal counts together, whatever the product, and the
          cheapest units get the reward. A deal replaces a product's discount.
        </p>
        <div v-if="deals.length === 0" class="text-body-2 text-medium-emphasis">No deals yet.</div>
        <v-table v-else density="compact">
          <thead>
            <tr>
              <th>Name</th>
              <th>Terms</th>
              <th class="text-end">Products</th>
              <th class="text-end">Rules</th>
              <th v-if="canEdit" style="width: 96px" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="deal in deals" :key="deal.id">
              <td>{{ deal.name }}</td>
              <td>{{ dealTerms(deal) }}</td>
              <td class="text-end">{{ deal.variants }}</td>
              <td class="text-end">{{ deal.rules }}</td>
              <td v-if="canEdit" class="text-end text-no-wrap">
                <v-btn
                  density="comfortable"
                  icon="mdi-pencil"
                  size="small"
                  variant="text"
                  @click="openEdit(deal)"
                />
                <v-btn
                  color="error"
                  density="comfortable"
                  :disabled="deal.variants > 0 || deal.rules > 0"
                  icon="mdi-delete"
                  size="small"
                  :title="deal.variants > 0 || deal.rules > 0 ? 'Take it off its lists and rules first' : 'Delete'"
                  variant="text"
                  @click="remove(deal)"
                />
              </td>
            </tr>
          </tbody>
        </v-table>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Close" variant="text" @click="open = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="editDialog" max-width="520">
    <v-card>
      <v-card-title>{{ form.id ? 'Edit deal' : 'New deal' }}</v-card-title>
      <v-card-text>
        <div v-if="!form.id" class="d-flex flex-wrap ga-1 mb-4">
          <v-chip
            v-for="p in PRESETS"
            :key="p.name"
            size="small"
            :text="p.name"
            variant="outlined"
            @click="applyPreset(p)"
          />
        </div>
        <v-text-field
          v-model="form.name"
          density="compact"
          label="Name (printed on the receipt)"
          variant="outlined"
        />
        <v-select
          v-model="form.kind"
          density="compact"
          :items="KINDS"
          label="Kind"
          variant="outlined"
        />
        <div class="d-flex flex-wrap ga-3">
          <v-text-field
            v-model.number="form.buy_qty"
            density="compact"
            :label="form.kind === 'buy_get' ? 'Buy' : (form.kind === 'bundle_price' ? 'Units in the bundle' : 'From units')"
            min="1"
            style="max-width: 150px"
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-if="form.kind === 'buy_get'"
            v-model.number="form.get_qty"
            density="compact"
            label="Get"
            min="1"
            style="max-width: 150px"
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-if="form.kind !== 'bundle_price'"
            v-model.number="form.percent"
            density="compact"
            :hint="form.kind === 'buy_get' ? '100% is free' : ''"
            label="Off"
            max="100"
            min="1"
            persistent-hint
            style="max-width: 150px"
            suffix="%"
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-else
            v-model="form.bundlePrice"
            density="compact"
            label="Bundle price"
            min="0"
            prefix="$"
            style="max-width: 170px"
            type="number"
            variant="outlined"
          />
        </div>
        <v-text-field
          v-if="form.kind === 'buy_get'"
          v-model.number="form.fallback_percent"
          class="mt-3"
          density="compact"
          hint="What units that don't make up a set get: with Buy 2 get 1, one or two items. 0 for nothing."
          label="Otherwise, off"
          max="100"
          min="0"
          persistent-hint
          style="max-width: 220px"
          suffix="%"
          type="number"
          variant="outlined"
        />
        <v-alert class="mt-2" density="compact" type="info" variant="tonal">
          {{ dealTerms(formDeal) }}
        </v-alert>
        <v-alert
          v-if="form.id"
          class="mt-2"
          density="compact"
          type="warning"
          variant="tonal"
        >
          The new terms reach every branch using this deal. Sales already made keep the terms they
          were sold under, for exchanges too.
        </v-alert>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Cancel" variant="text" @click="editDialog = false" />
        <v-btn
          color="primary"
          :loading="saving"
          text="Save"
          variant="flat"
          @click="save"
        />
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { useDeals } from '@/composables/useDeals'
  import { useNotify } from '@/composables/useNotify'

  const open = defineModel({ type: Boolean, default: false })
  defineProps({ canEdit: { type: Boolean, default: false } })
  const notify = useNotify()
  const { deals, fetchDeals, saveDeal, deleteDeal, dealTerms } = useDeals()

  const KINDS = [
    { title: 'Buy X, get Y free or % off', value: 'buy_get' },
    { title: 'X for a fixed price', value: 'bundle_price' },
    { title: 'Buy X or more, % off all', value: 'volume_percent' },
  ]
  // The usual ones, one click away. "Every 3rd item 50% off" is buy 2 get 1 at 50%.
  const PRESETS = [
    { name: 'Buy 2 get 1 free', kind: 'buy_get', buy_qty: 2, get_qty: 1, percent: 100 },
    { name: 'Buy 2 get 1 free, otherwise 30% off', kind: 'buy_get', buy_qty: 2, get_qty: 1, percent: 100, fallback_percent: 30 },
    { name: 'Buy 1 get 1 free', kind: 'buy_get', buy_qty: 1, get_qty: 1, percent: 100 },
    { name: 'Buy 3 get 2 free', kind: 'buy_get', buy_qty: 3, get_qty: 2, percent: 100 },
    { name: '2nd item 50% off', kind: 'buy_get', buy_qty: 1, get_qty: 1, percent: 50 },
    { name: 'Every 3rd item 50% off', kind: 'buy_get', buy_qty: 2, get_qty: 1, percent: 50 },
    { name: '3 for $50', kind: 'bundle_price', buy_qty: 3, bundle_price: 5000 },
    { name: 'Buy 3+, 20% off all', kind: 'volume_percent', buy_qty: 3, percent: 20 },
  ]

  watch(open, value => {
    if (value) fetchDeals(true).catch(error => notify.error(error.message))
  })

  const editDialog = ref(false)
  const saving = ref(false)
  const form = reactive({ id: 0, name: '', kind: 'buy_get', buy_qty: 2, get_qty: 1, percent: 100, fallback_percent: 0, bundlePrice: '' })

  // The form as a deal, for the terms preview and the save.
  const formDeal = computed(() => ({
    id: form.id, name: form.name.trim(), kind: form.kind,
    buy_qty: Number(form.buy_qty) || 0,
    get_qty: form.kind === 'buy_get' ? Number(form.get_qty) || 0 : 0,
    percent: form.kind === 'bundle_price' ? 0 : Number(form.percent) || 0,
    bundle_price: form.kind === 'bundle_price' ? Math.round(Number(form.bundlePrice) * 100) || 0 : 0,
    fallback_percent: form.kind === 'buy_get' ? Number(form.fallback_percent) || 0 : 0,
  }))

  function openEdit (deal) {
    Object.assign(form, deal
      ? { ...deal, bundlePrice: deal.bundle_price ? String(deal.bundle_price / 100) : '' }
      : { id: 0, name: '', kind: 'buy_get', buy_qty: 2, get_qty: 1, percent: 100, fallback_percent: 0, bundlePrice: '' })
    editDialog.value = true
  }

  function applyPreset (p) {
    Object.assign(form, {
      name: p.name, kind: p.kind, buy_qty: p.buy_qty, get_qty: p.get_qty ?? 0,
      percent: p.percent ?? 0, fallback_percent: p.fallback_percent ?? 0, bundlePrice: p.bundle_price ? String(p.bundle_price / 100) : '',
    })
  }

  async function save () {
    saving.value = true
    try {
      await saveDeal(formDeal.value)
      editDialog.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  async function remove (deal) {
    try {
      await deleteDeal(deal.id)
    } catch (error) {
      notify.error(error.message)
    }
  }
</script>
