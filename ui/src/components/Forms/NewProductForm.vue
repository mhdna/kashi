<template>
  <!-- Fixed height so the dialog doesn't jump as fields fill in; only the
       body scrolls, the title and actions stay put (the parent v-dialog is
       `scrollable`). -->
  <v-card class="new-product-card" tag="form" @submit.prevent="submit">
    <v-card-title class="d-flex align-center ga-2">
      <span class="text-h6">New Product</span>
      <v-spacer />
      <v-btn
        v-if="settingsStore.productAutofillEnabled"
        color="primary"
        :disabled="!code.value.value && !name.value.value"
        prepend-icon="mdi-auto-fix"
        size="small"
        variant="tonal"
        @click="autofill"
      >
        Auto-fill
        <v-tooltip activator="parent" location="bottom">
          Fill attributes and price from the code and name
        </v-tooltip>
      </v-btn>
      <v-btn icon="mdi-close" size="small" variant="text" @click="$emit('close')" />
    </v-card-title>
    <v-divider />

    <v-card-text class="py-4">
      <v-row density="compact">
        <v-col cols="12" sm="6">
          <v-text-field
            v-model="code.value.value"
            density="compact"
            :error-messages="code.errorMessage.value"
            label="Code"
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            v-model="name.value.value"
            density="compact"
            :error-messages="name.errorMessage.value"
            label="Name"
            variant="outlined"
          />
        </v-col>
      </v-row>

      <v-textarea
        v-model="description.value.value"
        auto-grow
        density="compact"
        :error-messages="description.errorMessage.value"
        label="Description (optional)"
        rows="2"
        variant="outlined"
      />

      <div class="text-overline text-medium-emphasis mt-1 mb-1">Attributes</div>
      <v-row density="compact">
        <v-col v-for="field in attributeFields" :key="field.key" cols="12" sm="6">
          <v-combobox
            v-model="field.model.value.value"
            clearable
            density="compact"
            hide-details="auto"
            :items="optionsFor(field.attribute)"
            :label="field.label"
            variant="outlined"
          />
        </v-col>
      </v-row>

      <div class="text-overline text-medium-emphasis mt-3 mb-1">Variants</div>
      <v-row density="compact">
        <v-col cols="12" sm="6">
          <v-autocomplete
            v-model="colorIds.value.value"
            chips
            clearable
            closable-chips
            density="compact"
            hide-details="auto"
            item-title="name"
            item-value="id"
            :items="colors"
            label="Colors"
            :menu-props="{ maxHeight: 320 }"
            multiple
            variant="outlined"
          >
            <!-- Vuetify 4: #item slot's `item` is the raw record, not an InternalItem. -->
            <template #item="{ props: itemProps, item }">
              <v-list-item v-bind="itemProps" :title="item.name">
                <template #prepend>
                  <div
                    class="rounded me-2 swatch"
                    :style="{ backgroundColor: item.hex_value || 'transparent' }"
                  />
                </template>
              </v-list-item>
            </template>
          </v-autocomplete>
        </v-col>
        <v-col cols="12" sm="6">
          <v-autocomplete
            v-model="sizeIds.value.value"
            chips
            clearable
            closable-chips
            density="compact"
            hide-details="auto"
            :item-title="s => `${s.type}: ${s.name}`"
            item-value="id"
            :items="sizes"
            label="Sizes"
            :menu-props="{ maxHeight: 320 }"
            multiple
            variant="outlined"
          />
        </v-col>
      </v-row>
      <div class="text-caption text-medium-emphasis mt-1">
        {{ variantCount }} variant{{ variantCount === 1 ? '' : 's' }} will be created,
        each with its own barcode.
      </div>

      <v-text-field
        v-model.number="barcode.value.value"
        class="mt-3"
        clearable
        density="compact"
        :disabled="variantCount > 1"
        :error-messages="barcode.errorMessage.value"
        :hint="variantCount > 1 ? 'Barcodes are auto-generated when creating multiple variants' : 'Leave empty to auto-generate'"
        label="Barcode (optional)"
        persistent-hint
        type="number"
        variant="outlined"
      />

      <div class="text-overline text-medium-emphasis mt-3 mb-1">Price</div>
      <v-row density="compact">
        <v-col cols="12" sm="6">
          <v-select
            v-model="priceListId.value.value"
            density="compact"
            hide-details="auto"
            item-title="name"
            item-value="id"
            :items="priceLists"
            label="Price list"
            variant="outlined"
          >
            <template #item="{ props: itemProps, item }">
              <v-list-item v-bind="itemProps">
                <template #append>
                  <v-chip v-if="item.is_default" color="primary" size="x-small" text="Default" />
                  <v-chip v-else-if="!item.is_active" size="x-small" text="Inactive" />
                </template>
              </v-list-item>
            </template>
          </v-select>
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            v-model="price.value.value"
            clearable
            density="compact"
            :disabled="!priceListId.value.value"
            :error-messages="price.errorMessage.value"
            :hint="variantCount > 1 ? `Applied to all ${variantCount} variants` : 'Leave empty to price it later'"
            inputmode="decimal"
            label="Price (optional)"
            persistent-hint
            variant="outlined"
          />
        </v-col>
      </v-row>

    </v-card-text>

    <v-divider />
    <v-card-actions class="px-4 py-3">
      <v-spacer />
      <v-btn text="Clear" variant="text" @click="resetForm" />
      <v-btn
        color="primary"
        :loading="submitting"
        text="Add Product"
        type="submit"
        variant="flat"
      />
    </v-card-actions>
  </v-card>
</template>

<script setup>
  import { useField, useForm } from 'vee-validate'
  import { computed, ref } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useAttributes } from '@/composables/useAttributes'
  import { useColorsAndSizes } from '@/composables/useColorsAndSizes'
  import { useNotify } from '@/composables/useNotify'
  import { usePriceLists } from '@/composables/usePriceLists'
  import { API_BASE } from '@/config'
  import { useSettingsStore } from '@/stores/settings'
  import { applyAutofillRules, AUTOFILL_FIELDS } from '@/utils/productAutofill'

  const emit = defineEmits(['created', 'close'])
  const settingsStore = useSettingsStore()

  const { colors, sizes, fetchColors, fetchSizes } = useColorsAndSizes()
  const { valuesByType, fetchAll: fetchAttributes } = useAttributes()
  fetchColors()
  fetchSizes()
  fetchAttributes()

  const { list: priceLists, fetchLists: fetchPriceLists } = usePriceLists()
  const defaultPriceListId = () => priceLists.value.find(l => l.is_default)?.id ?? null

  const { handleSubmit, handleReset } = useForm({
    validationSchema: {
      code (value) {
        return value?.length >= 1 ? true : 'Code is required.'
      },
      name (value) {
        return value?.length >= 2 ? true : 'Name needs to be at least 2 characters.'
      },
      description () {
        return true
      },
      barcode () {
        return true
      },
      colorIds () {
        return true
      },
      sizeIds () {
        return true
      },
      priceListId () {
        return true
      },
      price (value) {
        if (value === null || value === undefined || String(value).trim() === '') return true
        const n = Number(value)
        return Number.isFinite(n) && n >= 0 ? true : 'Price must be a positive number.'
      },
    },
  })

  const code = useField('code')
  const name = useField('name')
  const description = useField('description')
  const barcode = useField('barcode')
  const colorIds = useField('colorIds', undefined, { initialValue: [] })
  const sizeIds = useField('sizeIds', undefined, { initialValue: [] })
  const priceListId = useField('priceListId')
  const price = useField('price')

  // Preselect the default list once the lists are in, and again after each
  // reset so the next product starts from it too.
  fetchPriceLists().then(() => {
    if (!priceListId.value.value) priceListId.value.value = defaultPriceListId()
  })
  function resetForm () {
    handleReset()
    priceListId.value.value = defaultPriceListId()
  }

  // One variant per colour x size; at least one either way.
  const variantCount = computed(() => {
    const c = colorIds.value.value?.length || 0
    const s = sizeIds.value.value?.length || 0
    return Math.max(1, c || 1) * Math.max(1, s || 1)
  })

  // key -> payload key sent to the API; attribute -> seeded attribute type name.
  const attributeFields = [
    { key: 'category', attribute: 'category', label: 'Category', model: useField('category') },
    { key: 'subcategory', attribute: 'sub-category', label: 'Sub-Category', model: useField('subcategory') },
    { key: 'brand', attribute: 'brand', label: 'Brand', model: useField('brand') },
    { key: 'kind', attribute: 'kind', label: 'Kind', model: useField('kind') },
    { key: 'type', attribute: 'type', label: 'Type', model: useField('type') },
    { key: 'unit', attribute: 'unit', label: 'Unit', model: useField('unit') },
    { key: 'year', attribute: 'year', label: 'Year', model: useField('year') },
    { key: 'season', attribute: 'season', label: 'Season', model: useField('season') },
    { key: 'origin', attribute: 'origin', label: 'Origin', model: useField('origin') },
  ]

  function optionsFor (attribute) {
    return (valuesByType.value[attribute] ?? []).map(v => v.value)
  }

  const notify = useNotify()

  // Smart auto-fill: runs the Auto-fill Rules over the code and name and
  // overwrites only the fields a rule matched. A filled attribute reuses an
  // existing value's spelling when it differs only in case, so "lebanon"
  // doesn't create a second Lebanon.
  function autofill () {
    const { values } = applyAutofillRules(settingsStore.productAutofillRules, {
      code: code.value.value,
      name: name.value.value,
    }, settingsStore.productAutofillCodeLayout)
    const filled = []
    for (const [key, value] of Object.entries(values)) {
      if (key === 'price') {
        price.value.value = value
      } else if (key === 'description') {
        description.value.value = value
      } else {
        const field = attributeFields.find(f => f.key === key)
        if (!field) continue
        const existing = optionsFor(field.attribute).find(o => o.toLowerCase() === value.toLowerCase())
        field.model.value.value = existing ?? value
      }
      filled.push(AUTOFILL_FIELDS.find(f => f.key === key)?.label ?? key)
    }
    if (filled.length === 0) {
      notify.info('No auto-fill rule matched this code or name.')
    } else {
      notify.success(`Auto-filled ${filled.join(', ')}.`)
    }
  }

  const submitting = ref(false)

  const submit = handleSubmit(async values => {
    submitting.value = true
    try {
      const attributes = {}
      for (const field of attributeFields) {
        attributes[field.key] = values[field.key] ?? ''
      }

      // Entered in currency units, stored in cents (as on the price lists page).
      const hasPrice = values.price !== null && values.price !== undefined && String(values.price).trim() !== ''
      const priceCents = hasPrice && values.priceListId
        ? Math.round(Number.parseFloat(values.price) * 100)
        : null

      const res = await authFetch(`${API_BASE}/products`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          code: values.code,
          name: values.name,
          description: values.description ?? '',
          attributes,
          barcode: variantCount.value > 1 ? null : (values.barcode || null),
          color_ids: values.colorIds ?? [],
          size_ids: values.sizeIds ?? [],
          price: priceCents,
          price_list_id: priceCents === null ? null : values.priceListId,
        }),
      })

      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || `Request failed with status ${res.status}`)
      }

      resetForm()
      emit('created')
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  })
</script>

<style scoped>
.new-product-card {
  height: min(780px, calc(100dvh - 48px));
}
.swatch {
  width: 16px;
  height: 16px;
  border: 1px solid rgba(128, 128, 128, 0.6);
}
</style>
