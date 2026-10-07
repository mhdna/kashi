<template>
  <div class="page-root">
    <v-card class="afr-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-auto-fix" />
        <span>Auto-fill Rules</span>
        <v-chip
          :color="settingsStore.productAutofillEnabled ? 'success' : undefined"
          size="small"
          :text="settingsStore.productAutofillEnabled ? 'On' : 'Off in Settings'"
          to="/settings"
          variant="tonal"
        />
        <v-spacer />
        <template v-if="canEdit">
          <v-btn prepend-icon="mdi-plus" text="Add rule" variant="tonal" @click="addRule" />
          <v-btn :disabled="!dirty" text="Discard" variant="text" @click="seedDraft" />
          <v-btn
            color="primary"
            :disabled="!dirty || hasErrors"
            :loading="saving"
            text="Save"
            variant="flat"
            @click="save"
          />
        </template>
      </v-card-title>
      <v-divider />

      <div class="afr-body pa-4">
        <p class="text-body-2 text-medium-emphasis mb-4">
          The Auto-fill button on the New Product dialog runs these rules over the product's
          code and name. Rules run top to bottom and the first one that matches a field fills
          it, so put specific rules above general ones (e.g. &ldquo;t-shirt&rdquo; above
          &ldquo;shirt&rdquo;). Matching ignores case, and a pattern can list alternatives
          with <code>|</code>, like <code>boy|boys</code>.
        </p>

        <!-- Code layout: splits a code into named parts for "Code part is" rules. -->
        <v-sheet border class="pa-3 mb-4" rounded>
          <div class="text-overline text-medium-emphasis mb-1">Code layout</div>
          <p class="text-body-2 text-medium-emphasis mb-3">
            A regular expression whose named groups, written <code>(?&lt;name&gt;...)</code>,
            split a code into parts that &ldquo;Code part is&rdquo; rules can match. The default
            reads codes like <code>CAJUL25KBPA22</code> as origin CA, month JUL, year 25,
            audience KB, type PA and price 22. A code the layout doesn't fit simply has no parts.
          </p>
          <v-text-field
            v-model="draftLayout"
            class="afr-mono"
            density="compact"
            :disabled="!canEdit"
            :error-messages="layoutError"
            :hint="layoutParts.length > 0 ? `Parts: ${layoutParts.join(', ')}` : ''"
            label="Layout"
            persistent-hint
            variant="outlined"
          />
        </v-sheet>

        <!-- Try it: runs the unsaved rules, so edits can be checked before saving. -->
        <v-sheet border class="pa-3 mb-4" rounded>
          <div class="text-overline text-medium-emphasis mb-2">Try it</div>
          <v-row density="compact">
            <v-col cols="12" sm="4">
              <v-text-field
                v-model="testCode"
                density="compact"
                hide-details
                label="Code"
                placeholder="L6KBSWT25"
                variant="outlined"
              />
            </v-col>
            <v-col cols="12" sm="8">
              <v-text-field
                v-model="testName"
                density="compact"
                hide-details
                label="Name"
                placeholder="Kids boy sweatshirt"
                variant="outlined"
              />
            </v-col>
          </v-row>
          <div v-if="testPartList.length > 0" class="d-flex flex-wrap align-center ga-2 mt-3">
            <span class="text-caption text-medium-emphasis">Code parts:</span>
            <v-chip
              v-for="p in testPartList"
              :key="p.name"
              class="afr-mono"
              size="small"
              variant="outlined"
            >
              {{ p.name }} <strong class="ms-1">{{ p.value }}</strong>
            </v-chip>
          </div>
          <div class="d-flex flex-wrap ga-2 mt-3">
            <span v-if="!testCode && !testName" class="text-caption text-medium-emphasis">
              Type a code or name to see what gets filled.
            </span>
            <span v-else-if="testFilled.length === 0" class="text-caption text-medium-emphasis">
              No rule matches.
            </span>
            <v-chip
              v-for="f in testFilled"
              :key="f.key"
              color="primary"
              size="small"
              variant="tonal"
            >
              {{ f.label }}: <strong class="ms-1">{{ f.value }}</strong>
              <span class="text-medium-emphasis ms-1">(rule {{ f.rule + 1 }})</span>
            </v-chip>
          </div>
        </v-sheet>

        <v-text-field
          v-model="filter"
          class="afr-filter mb-2"
          clearable
          density="compact"
          hide-details
          label="Find rules"
          placeholder="e.g. jeans, JE or origin"
          variant="outlined"
        />

        <div class="afr-table-wrap">
          <v-table class="afr-table" density="compact">
            <thead>
              <tr>
                <th class="afr-num">#</th>
                <th>Read from</th>
                <th>When</th>
                <th class="afr-pos">Position / part</th>
                <th>Pattern</th>
                <th>Fill</th>
                <th>With value</th>
                <th v-if="canEdit" />
              </tr>
            </thead>
            <tbody>
              <tr v-if="draft.length === 0">
                <td class="text-medium-emphasis text-center py-6" colspan="8">
                  No rules yet.
                </td>
              </tr>
              <tr
                v-for="{ rule, i } in visibleRules"
                :key="rule._key"
                :class="{ 'afr-hit': winningRules.has(i) }"
              >
                <td class="afr-num text-medium-emphasis">{{ i + 1 }}</td>
                <td class="afr-source">
                  <v-select
                    v-model="rule.source"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    :items="AUTOFILL_SOURCES"
                    variant="plain"
                  />
                </td>
                <td class="afr-match">
                  <v-select
                    v-model="rule.match"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    :items="AUTOFILL_MATCHES"
                    variant="plain"
                  />
                </td>
                <td class="afr-pos">
                  <v-text-field
                    v-if="rule.match === 'char'"
                    v-model.number="rule.position"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    min="1"
                    type="number"
                    variant="plain"
                  />
                  <v-select
                    v-else-if="rule.match === 'part'"
                    v-model="rule.part"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    :items="layoutParts"
                    variant="plain"
                  />
                </td>
                <td class="afr-pattern">
                  <v-text-field
                    v-if="rule.match !== 'trailing_number'"
                    v-model="rule.pattern"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    :placeholder="patternPlaceholder(rule.match)"
                    variant="plain"
                  />
                </td>
                <td class="afr-field">
                  <v-select
                    v-model="rule.field"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    item-title="label"
                    item-value="key"
                    :items="AUTOFILL_FIELDS"
                    variant="plain"
                  />
                </td>
                <td class="afr-value">
                  <span v-if="rule.match === 'trailing_number'" class="text-caption text-medium-emphasis">
                    the number itself
                  </span>
                  <v-text-field
                    v-else
                    v-model="rule.value"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    :placeholder="valuePlaceholder(rule.match)"
                    variant="plain"
                  />
                  <div v-if="ruleErrors[i]" class="text-caption text-error">{{ ruleErrors[i] }}</div>
                </td>
                <td v-if="canEdit" class="afr-actions">
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
                    :disabled="i === draft.length - 1"
                    icon="mdi-arrow-down"
                    size="small"
                    variant="text"
                    @click="move(i, 1)"
                  />
                  <v-btn
                    color="error"
                    density="comfortable"
                    icon="mdi-delete"
                    size="small"
                    variant="text"
                    @click="draft.splice(i, 1)"
                  />
                </td>
              </tr>
            </tbody>
          </v-table>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref, watch } from 'vue'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useSettingsStore } from '@/stores/settings'
  import {
    applyAutofillRules,
    AUTOFILL_FIELDS,
    AUTOFILL_MATCHES,
    AUTOFILL_SOURCES,
    autofillRuleError,
    codeLayoutError,
    codeLayoutParts,
  } from '@/utils/productAutofill'

  const { canEdit } = usePageAccess()
  const notify = useNotify()
  const settingsStore = useSettingsStore()

  // The page edits a local copy and writes the whole list on Save. `_key` is a
  // row id for v-for only; it's stripped before saving.
  let nextKey = 0
  const draft = ref([])
  const draftLayout = ref('')

  function strip (rules) {
    return rules.map(({ source, match, position, pattern, field, value, part }) => ({
      source,
      match,
      position: Number(position) || 1,
      pattern: pattern ?? '',
      field,
      value: value ?? '',
      part: part ?? '',
    }))
  }

  function seedDraft () {
    draft.value = strip(settingsStore.productAutofillRules).map(r => ({ ...r, _key: nextKey++ }))
    draftLayout.value = settingsStore.productAutofillCodeLayout
  }

  onMounted(async () => {
    await settingsStore.init()
    seedDraft()
  })
  watch(() => settingsStore.loaded, loaded => {
    if (loaded && !dirty.value) seedDraft()
  })

  const dirty = computed(() =>
    draftLayout.value !== settingsStore.productAutofillCodeLayout
    || JSON.stringify(strip(draft.value)) !== JSON.stringify(strip(settingsStore.productAutofillRules)),
  )
  const layoutParts = computed(() => codeLayoutParts(draftLayout.value))
  const layoutError = computed(() => codeLayoutError(draftLayout.value))
  const ruleErrors = computed(() => draft.value.map(r => autofillRuleError(r)))
  const hasErrors = computed(() => Boolean(layoutError.value) || ruleErrors.value.some(Boolean))

  function patternPlaceholder (match) {
    if (match === 'regex') {return '^..([A-Z]+)'}
    if (match === 'part') {return 'e.g. JE or SWE|SWT (empty = any)'}
    return 'e.g. L or boy|boys'
  }
  function valuePlaceholder (match) {
    if (match === 'regex') {return '$1 (empty = group 1)'}
    if (match === 'part') {return 'empty = the part itself'}
    return ''
  }

  // "Find rules" narrows the table; rows keep their real number, so moving
  // and the Try it highlight still refer to the full list.
  const filter = ref('')
  const visibleRules = computed(() => {
    const rows = draft.value.map((rule, i) => ({ rule, i }))
    const q = (filter.value ?? '').trim().toLowerCase()
    if (!q) {return rows}
    return rows.filter(({ rule }) =>
      [rule.pattern, rule.value, rule.field, rule.part, rule.source]
        .some(v => String(v ?? '').toLowerCase().includes(q)))
  })

  function addRule () {
    draft.value.push({
      _key: nextKey++,
      source: 'name',
      match: 'word',
      position: 1,
      pattern: '',
      field: 'type',
      value: '',
      part: '',
    })
  }

  function move (i, delta) {
    const rows = draft.value
    const [row] = rows.splice(i, 1)
    rows.splice(i + delta, 0, row)
  }

  const saving = ref(false)
  async function save () {
    if (!dirty.value || hasErrors.value) return
    saving.value = true
    try {
      await settingsStore.saveProductAutofillRules(strip(draft.value), draftLayout.value)
      seedDraft()
      notify.success('Auto-fill rules saved.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  // ---- try it ------------------------------------------------------------
  const testCode = ref('')
  const testName = ref('')
  const testResult = computed(() =>
    applyAutofillRules(draft.value, { code: testCode.value, name: testName.value }, draftLayout.value),
  )
  const testPartList = computed(() =>
    Object.entries(testResult.value.parts).map(([name, value]) => ({ name, value })))
  const testFilled = computed(() =>
    AUTOFILL_FIELDS
      .filter(f => f.key in testResult.value.values)
      .map(f => ({
        key: f.key,
        label: f.label,
        value: testResult.value.values[f.key],
        rule: testResult.value.ruleFor[f.key],
      })),
  )
  const winningRules = computed(() => new Set(Object.values(testResult.value.ruleFor)))
</script>

<style scoped>
.page-root, .afr-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}
.afr-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
}
.afr-table-wrap {
  overflow-x: auto;
}
.afr-table {
  min-width: 920px;
}
.afr-table td {
  vertical-align: middle;
}
.afr-num {
  width: 36px;
}
.afr-source {
  width: 110px;
}
.afr-match {
  width: 210px;
}
.afr-pos {
  width: 90px;
}
.afr-field {
  width: 150px;
}
.afr-actions {
  width: 120px;
  white-space: nowrap;
}
.afr-mono :deep(input),
.afr-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.afr-filter {
  max-width: 320px;
}
.afr-hit {
  background: rgba(var(--v-theme-primary), 0.08);
}
</style>
