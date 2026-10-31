<template>
  <div class="pa-4 pt-3">
    <div class="d-flex align-center flex-wrap ga-2 mb-3">
      <p class="text-body-2 text-medium-emphasis wr-intro mb-0">
        After each sale to a client with a phone number, these rules run top to bottom and the
        <strong>first one that matches</strong> decides: send a message (with its own template, or
        the default) or send nothing. A sale no rule matches gets the default template
        <template v-if="defaultTemplate"><code>{{ defaultTemplate }}</code></template>
        from <router-link to="/settings?tab=whatsapp">Settings</router-link>. To only message the
        sales you pick, end with a rule that has no conditions set to &ldquo;Don't send&rdquo;.
      </p>
      <v-spacer />
      <template v-if="canEdit">
        <v-menu>
          <template #activator="{ props: menuProps }">
            <v-btn v-bind="menuProps" prepend-icon="mdi-plus" text="Add rule" variant="tonal" />
          </template>
          <v-list density="compact">
            <v-list-item
              v-for="p in PRESETS"
              :key="p.key"
              :subtitle="p.subtitle"
              :title="p.title"
              @click="addRule(p.key)"
            />
          </v-list>
        </v-menu>
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
    </div>

    <!-- Try it: runs the unsaved list on the server, so edits can be checked before saving. -->
    <v-sheet border class="pa-3 mb-4" rounded>
      <div class="text-overline text-medium-emphasis mb-2">Try it</div>
      <v-row density="compact">
        <v-col cols="12" md="2" sm="4">
          <v-select
            v-model="test.branch_id"
            density="compact"
            hide-details
            :items="branchItems"
            label="Branch"
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="3" sm="8">
          <v-text-field
            v-model="test.phone"
            density="compact"
            hide-details
            label="Client phone"
            placeholder="+961 70 123 456"
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="2" sm="4">
          <v-text-field
            v-model="test.total"
            density="compact"
            hide-details
            label="Total"
            min="0"
            prefix="$"
            type="number"
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="3" sm="8">
          <v-combobox
            v-model="test.codes"
            chips
            closable-chips
            density="compact"
            hide-details
            label="Item codes sold"
            multiple
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="2" sm="12">
          <v-combobox
            v-model="test.categories"
            chips
            closable-chips
            density="compact"
            hide-details
            :items="categoryNames"
            label="Categories"
            multiple
            variant="outlined"
          />
        </v-col>
      </v-row>
      <div class="d-flex align-center flex-wrap ga-2 mt-3 text-body-2">
        <PreparingState v-if="testing" kind="chat" :size="24" title="Loading" />
        <template v-else-if="testResult">
          <v-chip
            :color="testResult.action === 'skip' ? undefined : 'success'"
            :prepend-icon="testResult.action === 'skip' ? 'mdi-cancel' : 'mdi-send-outline'"
            size="small"
            variant="tonal"
          >
            {{ testResult.action === 'skip' ? "Don't send" : 'Send' }}
          </v-chip>
          <span v-if="testResult.rule_index >= 0">
            decided by <strong>{{ testResult.rule_index + 1 }}. {{ testResult.rule_name }}</strong>
          </span>
          <span v-else class="text-medium-emphasis">no rule matches — the default applies</span>
          <span v-if="testResult.action !== 'skip'" class="text-medium-emphasis">
            · template <code>{{ testResult.template_name || '(none set)' }}</code>
            ({{ testResult.template_lang }}) filled with <code>{{ testResult.template_params || '—' }}</code>
          </span>
          <span v-if="testResult.last_sent_at" class="text-caption text-medium-emphasis">
            · this number was last messaged {{ new Date(testResult.last_sent_at).toLocaleDateString() }}
          </span>
        </template>
      </div>
    </v-sheet>

    <PreparingState v-if="loading" kind="chat" min-height="240px" title="Loading message rules" />
    <div v-else-if="draft.length === 0" class="text-body-2 text-medium-emphasis text-center py-8">
      No rules: every sale to a client with a phone gets the default message.
    </div>

    <v-card
      v-for="(rule, i) in draft"
      :key="rule._key"
      class="wr-rule mb-3"
      :class="{ 'wr-rule--hit': testResult?.rule_index === i, 'wr-rule--off': !rule.enabled }"
      variant="flat"
    >
      <div class="d-flex align-center flex-wrap ga-2 pa-3 pb-1">
        <span class="wr-num text-medium-emphasis">{{ i + 1 }}</span>
        <v-switch
          v-model="rule.enabled"
          color="primary"
          density="compact"
          :disabled="!canEdit"
          hide-details
          inset
          :title="rule.enabled ? 'On' : 'Off'"
        />
        <v-text-field
          v-model="rule.name"
          class="wr-name"
          density="compact"
          :disabled="!canEdit"
          hide-details
          placeholder="Rule name"
          variant="plain"
        />
        <v-spacer />
        <v-btn-toggle
          v-model="rule.action"
          color="primary"
          density="compact"
          :disabled="!canEdit"
          mandatory
          variant="outlined"
        >
          <v-btn prepend-icon="mdi-send-outline" text="Send" value="send" />
          <v-btn prepend-icon="mdi-cancel" text="Don't send" value="skip" />
        </v-btn-toggle>
        <template v-if="canEdit">
          <v-btn
            density="comfortable"
            :disabled="i === 0"
            icon="mdi-arrow-up"
            size="small"
            title="Move up"
            variant="text"
            @click="move(i, -1)"
          />
          <v-btn
            density="comfortable"
            :disabled="i === draft.length - 1"
            icon="mdi-arrow-down"
            size="small"
            title="Move down"
            variant="text"
            @click="move(i, 1)"
          />
          <v-btn
            color="error"
            density="comfortable"
            icon="mdi-delete"
            size="small"
            title="Delete rule"
            variant="text"
            @click="draft.splice(i, 1)"
          />
        </template>
      </div>

      <div class="px-3 pb-1 text-body-2 text-medium-emphasis">{{ summary(rule) }}</div>

      <div class="pa-3 pt-2">
        <div class="text-caption text-medium-emphasis mb-2">When the sale… (leave a field empty to ignore it)</div>
        <v-row density="compact">
          <v-col cols="6" md="2">
            <v-text-field
              v-model="rule.min_total"
              density="compact"
              :disabled="!canEdit"
              hide-details="auto"
              label="Total at least"
              min="0"
              prefix="$"
              type="number"
              variant="outlined"
            />
          </v-col>
          <v-col cols="6" md="2">
            <v-text-field
              v-model="rule.max_total"
              density="compact"
              :disabled="!canEdit"
              :error-messages="errors[i]?.total"
              hide-details="auto"
              label="Total at most"
              min="0"
              prefix="$"
              type="number"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" md="3">
            <v-select
              v-model="rule.branch_ids"
              chips
              closable-chips
              density="compact"
              :disabled="!canEdit"
              hide-details
              :items="branchItems.filter(b => b.value)"
              label="At branches"
              multiple
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" md="5">
            <v-autocomplete
              v-model="rule.client_phones"
              chips
              closable-chips
              density="compact"
              :disabled="!canEdit"
              hide-details
              hide-no-data
              item-title="label"
              item-value="digits"
              :items="clientItems(rule)"
              label="Client is"
              multiple
              no-filter
              placeholder="Search a name or type a number"
              variant="outlined"
              @update:search="searchClients"
            />
          </v-col>
          <v-col cols="12" md="4">
            <v-combobox
              v-model="rule.product_codes"
              chips
              closable-chips
              density="compact"
              :disabled="!canEdit"
              hint="Press Enter after each. JE* = every code starting with JE."
              label="Includes item code"
              multiple
              persistent-hint
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" md="4">
            <v-combobox
              v-model="rule.categories"
              chips
              closable-chips
              density="compact"
              :disabled="!canEdit"
              hide-details
              :items="categoryNames"
              label="Includes an item in category"
              multiple
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" md="4">
            <v-text-field
              v-model="rule.messaged_within_days"
              density="compact"
              :disabled="!canEdit"
              hint="Matches a client already messaged this recently — use with Don't send to avoid spamming."
              label="Client was messaged in the last … days"
              min="0"
              persistent-hint
              type="number"
              variant="outlined"
            />
          </v-col>
        </v-row>

        <template v-if="rule.action === 'send'">
          <div class="text-caption text-medium-emphasis mt-3 mb-2">
            Send this approved template (empty = the default from Settings)
          </div>
          <v-row density="compact">
            <v-col cols="12" md="4">
              <v-text-field
                v-model="rule.template_name"
                density="compact"
                :disabled="!canEdit"
                hide-details
                label="Template name"
                :placeholder="defaultTemplate || 'from Settings'"
                variant="outlined"
              />
            </v-col>
            <v-col cols="6" md="2">
              <v-text-field
                v-model="rule.template_lang"
                density="compact"
                :disabled="!canEdit"
                hide-details
                label="Language"
                :placeholder="settings?.whatsapp_template_lang || 'en_US'"
                variant="outlined"
              />
            </v-col>
            <v-col cols="6" md="6">
              <v-text-field
                v-model="rule.template_params"
                density="compact"
                :disabled="!canEdit"
                :error-messages="errors[i]?.params"
                hide-details="auto"
                label="Body params"
                :placeholder="settings?.whatsapp_template_params || 'name,invoice,total'"
                variant="outlined"
              />
            </v-col>
          </v-row>
        </template>
      </div>
    </v-card>

    <div v-if="draft.length > 0" class="d-flex align-center ga-2 text-body-2 text-medium-emphasis wr-default pa-3">
      <v-icon icon="mdi-arrow-down" size="18" />
      Nothing above matched: send the default template
      <code v-if="defaultTemplate">{{ defaultTemplate }}</code>.
    </div>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { API_BASE } from '@/config'
  import { formatPhone, TEMPLATE_PARAMS } from '@/utils/whatsapp'

  const props = defineProps({
    branches: { type: Array, default: () => [] },
    // GET /whatsapp_settings: the default template a rule falls back to.
    settings: { type: Object, default: null },
  })

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { getRules, saveRules, testRules, searchClients: findClients, clientsByPhones } = useWhatsApp()

  const defaultTemplate = computed(() => props.settings?.whatsapp_template_name ?? '')
  const branchItems = computed(() => [
    { title: 'Any branch', value: 0 },
    ...props.branches.map(b => ({ title: b.name, value: b.id })),
  ])
  const branchName = id => props.branches.find(b => b.id === id)?.name ?? `branch ${id}`

  // ---- draft -----------------------------------------------------------------
  // The page edits a flat copy (dollars, plain lists) and writes the whole
  // list on Save; `_key` is a v-for id only.
  let nextKey = 0
  const saved = ref([])
  const draft = ref([])
  const loading = ref(false)

  function toDraft (rule, key = nextKey++) {
    const c = rule.conditions ?? {}
    return {
      _key: key,
      name: rule.name ?? '',
      enabled: rule.enabled ?? true,
      action: rule.action ?? 'send',
      min_total: c.min_total_cents == null ? '' : String(c.min_total_cents / 100),
      max_total: c.max_total_cents == null ? '' : String(c.max_total_cents / 100),
      branch_ids: c.branch_ids ?? [],
      client_phones: c.client_phones ?? [],
      product_codes: c.product_codes ?? [],
      categories: c.categories ?? [],
      messaged_within_days: c.messaged_within_days ? String(c.messaged_within_days) : '',
      template_name: rule.template_name ?? '',
      template_lang: rule.template_lang ?? '',
      template_params: rule.template_params ?? '',
    }
  }

  function cents (value) {
    if (value === '' || value == null) return null
    const n = Number(value)
    return Number.isFinite(n) ? Math.round(n * 100) : null
  }

  function toApi (rule) {
    const conditions = {}
    const min = cents(rule.min_total)
    const max = cents(rule.max_total)
    if (min != null) conditions.min_total_cents = min
    if (max != null) conditions.max_total_cents = max
    if (rule.branch_ids.length > 0) conditions.branch_ids = [...rule.branch_ids]
    if (rule.client_phones.length > 0) conditions.client_phones = [...rule.client_phones]
    if (rule.product_codes.length > 0) conditions.product_codes = rule.product_codes.map(c => String(c).trim().toUpperCase())
    if (rule.categories.length > 0) conditions.categories = rule.categories.map(c => String(c).trim())
    const days = Number(rule.messaged_within_days) || 0
    if (days > 0) conditions.messaged_within_days = Math.round(days)
    const send = rule.action === 'send'
    return {
      name: rule.name.trim(),
      enabled: rule.enabled,
      action: rule.action,
      conditions,
      template_name: send ? rule.template_name.trim() : '',
      template_lang: send ? rule.template_lang.trim() : '',
      template_params: send ? rule.template_params.trim() : '',
    }
  }

  function seedDraft () {
    draft.value = saved.value.map(r => toDraft(r))
  }

  const savedJSON = computed(() => JSON.stringify(saved.value.map(r => toApi(toDraft(r, 0)))))
  const dirty = computed(() => JSON.stringify(draft.value.map(r => toApi(r))) !== savedJSON.value)

  const errors = computed(() => draft.value.map(rule => {
    const e = {}
    const min = cents(rule.min_total)
    const max = cents(rule.max_total)
    if ((min != null && min < 0) || (max != null && max < 0)) e.total = "Totals can't be negative"
    else if (min != null && max != null && min > max) e.total = 'Below the minimum'
    const unknown = rule.template_params
      .split(',')
      .map(p => p.trim())
      .filter(p => p && !TEMPLATE_PARAMS.includes(p))
    if (unknown.length > 0) e.params = `Unknown: ${unknown.join(', ')} (use ${TEMPLATE_PARAMS.join(', ')})`
    return e
  }))
  const hasErrors = computed(() => errors.value.some(e => Object.keys(e).length > 0))

  async function load () {
    loading.value = true
    try {
      saved.value = await getRules()
      seedDraft()
      await rememberClients(saved.value.flatMap(r => r.conditions?.client_phones ?? []))
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  const saving = ref(false)
  async function save () {
    if (!dirty.value || hasErrors.value) return
    saving.value = true
    try {
      saved.value = await saveRules(draft.value.map(r => toApi(r)))
      seedDraft()
      notify.success('WhatsApp rules saved.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  function move (i, delta) {
    const rows = draft.value
    const [row] = rows.splice(i, 1)
    rows.splice(i + delta, 0, row)
  }

  // ---- adding ----------------------------------------------------------------
  const PRESETS = [
    { key: 'min-total', title: 'Only message bigger sales', subtitle: "Don't send when the total is under $50" },
    { key: 'clients', title: 'Never message some clients', subtitle: "Don't send to the clients you pick" },
    { key: 'item', title: 'Different message for an item', subtitle: 'Send another template when a code is sold' },
    { key: 'cooldown', title: "Don't message the same client too often", subtitle: 'Skip clients messaged in the last 7 days' },
    { key: 'blank', title: 'Blank rule', subtitle: 'Set every condition yourself' },
  ]

  function addRule (preset) {
    const rule = toDraft({ enabled: true, action: 'skip', conditions: {} })
    switch (preset) {
      case 'min-total': {
        Object.assign(rule, { name: 'Under $50', max_total: '49.99' })
        break
      }
      case 'clients': {
        rule.name = 'Never message'
        break
      }
      case 'item': {
        Object.assign(rule, { name: 'Item sold', action: 'send' })
        break
      }
      case 'cooldown': {
        Object.assign(rule, { name: 'Messaged this week', messaged_within_days: '7' })
        break
      }
      default: {
        rule.action = 'send'
      }
    }
    // Skips usually guard everything below them, so they go on top; a
    // special message goes above the general ones too, but under the skips.
    const firstSend = draft.value.findIndex(r => r.action === 'send')
    if (rule.action === 'skip') draft.value.unshift(rule)
    else if (firstSend === -1) draft.value.push(rule)
    else draft.value.splice(firstSend, 0, rule)
  }

  // ---- summary ---------------------------------------------------------------
  function summary (rule) {
    const parts = []
    const min = cents(rule.min_total)
    const max = cents(rule.max_total)
    if (min != null && max != null) parts.push(`the total is $${min / 100}–$${max / 100}`)
    else if (min != null) parts.push(`the total is at least $${min / 100}`)
    else if (max != null) parts.push(`the total is at most $${max / 100}`)
    if (rule.branch_ids.length > 0) parts.push(`it's at ${rule.branch_ids.map(id => branchName(id)).join(' or ')}`)
    if (rule.client_phones.length > 0) {
      const names = rule.client_phones.map(p => clientLabel(p))
      parts.push(`the client is ${names.length > 3 ? `one of ${names.length} clients` : names.join(' or ')}`)
    }
    if (rule.product_codes.length > 0) parts.push(`it includes ${rule.product_codes.join(', ')}`)
    if (rule.categories.length > 0) parts.push(`it includes ${rule.categories.join(' or ')}`)
    const days = Number(rule.messaged_within_days) || 0
    if (days > 0) parts.push(`the client was messaged in the last ${days} day${days === 1 ? '' : 's'}`)

    const when = parts.length > 0 ? `When ${parts.join(' and ')}` : 'Every sale'
    if (rule.action === 'skip') return `${when}: don't send anything.`
    const tpl = rule.template_name.trim() || defaultTemplate.value || 'the default template'
    return `${when}: send ${tpl}.`
  }

  // ---- clients ---------------------------------------------------------------
  // Names for phones, from searches and from the saved rules; a rule stores
  // digits only.
  const knownClients = reactive(new Map())
  const searchResults = ref([])
  const searchText = ref('')

  function remember (list) {
    for (const c of list) {
      if (c.digits && !knownClients.has(c.digits)) knownClients.set(c.digits, c)
    }
  }
  async function rememberClients (phones) {
    const unknown = [...new Set(phones)].filter(p => !knownClients.has(p))
    if (unknown.length === 0) return
    try {
      remember(await clientsByPhones(unknown))
    } catch { /* chips show the bare number */ }
  }
  function clientLabel (digits) {
    const c = knownClients.get(digits)
      ?? [...knownClients.values()].find(k => k.digits.slice(-8) === digits.slice(-8))
    return c ? `${c.name} · ${formatPhone(digits)}` : formatPhone(digits)
  }

  let searchTimer = 0
  function searchClients (text) {
    searchText.value = text ?? ''
    clearTimeout(searchTimer)
    if (!searchText.value.trim()) {
      searchResults.value = []
      return
    }
    searchTimer = setTimeout(async () => {
      try {
        const found = await findClients(searchText.value.trim())
        remember(found)
        searchResults.value = found
      } catch {
        searchResults.value = []
      }
    }, 250)
  }

  // The rule's own phones, search hits, and a typed number as itself.
  function clientItems (rule) {
    const items = new Map()
    for (const digits of rule.client_phones) items.set(digits, { digits, label: clientLabel(digits) })
    for (const c of searchResults.value) {
      if (!items.has(c.digits)) items.set(c.digits, { digits: c.digits, label: `${c.name} · ${formatPhone(c.digits)}` })
    }
    const typed = searchText.value.replaceAll(/\D/g, '')
    if (typed.length >= 8 && !items.has(typed)) {
      items.set(typed, { digits: typed, label: `${formatPhone(typed)} (this number)` })
    }
    return [...items.values()]
  }

  // ---- categories --------------------------------------------------------------
  const categoryNames = ref([])
  async function loadCategories () {
    try {
      const res = await authFetch(`${API_BASE}/product_categories`)
      if (!res.ok) return
      const rows = await res.json()
      categoryNames.value = (Array.isArray(rows) ? rows : [])
        .filter(c => c.is_active !== false)
        .map(c => c.name)
        .toSorted((a, b) => a.localeCompare(b))
    } catch { /* free typing still works */ }
  }

  // ---- try it ------------------------------------------------------------------
  const test = reactive({ branch_id: 0, phone: '', total: '60', codes: [], categories: [] })
  const testResult = ref(null)
  const testing = ref(false)
  let testTimer = 0
  let testSeq = 0
  function runTest () {
    clearTimeout(testTimer)
    testTimer = setTimeout(async () => {
      const seq = ++testSeq
      testing.value = true
      try {
        const result = await testRules({
          rules: draft.value.map(r => toApi(r)),
          branch_id: test.branch_id,
          client_phone: test.phone,
          total_cents: cents(test.total) ?? 0,
          items: [
            ...test.codes.map(code => ({ code: String(code) })),
            ...test.categories.map(category => ({ category: String(category) })),
          ],
        })
        if (seq === testSeq) testResult.value = result
      } catch (error) {
        if (seq === testSeq) {
          testResult.value = null
          notify.error(error.message)
        }
      } finally {
        if (seq === testSeq) testing.value = false
      }
    }, 300)
  }
  watch([() => JSON.stringify(draft.value.map(r => toApi(r))), () => JSON.stringify(test)], runTest)

  onMounted(async () => {
    await Promise.all([load(), loadCategories()])
    runTest()
  })
  onUnmounted(() => {
    clearTimeout(testTimer)
    clearTimeout(searchTimer)
  })
</script>

<style scoped>
.wr-intro {
  max-width: 900px;
}
.wr-rule {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
}
.wr-rule--hit {
  border-color: rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.06);
}
.wr-rule--off {
  opacity: 0.6;
}
.wr-num {
  width: 20px;
  text-align: right;
  font-weight: 600;
}
.wr-name {
  min-width: 180px;
  max-width: 360px;
}
.wr-name :deep(input) {
  font-weight: 600;
}
.wr-default {
  border: 1px dashed rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
}
</style>
