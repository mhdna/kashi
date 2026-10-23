<!--
  A discount list's rules (api/discount_rules.go): "Lady + Turkey: 30%",
  "codes starting CA: winter 20%, summer 50%". The first rule a product
  matches decides its discount; saving writes the rules out as the list's
  rows below, and kashi keeps them in step as products change. A row set
  by hand beats the rules.
-->
<template>
  <div class="px-4 pt-3 pb-1">
    <div class="d-flex align-center ga-2 mb-2">
      <span class="text-subtitle-1 font-weight-medium">Rules</span>
      <span class="text-body-2 text-medium-emphasis">first match wins, top to bottom</span>
      <v-spacer />
      <template v-if="canEdit">
        <v-chip
          v-if="dirty"
          color="warning"
          size="small"
          text="Not saved"
          variant="tonal"
        />
        <v-btn
          prepend-icon="mdi-plus"
          size="small"
          text="Add rule"
          variant="tonal"
          @click="openRule(-1)"
        />
        <v-btn
          color="primary"
          :disabled="!dirty"
          :loading="saving"
          prepend-icon="mdi-content-save"
          size="small"
          text="Save rules"
          variant="flat"
          @click="save"
        />
      </template>
    </div>

    <div v-if="loading" class="py-2"><v-progress-linear indeterminate /></div>
    <div v-else-if="rules.length === 0" class="text-body-2 text-medium-emphasis pb-2">
      No rules: this list has only the products added below.
    </div>
    <v-table v-else class="rules-table" density="compact">
      <thead>
        <tr>
          <th style="width: 32px">#</th>
          <th>Rule</th>
          <th>Matches</th>
          <th class="text-end">Discount</th>
          <th class="text-end">Products</th>
          <th v-if="canEdit" style="width: 132px" />
        </tr>
      </thead>
      <tbody>
        <tr v-for="(rule, i) in rules" :key="i">
          <td class="text-medium-emphasis">{{ i + 1 }}</td>
          <td>{{ rule.name || '—' }}</td>
          <td>
            <div class="d-flex flex-wrap ga-1 py-1">
              <v-chip
                v-for="c in conditionChips(rule)"
                :key="c"
                size="x-small"
                :text="c"
                variant="tonal"
              />
            </div>
          </td>
          <td class="text-end text-no-wrap">{{ discountText(rule) }}</td>
          <td class="text-end text-medium-emphasis">{{ dirty ? '…' : rule.variants }}</td>
          <td v-if="canEdit" class="text-end text-no-wrap">
            <v-btn
              density="comfortable"
              :disabled="i === 0"
              icon="mdi-arrow-up"
              size="small"
              variant="text"
              @click="move(i, -1)"
            />
            <v-btn
              density="comfortable"
              :disabled="i === rules.length - 1"
              icon="mdi-arrow-down"
              size="small"
              variant="text"
              @click="move(i, 1)"
            />
            <v-btn
              density="comfortable"
              icon="mdi-pencil"
              size="small"
              variant="text"
              @click="openRule(i)"
            />
            <v-btn
              color="error"
              density="comfortable"
              icon="mdi-delete"
              size="small"
              variant="text"
              @click="remove(i)"
            />
          </td>
        </tr>
      </tbody>
    </v-table>

    <v-dialog v-model="ruleDialog" max-width="600">
      <v-card>
        <v-card-title>{{ editingIndex < 0 ? 'Add rule' : 'Edit rule' }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="form.name"
            density="compact"
            label="Name"
            placeholder="e.g. Lady Turkey"
            variant="outlined"
          />
          <p class="text-body-2 text-medium-emphasis mb-3">
            A product matches when it fits every box that has something in it; within a box,
            any one value is enough.
          </p>
          <div class="rule-grid">
            <v-combobox
              v-model="form.code_prefixes"
              chips
              closable-chips
              density="compact"
              hide-details
              label="Code starts with"
              multiple
              variant="outlined"
            />
            <v-combobox
              v-model="form.except_code_prefixes"
              chips
              closable-chips
              density="compact"
              hide-details
              label="But not codes starting with"
              multiple
              variant="outlined"
            />
            <v-combobox
              v-for="f in attributeFields"
              :key="f.key"
              v-model="form[f.key]"
              chips
              closable-chips
              density="compact"
              hide-details
              :items="valuesFor(f.attribute)"
              :label="f.label"
              multiple
              variant="outlined"
            />
          </div>

          <div class="d-flex flex-wrap ga-3 mt-4">
            <v-text-field
              v-model="form.discount"
              density="compact"
              hint="0 keeps these products at full price, and out of the rules below"
              label="Discount"
              max="100"
              min="0"
              persistent-hint
              style="max-width: 220px"
              suffix="%"
              type="number"
              variant="outlined"
            />
            <v-text-field
              v-for="o in saleOptions"
              :key="o.id"
              v-model="form.options[o.id]"
              density="compact"
              hint="when the till sells it as this"
              :label="`Sold as ${o.name}`"
              max="100"
              min="0"
              persistent-hint
              placeholder="same"
              style="max-width: 170px"
              suffix="%"
              type="number"
              variant="outlined"
            />
          </div>
          <v-alert
            v-if="Object.values(form.options).some(v => v !== '' && v != null)"
            class="mt-3"
            density="compact"
            type="info"
            variant="tonal"
          >
            Option discounts apply only to products that ask at the till (“Ask at the till”).
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="ruleDialog = false" />
          <v-btn color="primary" text="Done" variant="flat" @click="applyRule" />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { reactive, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'

  const props = defineProps({
    listId: { type: Number, required: true },
    canEdit: { type: Boolean, default: false },
    saleOptions: { type: Array, default: () => [] },
  })
  const emit = defineEmits(['saved'])
  const notify = useNotify()

  // The product fields a rule can look at, by their attribute name.
  const attributeFields = [
    { key: 'kinds', label: 'Kind', attribute: 'kind' },
    { key: 'categories', label: 'Category', attribute: 'category' },
    { key: 'sub_categories', label: 'Sub-category', attribute: 'sub-category' },
    { key: 'origins', label: 'Origin', attribute: 'origin' },
    { key: 'brands', label: 'Brand', attribute: 'brand' },
  ]
  const CONDITIONS = [
    { key: 'code_prefixes', label: 'Code', fmt: v => `${v}…` },
    { key: 'except_code_prefixes', label: 'Not', fmt: v => `${v}…` },
    ...attributeFields.map(f => ({ key: f.key, label: f.label, fmt: v => v })),
  ]

  const rules = ref([])
  const loading = ref(false)
  const saving = ref(false)
  const dirty = ref(false)
  const attributeValues = ref([])

  async function request (method, body) {
    const res = await authFetch(`${API_BASE}/discount_lists/${props.listId}/rules`, {
      method,
      headers: body ? { 'Content-Type': 'application/json' } : undefined,
      body: body ? JSON.stringify(body) : undefined,
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(data.error || `Request failed with status ${res.status}`)
    return data.rules ?? []
  }

  async function load () {
    loading.value = true
    try {
      rules.value = await request('GET')
      dirty.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  async function loadAttributeValues () {
    if (attributeValues.value.length > 0) return
    try {
      const res = await authFetch(`${API_BASE}/attribute_values`)
      const data = await res.json().catch(() => ({}))
      if (res.ok) attributeValues.value = data.values ?? []
    } catch {}
  }

  function valuesFor (attribute) {
    return attributeValues.value
      .filter(v => v.attribute_name === attribute && v.value.trim())
      .map(v => v.value)
  }

  function conditionChips (rule) {
    return CONDITIONS.flatMap(c => (rule[c.key] ?? []).length > 0
      ? [`${c.label}: ${rule[c.key].map(v => c.fmt(v)).join(', ')}`]
      : [])
  }

  function discountText (rule) {
    const parts = (rule.options ?? []).map(o => `${o.sale_option} ${o.discount}%`)
    const plain = rule.discount > 0 ? `${rule.discount}%` : 'none'
    return parts.length > 0 ? `${plain} · ${parts.join(' · ')}` : plain
  }

  function move (i, step) {
    const next = [...rules.value]
    ;[next[i], next[i + step]] = [next[i + step], next[i]]
    rules.value = next
    dirty.value = true
  }

  function remove (i) {
    rules.value = rules.value.filter((_, j) => j !== i)
    dirty.value = true
  }

  const ruleDialog = ref(false)
  const editingIndex = ref(-1)
  const form = reactive({ name: '', discount: '0', options: {} })

  function openRule (i) {
    loadAttributeValues()
    editingIndex.value = i
    const rule = i >= 0 ? rules.value[i] : {}
    form.name = rule.name ?? ''
    for (const c of CONDITIONS) form[c.key] = [...(rule[c.key] ?? [])]
    form.discount = String(rule.discount ?? 0)
    form.options = Object.fromEntries(props.saleOptions.map(o => {
      const found = (rule.options ?? []).find(x => x.sale_option_id === o.id)
      return [o.id, found ? String(found.discount) : '']
    }))
    ruleDialog.value = true
  }

  function percent (value) {
    const n = Number(value)
    return Number.isInteger(n) && n >= 0 && n <= 100 ? n : null
  }

  function applyRule () {
    const discount = percent(form.discount)
    if (discount === null) {
      notify.error('Enter a discount from 0 to 100.')
      return
    }
    const options = []
    for (const o of props.saleOptions) {
      const raw = form.options[o.id]
      if (raw === '' || raw == null) continue
      const d = percent(raw)
      if (d === null) {
        notify.error(`Enter a ${o.name} discount from 0 to 100, or leave it empty.`)
        return
      }
      options.push({ sale_option_id: o.id, sale_option: o.name, discount: d })
    }
    const rule = { name: form.name.trim(), discount, options, variants: 0 }
    for (const c of CONDITIONS) rule[c.key] = form[c.key].map(v => String(v).trim()).filter(Boolean)
    if (attributeFields.every(f => rule[f.key].length === 0) && rule.code_prefixes.length === 0) {
      notify.error('Give the rule at least one condition: a code start, kind, category, origin or brand.')
      return
    }
    const next = [...rules.value]
    if (editingIndex.value < 0) next.push(rule)
    else next[editingIndex.value] = rule
    rules.value = next
    dirty.value = true
    ruleDialog.value = false
  }

  async function save () {
    saving.value = true
    try {
      rules.value = await request('PUT', { rules: rules.value })
      dirty.value = false
      notify.success('Rules saved: the list below now follows them.')
      emit('saved')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  watch(() => props.listId, load, { immediate: true })
</script>

<style scoped>
.rule-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.rules-table td {
  vertical-align: middle;
}
</style>
