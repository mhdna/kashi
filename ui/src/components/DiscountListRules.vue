<!--
  A discount list's rules (api/discount_rules.go): "Lady + Turkey: 30%",
  "codes starting CA: winter 20%, summer 50%", "everything: Buy 2 get 1
  free" (a deal, 000201, in place of a discount). The first rule a product
  matches decides its discount; saving writes the rules out as the list's
  rows below, and kashi keeps them in step as products change. A row set
  by hand beats the rules.

  Rules sit in named groups (000203) -- "Exclusions", "Winter sale" -- and
  each group and each rule has an on/off switch; one that is off decides
  nothing. Switches, like every other change here, take effect on Save.
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
        <v-switch
          v-if="ruleCount > 0"
          v-model="allOn"
          class="flex-grow-0"
          color="primary"
          density="compact"
          hide-details
          label="All on"
        />
        <v-btn
          prepend-icon="mdi-plus-box-outline"
          size="small"
          text="Add group"
          variant="tonal"
          @click="openGroup(-1)"
        />
        <v-btn
          prepend-icon="mdi-plus"
          size="small"
          text="Add rule"
          variant="tonal"
          @click="openRule(groups.length - 1, -1)"
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
    <div v-else-if="groups.length === 0" class="text-body-2 text-medium-emphasis pb-2">
      No rules: this list has only the products added below.
    </div>
    <v-table v-else class="rules-table" density="compact">
      <thead>
        <tr>
          <th style="width: 56px" />
          <th style="width: 32px">#</th>
          <th>Rule</th>
          <th>Matches</th>
          <th class="text-end">Discount</th>
          <th class="text-end">Products</th>
          <th v-if="canEdit" style="width: 172px" />
        </tr>
      </thead>
      <tbody v-for="(group, g) in groups" :key="`group-${g}`">
        <tr class="group-row">
          <td>
            <v-switch
              v-model="group.enabled"
              color="primary"
              density="compact"
              :disabled="!canEdit"
              hide-details
              @update:model-value="dirty = true"
            />
          </td>
          <td colspan="5">
            <span class="font-weight-medium">{{ group.name || 'Untitled group' }}</span>
            <span class="text-body-2 text-medium-emphasis">
              · {{ group.rules.length }} {{ group.rules.length === 1 ? 'rule' : 'rules' }}
              <template v-if="!group.enabled"> · off</template>
            </span>
          </td>
          <td v-if="canEdit" class="text-end text-no-wrap">
            <v-btn
              density="comfortable"
              :disabled="g === 0"
              icon="mdi-arrow-up"
              size="small"
              title="Move the group up"
              variant="text"
              @click="moveGroup(g, -1)"
            />
            <v-btn
              density="comfortable"
              :disabled="g === groups.length - 1"
              icon="mdi-arrow-down"
              size="small"
              title="Move the group down"
              variant="text"
              @click="moveGroup(g, 1)"
            />
            <v-btn
              density="comfortable"
              icon="mdi-plus"
              size="small"
              title="Add a rule to this group"
              variant="text"
              @click="openRule(g, -1)"
            />
            <v-btn
              density="comfortable"
              icon="mdi-pencil"
              size="small"
              title="Rename the group"
              variant="text"
              @click="openGroup(g)"
            />
            <v-btn
              color="error"
              density="comfortable"
              icon="mdi-delete"
              size="small"
              title="Delete the group and its rules"
              variant="text"
              @click="removeGroup(g)"
            />
          </td>
        </tr>
        <tr
          v-for="(rule, r) in group.rules"
          :key="`rule-${g}-${r}`"
          :class="{ 'rule-off': !group.enabled || !rule.enabled }"
        >
          <td>
            <v-switch
              v-model="rule.enabled"
              color="primary"
              density="compact"
              :disabled="!canEdit"
              hide-details
              @update:model-value="dirty = true"
            />
          </td>
          <td class="text-medium-emphasis">{{ ruleNumber(g, r) }}</td>
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
              icon="mdi-target"
              size="small"
              title="Only this rule: turn every other rule off"
              variant="text"
              @click="onlyThis(g, r)"
            />
            <v-btn
              density="comfortable"
              :disabled="r === 0"
              icon="mdi-arrow-up"
              size="small"
              variant="text"
              @click="moveRule(g, r, -1)"
            />
            <v-btn
              density="comfortable"
              :disabled="r === group.rules.length - 1"
              icon="mdi-arrow-down"
              size="small"
              variant="text"
              @click="moveRule(g, r, 1)"
            />
            <v-btn
              density="comfortable"
              icon="mdi-pencil"
              size="small"
              variant="text"
              @click="openRule(g, r)"
            />
            <v-btn
              color="error"
              density="comfortable"
              icon="mdi-delete"
              size="small"
              variant="text"
              @click="removeRule(g, r)"
            />
          </td>
        </tr>
      </tbody>
    </v-table>

    <v-dialog v-model="groupDialog" max-width="420">
      <v-card>
        <v-card-title>{{ editingGroup < 0 ? 'Add group' : 'Rename group' }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="groupName"
            autofocus
            density="compact"
            hide-details
            label="Name"
            placeholder="e.g. Exclusions, Winter sale"
            variant="outlined"
            @keyup.enter="applyGroup"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="groupDialog = false" />
          <v-btn color="primary" text="Done" variant="flat" @click="applyGroup" />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="ruleDialog" max-width="600">
      <v-card>
        <v-card-title>{{ editing.rule < 0 ? 'Add rule' : 'Edit rule' }}</v-card-title>
        <v-card-text>
          <div class="d-flex ga-3">
            <v-text-field
              v-model="form.name"
              density="compact"
              label="Name"
              placeholder="e.g. Lady Turkey"
              variant="outlined"
            />
            <v-select
              v-if="groups.length > 1"
              v-model="form.group"
              density="compact"
              :items="groups.map((group, g) => ({ title: group.name || 'Untitled group', value: g }))"
              label="Group"
              style="max-width: 200px"
              variant="outlined"
            />
          </div>
          <p class="text-body-2 text-medium-emphasis mb-3">
            A product matches when it fits every box that has something in it; within a box,
            any one value is enough.
          </p>
          <v-switch
            v-model="form.allProducts"
            color="primary"
            density="compact"
            hide-details
            label="Every product (no conditions)"
          />
          <div v-if="!form.allProducts" class="rule-grid">
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

          <v-select
            v-model="form.dealId"
            class="mt-4"
            density="compact"
            hide-details
            :items="dealChoices"
            label="Gives"
            style="max-width: 280px"
            variant="outlined"
          />
          <p v-if="form.dealId === NO_DISCOUNT" class="text-body-2 text-medium-emphasis mt-2">
            These products sell at full price, with no discount and no deal, and the rules below
            don't apply to them: put this rule above the ones it should keep them out of.
          </p>
          <div v-if="form.dealId === 0" class="d-flex flex-wrap ga-3 mt-4">
            <v-text-field
              v-model="form.discount"
              density="compact"
              hint="0 is the same as No discount"
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
  import { computed, reactive, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'

  const props = defineProps({
    listId: { type: Number, required: true },
    canEdit: { type: Boolean, default: false },
    saleOptions: { type: Array, default: () => [] },
    deals: { type: Array, default: () => [] },
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

  // [{ name, enabled, rules: [...] }], in order; rules run top to bottom
  // straight through the groups.
  const groups = ref([])
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
    return data.groups ?? []
  }

  async function load () {
    loading.value = true
    try {
      groups.value = await request('GET')
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

  // What a rule gives: no discount at all (full price, and the rules below
  // don't apply -- how exclusions are made), a discount (0), or a deal.
  const NO_DISCOUNT = -1
  const dealChoices = computed(() => [
    { title: 'No discount (full price)', value: NO_DISCOUNT },
    { title: 'A discount', value: 0 },
    ...props.deals.map(d => ({ title: d.name, value: d.id })),
  ])

  function hasConditions (rule) {
    return CONDITIONS.some(c => c.key !== 'except_code_prefixes' && (rule[c.key] ?? []).length > 0)
  }

  function conditionChips (rule) {
    const chips = CONDITIONS.flatMap(c => (rule[c.key] ?? []).length > 0
      ? [`${c.label}: ${rule[c.key].map(v => c.fmt(v)).join(', ')}`]
      : [])
    return hasConditions(rule) ? chips : ['Every product', ...chips]
  }

  function discountText (rule) {
    if (rule.deal_id) return props.deals.find(d => d.id === rule.deal_id)?.name ?? 'Deal'
    const parts = (rule.options ?? []).map(o => `${o.sale_option} ${o.discount}%`)
    const plain = rule.discount > 0 ? `${rule.discount}%` : 'No discount'
    return parts.length > 0 ? `${plain} · ${parts.join(' · ')}` : plain
  }

  const ruleCount = computed(() => groups.value.reduce((n, group) => n + group.rules.length, 0))

  // A rule's number on the page: counted straight through the groups, as
  // the rules run (and as the API's errors number them).
  function ruleNumber (g, r) {
    return groups.value.slice(0, g).reduce((n, group) => n + group.rules.length, 0) + r + 1
  }

  // On while every group and every rule is; switching it sets them all.
  const allOn = computed({
    get: () => groups.value.every(group => group.enabled && group.rules.every(rule => rule.enabled)),
    set: on => {
      for (const group of groups.value) {
        group.enabled = on
        for (const rule of group.rules) rule.enabled = on
      }
      dirty.value = true
    },
  })

  // Only this rule: every other group off (their rules keep their own
  // switches, so turning a group back on brings back what it had), and in
  // this group every other rule off.
  function onlyThis (g, r) {
    for (const [gi, group] of groups.value.entries()) {
      group.enabled = gi === g
      if (gi === g) {
        for (const [ri, rule] of group.rules.entries()) rule.enabled = ri === r
      }
    }
    dirty.value = true
  }

  function swap (list, i, step) {
    const next = [...list]
    ;[next[i], next[i + step]] = [next[i + step], next[i]]
    return next
  }

  function moveRule (g, r, step) {
    groups.value[g].rules = swap(groups.value[g].rules, r, step)
    dirty.value = true
  }

  function moveGroup (g, step) {
    groups.value = swap(groups.value, g, step)
    dirty.value = true
  }

  function removeRule (g, r) {
    groups.value[g].rules = groups.value[g].rules.filter((_, j) => j !== r)
    dirty.value = true
  }

  function removeGroup (g) {
    groups.value = groups.value.filter((_, j) => j !== g)
    dirty.value = true
  }

  const groupDialog = ref(false)
  const editingGroup = ref(-1)
  const groupName = ref('')

  function openGroup (g) {
    editingGroup.value = g
    groupName.value = g >= 0 ? groups.value[g].name : ''
    groupDialog.value = true
  }

  function applyGroup () {
    const name = groupName.value.trim()
    if (editingGroup.value < 0) groups.value = [...groups.value, { name, enabled: true, rules: [] }]
    else groups.value[editingGroup.value].name = name
    dirty.value = true
    groupDialog.value = false
  }

  const ruleDialog = ref(false)
  // Where the rule being edited is; rule -1 adds one to the group.
  const editing = reactive({ group: -1, rule: -1 })
  const form = reactive({ name: '', discount: '0', options: {}, dealId: 0, allProducts: false, group: 0 })

  function openRule (g, r) {
    loadAttributeValues()
    editing.group = g
    editing.rule = r
    const rule = r >= 0 ? groups.value[g].rules[r] : {}
    form.group = Math.max(g, 0)
    form.name = rule.name ?? ''
    for (const c of CONDITIONS) form[c.key] = [...(rule[c.key] ?? [])]
    form.discount = String(rule.discount ?? 0)
    // A saved rule giving nothing at all is shown as "No discount".
    const givesNothing = r >= 0 && !rule.deal_id && !rule.discount && (rule.options ?? []).length === 0
    form.dealId = givesNothing ? NO_DISCOUNT : (rule.deal_id ?? 0)
    // A saved rule with no conditions is an "every product" one.
    form.allProducts = r >= 0 && !hasConditions(rule)
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
    const noDiscount = form.dealId === NO_DISCOUNT
    const dealId = noDiscount ? 0 : form.dealId
    const discount = dealId || noDiscount ? 0 : percent(form.discount)
    if (discount === null) {
      notify.error('Enter a discount from 0 to 100.')
      return
    }
    const options = []
    // A deal replaces the discount, whatever the option, so it has no option discounts.
    for (const o of dealId || noDiscount ? [] : props.saleOptions) {
      const raw = form.options[o.id]
      if (raw === '' || raw == null) continue
      const d = percent(raw)
      if (d === null) {
        notify.error(`Enter a ${o.name} discount from 0 to 100, or leave it empty.`)
        return
      }
      options.push({ sale_option_id: o.id, sale_option: o.name, discount: d })
    }
    const old = editing.rule >= 0 ? groups.value[editing.group].rules[editing.rule] : null
    const rule = { name: form.name.trim(), discount, options, deal_id: dealId, enabled: old?.enabled ?? true, variants: 0 }
    for (const c of CONDITIONS) {
      rule[c.key] = form.allProducts ? [] : form[c.key].map(v => String(v).trim()).filter(Boolean)
    }
    rule.all_products = form.allProducts
    if (!form.allProducts && !hasConditions(rule)) {
      notify.error('Give the rule at least one condition: a code start, kind, category, origin or brand.')
      return
    }
    // The first rule of a list with no groups makes one for it.
    if (groups.value.length === 0) groups.value = [{ name: 'Rules', enabled: true, rules: [] }]
    const target = Math.min(form.group, groups.value.length - 1)
    if (old && target === editing.group) {
      groups.value[target].rules[editing.rule] = rule
    } else {
      if (old) removeRule(editing.group, editing.rule)
      groups.value[target].rules = [...groups.value[target].rules, rule]
    }
    dirty.value = true
    ruleDialog.value = false
  }

  async function save () {
    saving.value = true
    try {
      // A rule with no conditions was made "every product" on purpose (the
      // only way to make one here), so it says so to the API.
      groups.value = await request('PUT', {
        groups: groups.value.map(group => ({
          name: group.name,
          enabled: group.enabled,
          rules: group.rules.map(r => ({ ...r, all_products: !hasConditions(r) })),
        })),
      })
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
.group-row td {
  background: rgba(var(--v-theme-on-surface), 0.04);
}
/* A rule that is off, or in a group that is off, decides nothing. */
.rule-off td:not(:first-child) {
  opacity: 0.45;
}
</style>
