<template>
  <div class="page-root">
    <v-card class="dbs-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-content-copy" />
        <span>Default Settings</span>
        <v-spacer />
        <v-btn
          v-if="canEdit"
          color="primary"
          :disabled="loading"
          prepend-icon="mdi-send-outline"
          text="Apply to all branches"
          variant="tonal"
          @click="confirmApplyOpen = true"
        />
        <v-btn
          v-if="canEdit"
          color="primary"
          :loading="saving"
          text="Save"
          variant="flat"
          @click="save"
        />
      </v-card-title>
      <v-divider />

      <p class="text-body-2 text-medium-emphasis px-4 pt-3">
        A reusable template for the operational settings every branch shares &mdash; pricing,
        behaviour, page access, printing and attendance. Branch identity (name, storefront
        contact details), per-device fields (printer/COM ids, Akuvox IP) and secrets are never
        part of this template and stay branch-specific; edit those from
        <RouterLink to="/branch-settings">Branch Settings</RouterLink> instead.
        Saving here only updates the template &mdash; nothing is pushed to a branch until you
        click "Apply to all branches".
      </p>

      <div v-if="loading" class="d-flex justify-center pa-8">
        <v-progress-circular color="primary" indeterminate />
      </div>

      <div v-else class="dbs-detail">
        <v-tabs
          v-model="sectionTab"
          class="dbs-section-tabs rail-tabs"
          color="primary"
          :direction="$vuetify.display.smAndDown ? 'horizontal' : 'vertical'"
        >
          <v-tab class="text-none" text="Pricing" value="pricing" />
          <v-tab class="text-none" text="Behaviour" value="behaviour" />
          <v-tab class="text-none" text="Cash &amp; shifts" value="cash" />
          <v-tab class="text-none" text="Page access" value="pages" />
          <v-tab class="text-none" text="Printing" value="printing" />
          <v-tab class="text-none" text="Attendance" value="attendance" />
          <v-tab class="text-none" text="Till display" value="display" />
        </v-tabs>

        <v-divider vertical />

        <div class="dbs-section-pane">
          <v-form :disabled="!canEdit">
            <div v-show="sectionTab === 'pricing'">
              <section class="dbs-section">
                <div class="dbs-section-title">Pricing &amp; Exchange</div>
                <div class="field-grid">
                  <v-text-field
                    v-model.number="form.tax_rate"
                    density="compact"
                    hide-details="auto"
                    label="Tax Rate (0-1)"
                    max="1"
                    min="0"
                    step="0.01"
                    type="number"
                    variant="outlined"
                  />
                  <v-select
                    v-model="form.rounding_mode"
                    density="compact"
                    hide-details="auto"
                    :items="['half_up', 'up', 'down']"
                    label="Rounding Mode"
                    variant="outlined"
                  />
                  <CurrencySelect
                    v-model="form.rounding_currency"
                    label="Rounding Currency"
                    variant="outlined"
                  />
                  <CurrencySelect
                    v-model="form.split_primary_currency"
                    hint="First half of the till's split total; the rounding currency is the second."
                    label="Split Primary Currency"
                    persistent-hint
                    variant="outlined"
                  />
                  <v-switch
                    v-model="form.split_primary_first"
                    color="primary"
                    density="compact"
                    hide-details
                    :label="form.split_primary_first
                      ? 'Primary currency shown first'
                      : 'Rounding currency shown first'"
                  />
                  <v-text-field
                    v-model.number="form.split_primary_step"
                    density="compact"
                    hide-details="auto"
                    hint="Denomination the split truncates the primary to. 0 = use that currency's own Cash Rounding Unit."
                    label="Split Step"
                    persistent-hint
                    type="number"
                    variant="outlined"
                  />
                  <v-text-field
                    v-model.number="form.exchange_rate"
                    density="compact"
                    hide-details="auto"
                    label="Exchange Rate"
                    type="number"
                    variant="outlined"
                  />
                  <v-text-field
                    v-model.number="form.exchange_window_hours"
                    density="compact"
                    hide-details="auto"
                    label="Exchange Window (hours)"
                    type="number"
                    variant="outlined"
                  />
                </div>
              </section>
            </div>

            <div v-show="sectionTab === 'behaviour'">
              <section class="dbs-section">
                <div class="dbs-section-title">Behaviour</div>
                <div class="dbs-switch-grid">
                  <v-switch
                    v-model="form.search_button_enabled"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Product search button on sale screen"
                  />
                  <v-switch
                    v-model="form.custom_item_discounts_enabled"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Custom per-line discounts"
                  />
                  <v-switch
                    v-model="form.custom_item_prices_enabled"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Custom item prices"
                  />
                  <v-switch
                    v-model="form.per_unit_item_prices_enabled"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Per-unit item pricing"
                  />
                  <v-switch
                    v-model="form.price_change_manual_override_mode"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Price-change manual override mode"
                  />
                  <v-switch
                    v-model="form.invoice_keyboard_mode"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Invoice keyboard (number-pad) mode"
                  />
                  <v-switch
                    v-model="form.client_required"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Require a client on every invoice"
                  />
                </div>
                <v-select
                  v-model="form.shift_receipt_upload_mode"
                  class="mt-3"
                  density="compact"
                  hide-details
                  :items="SHIFT_RECEIPT_UPLOAD_MODES"
                  label="Close-shift receipt photo"
                  variant="outlined"
                />
                <p class="text-caption text-medium-emphasis mt-4 mb-2">
                  "Leave as is" keeps each branch's own value when you apply to all.
                </p>
                <div class="field-grid">
                  <template v-for="f in OPTIONAL_FIELDS.behaviour" :key="f.key">
                    <v-select
                      v-if="f.type === 'bool'"
                      v-model="form[f.key]"
                      density="compact"
                      hide-details
                      :items="TRI_STATE"
                      :label="f.label"
                      variant="outlined"
                    />
                    <v-text-field
                      v-else
                      clearable
                      density="compact"
                      hide-details
                      :label="f.label"
                      min="0"
                      :model-value="scaledValue(form[f.key], f.scale)"
                      placeholder="Leave as is"
                      :step="f.step"
                      type="number"
                      variant="outlined"
                      @update:model-value="v => (form[f.key] = unscaledValue(v, f.scale))"
                    />
                  </template>
                </div>
              </section>
            </div>

            <div v-show="sectionTab === 'cash'">
              <section class="dbs-section">
                <div class="dbs-section-title">Cash &amp; shifts</div>
                <p class="text-caption text-medium-emphasis mb-2">
                  "Leave as is" keeps each branch's own value when you apply to all.
                </p>
                <div class="field-grid">
                  <template v-for="f in OPTIONAL_FIELDS.cash" :key="f.key">
                    <v-select
                      v-if="f.type === 'bool'"
                      v-model="form[f.key]"
                      density="compact"
                      hide-details
                      :items="TRI_STATE"
                      :label="f.label"
                      variant="outlined"
                    />
                    <v-text-field
                      v-else
                      clearable
                      density="compact"
                      hide-details
                      :label="f.label"
                      min="0"
                      :model-value="scaledValue(form[f.key], f.scale)"
                      placeholder="Leave as is"
                      :step="f.step"
                      type="number"
                      variant="outlined"
                      @update:model-value="v => (form[f.key] = unscaledValue(v, f.scale))"
                    />
                  </template>
                </div>
              </section>
            </div>

            <div v-show="sectionTab === 'pages'">
              <section class="dbs-section">
                <div class="dbs-section-title">Page access (non-admin users)</div>
                <div class="dbs-switch-grid">
                  <v-switch
                    v-model="form.page_unlock_clients"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Clients"
                  />
                  <v-switch
                    v-model="form.page_unlock_inventory"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Inventory"
                  />
                  <v-switch
                    v-model="form.page_unlock_transfers"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Transfers"
                  />
                  <v-switch
                    v-model="form.page_unlock_attendance"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Attendance"
                  />
                  <v-switch
                    v-model="form.page_unlock_salespersons"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Salespersons"
                  />
                </div>
                <p class="text-caption text-medium-emphasis mt-4 mb-2">
                  "Leave as is" keeps each branch's own value when you apply to all.
                </p>
                <div class="field-grid">
                  <template v-for="f in OPTIONAL_FIELDS.pages" :key="f.key">
                    <v-select
                      v-if="f.type === 'bool'"
                      v-model="form[f.key]"
                      density="compact"
                      hide-details
                      :items="TRI_STATE"
                      :label="f.label"
                      variant="outlined"
                    />
                    <v-text-field
                      v-else
                      clearable
                      density="compact"
                      hide-details
                      :label="f.label"
                      min="0"
                      :model-value="scaledValue(form[f.key], f.scale)"
                      placeholder="Leave as is"
                      :step="f.step"
                      type="number"
                      variant="outlined"
                      @update:model-value="v => (form[f.key] = unscaledValue(v, f.scale))"
                    />
                  </template>
                </div>
              </section>
            </div>

            <div v-show="sectionTab === 'printing'">
              <section class="dbs-section">
                <div class="dbs-section-title">Printing</div>
                <div class="field-grid">
                  <v-select
                    v-model="form.printer_size"
                    density="compact"
                    hide-details="auto"
                    :items="['58mm', '80mm']"
                    label="Printer Size"
                    variant="outlined"
                  />
                  <v-text-field
                    v-model.number="form.receipt_width"
                    density="compact"
                    hide-details="auto"
                    label="Receipt Width (mm)"
                    type="number"
                    variant="outlined"
                  />
                  <v-text-field
                    v-model.number="form.receipt_height"
                    density="compact"
                    hide-details="auto"
                    label="Receipt Height (mm)"
                    type="number"
                    variant="outlined"
                  />
                  <v-select
                    v-model="form.receipt_font"
                    density="compact"
                    hide-details="auto"
                    :items="['Belleza', 'Helvetica', 'Times', 'Courier']"
                    label="Title Font"
                    variant="outlined"
                  />
                  <v-select
                    v-model="form.receipt_body_font"
                    density="compact"
                    hide-details="auto"
                    :items="['Printer Font A', 'Courier', 'Helvetica', 'Times', 'Belleza']"
                    label="Body Font"
                    variant="outlined"
                  />
                  <v-text-field
                    v-model.number="form.receipt_title_size"
                    density="compact"
                    hide-details="auto"
                    label="Title Size"
                    type="number"
                    variant="outlined"
                  />
                  <v-text-field
                    v-model.number="form.receipt_body_size"
                    density="compact"
                    hide-details="auto"
                    label="Body Size"
                    type="number"
                    variant="outlined"
                  />
                </div>
                <div class="dbs-switch-grid mt-3">
                  <v-switch
                    v-model="form.receipt_enabled"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Print receipt after checkout"
                  />
                  <v-switch
                    v-model="form.receipt_cutoff"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Cut paper after printing"
                  />
                </div>
                <p class="text-caption text-medium-emphasis mt-3">
                  Physical device identifiers (printer name, receipt printer, customer-screen
                  COM port) are specific to one till and are never part of this template &mdash;
                  set those per branch from Branch Settings.
                </p>
              </section>
            </div>

            <div v-show="sectionTab === 'attendance'">
              <section class="dbs-section">
                <div class="dbs-section-title">Attendance</div>
                <div class="field-grid">
                  <v-text-field
                    v-model.number="form.attendance_duplicate_interval_seconds"
                    density="compact"
                    hide-details="auto"
                    label="Duplicate interval (sec)"
                    min="0"
                    type="number"
                    variant="outlined"
                  />
                </div>
                <div class="dbs-switch-grid mt-3">
                  <v-switch
                    v-model="form.attendance_enabled"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Attendance enabled"
                  />
                  <v-switch
                    v-model="form.attendance_cashier_history"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Cashiers can browse previous days"
                  />
                  <v-switch
                    v-model="form.attendance_manager_approval_enabled"
                    color="primary"
                    density="compact"
                    hide-details
                    label="Manager approves requests on the till (manager password)"
                  />
                </div>
                <p class="text-caption text-medium-emphasis mt-3">
                  The Akuvox device IP/username and its device password are per-machine and
                  never part of this template &mdash; set those per branch from Branch Settings.
                </p>
              </section>
            </div>

            <div v-show="sectionTab === 'display'">
              <section class="dbs-section">
                <div class="dbs-section-title">How the till's screen looks</div>
                <p class="text-caption text-medium-emphasis mb-2">
                  "Leave as is" keeps each till's own value when you apply to all.
                </p>
                <div class="field-grid">
                  <template v-for="f in DISPLAY_FIELDS" :key="f.key">
                    <v-select
                      v-if="f.items"
                      v-model="form.display_prefs[f.key]"
                      density="compact"
                      hide-details
                      :items="f.items"
                      :label="f.label"
                      variant="outlined"
                    />
                    <v-text-field
                      v-else
                      clearable
                      density="compact"
                      hide-details
                      :hint="f.hint"
                      :label="f.label"
                      :max="f.max"
                      :min="f.min"
                      :model-value="form.display_prefs[f.key]"
                      placeholder="Leave as is"
                      type="number"
                      variant="outlined"
                      @update:model-value="v => (form.display_prefs[f.key] = unscaledValue(v, 1))"
                    />
                  </template>
                  <CurrencySelect
                    v-model="form.display_prefs.calculator_total_currency"
                    clearable
                    label="Calculator shows first"
                    placeholder="Leave as is"
                    variant="outlined"
                  />
                </div>
              </section>
            </div>
          </v-form>
        </div>
      </div>
    </v-card>

    <v-dialog v-model="confirmApplyOpen" max-width="480">
      <v-card>
        <v-card-title class="text-subtitle-1">Apply to all branches?</v-card-title>
        <v-card-text>
          This sends the saved template as a remote command to every branch
          ({{ branches.length }} total). Each branch applies it on its own within a few seconds
          if online, or on its next sync otherwise &mdash; nothing here writes to a branch
          directly. Save your changes first if you've edited anything above.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="confirmApplyOpen = false" />
          <v-btn
            color="primary"
            :loading="applying"
            text="Apply to all"
            variant="flat"
            @click="applyToAll"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { onMounted, ref } from 'vue'
  import CurrencySelect from '@/components/CurrencySelect.vue'
  import { useBranches } from '@/composables/useBranches'
  import { useDefaultBranchSettings } from '@/composables/useDefaultBranchSettings'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { branches, fetchBranches } = useBranches()
  const { getDefaultBranchSettings, updateDefaultBranchSettings, applyDefaultBranchSettingsToAll } = useDefaultBranchSettings()

  const sectionTab = ref('pricing')
  const loading = ref(false)
  const saving = ref(false)
  const applying = ref(false)
  const confirmApplyOpen = ref(false)

  const SHIFT_RECEIPT_UPLOAD_MODES = [
    { title: 'Hybrid (office when online, else the till)', value: 2 },
    { title: 'Local only (the till, over shop Wi-Fi)', value: 3 },
    { title: 'Central only (straight to the office)', value: 1 },
    { title: 'Off (no photo)', value: 0 },
  ]

  // null in a select = "leave as is": the field is left out of the template.
  const TRI_STATE = [
    { title: 'Leave as is', value: null },
    { title: 'On', value: true },
    { title: 'Off', value: false },
  ]

  // Fields added after the template's first version. Each is optional, so
  // applying the template never pushes a value the admin didn't pick (see
  // defaultBranchSettings in api/default_branch_settings.go). Money is
  // stored in cents and edited in dollars (scale 100).
  const OPTIONAL_FIELDS = {
    behaviour: [
      { key: 'salesperson_per_invoice_required', label: 'Salesperson on every invoice', type: 'bool' },
      { key: 'overchange_confirmation', label: 'Confirm change larger than owed', type: 'bool' },
      { key: 'hide_payment_method_currency', label: 'Hide currency on payment buttons', type: 'bool' },
      { key: 'auto_append_lbp_zeros', label: 'Type LBP in thousands', type: 'bool' },
      { key: 'visitor_counter_enabled', label: 'Visitor counter buttons', type: 'bool' },
      { key: 'non_admin_month_nav_enabled', label: "Cashiers browse other months' income", type: 'bool' },
      { key: 'close_shift_requires_sync', label: 'Upload everything before closing a shift', type: 'bool' },
      { key: 'scan_by_code_enabled', label: 'Accept a product code in the scan field', type: 'bool' },
      { key: 'sounds_enabled', label: 'Play sounds (scan, checkout, errors, keypad)', type: 'bool' },
    ],
    cash: [
      { key: 'opening_float_enabled', label: 'Shifts start with an opening float', type: 'bool' },
      { key: 'count_opening_cash_on_close', label: 'Count the opening float when closing', type: 'bool' },
      { key: 'auto_close_stale_shifts', label: "Close yesterday's open shift at startup", type: 'bool' },
      { key: 'default_opening_float_usd', label: 'Default opening float (USD)', type: 'number', scale: 100, step: 0.01 },
      { key: 'default_opening_float_lbp', label: 'Default opening float (LBP)', type: 'number', scale: 1, step: 1000 },
      { key: 'max_overpayment_cents', label: 'Max overpayment (USD, 0 = no limit)', type: 'number', scale: 100, step: 0.01 },
    ],
    pages: [
      { key: 'page_unlock_products', label: 'Products page', type: 'bool' },
    ],
  }
  const OPTIONAL_KEYS = Object.values(OPTIONAL_FIELDS).flat().map(f => f.key)

  // Till screen prefs (kashi-pos display_prefs.go), same "leave as is" rule.
  const DISPLAY_FIELDS = [
    { key: 'quick_access_rows', label: 'Quick access rows (0 hides)', min: 0, max: 8 },
    { key: 'quick_access_font_scale', label: 'Quick access text size (%)', min: 70, max: 200 },
    { key: 'quick_access_show_code', label: 'Code on quick access cards', items: TRI_STATE },
    { key: 'quick_access_show_name', label: 'Name on quick access cards', items: TRI_STATE },
    { key: 'quick_access_show_remove', label: 'Remove (X) on quick access cards', items: TRI_STATE },
    { key: 'animations_enabled', label: 'Animations', items: TRI_STATE },
    {
      key: 'theme',
      label: 'Theme',
      items: [{ title: 'Leave as is', value: null }, { title: 'Light', value: 'light' }, { title: 'Dark', value: 'dark' }],
    },
  ]

  function scaledValue (value, scale) {
    return value == null ? '' : value / scale
  }
  function unscaledValue (value, scale) {
    if (value === '' || value == null || Number.isNaN(Number(value))) return null
    return Math.round(Number(value) * scale)
  }

  // Only the prefs actually set; an empty object means none.
  function templatePayload () {
    const prefs = Object.fromEntries(
      Object.entries(form.value.display_prefs).filter(([, v]) => v != null && v !== ''),
    )
    return { ...form.value, display_prefs: prefs }
  }

  function formFromTemplate (data) {
    return { ...blankForm(), ...data, display_prefs: { ...data?.display_prefs } }
  }

  function blankForm () {
    return {
      tax_rate: 0,
      rounding_mode: 'half_up',
      rounding_currency: '',
      exchange_rate: 0,
      exchange_window_hours: 0,
      split_primary_currency: '',
      split_primary_first: true,
      split_primary_step: 0,
      search_button_enabled: true,
      custom_item_discounts_enabled: false,
      custom_item_prices_enabled: false,
      per_unit_item_prices_enabled: false,
      price_change_manual_override_mode: false,
      invoice_keyboard_mode: false,
      client_required: false,
      shift_receipt_upload_mode: 2,
      page_unlock_clients: false,
      page_unlock_inventory: false,
      page_unlock_transfers: false,
      page_unlock_attendance: false,
      page_unlock_salespersons: false,
      printer_size: '58mm',
      receipt_width: 0,
      receipt_height: 0,
      receipt_enabled: true,
      receipt_font: 'Belleza',
      receipt_body_font: 'Printer Font A',
      receipt_title_size: 0,
      receipt_body_size: 0,
      receipt_cutoff: true,
      attendance_enabled: false,
      attendance_duplicate_interval_seconds: 0,
      attendance_cashier_history: false,
      attendance_manager_approval_enabled: true,
      ...Object.fromEntries(OPTIONAL_KEYS.map(k => [k, null])),
      display_prefs: {},
    }
  }

  const form = ref(blankForm())

  onMounted(async () => {
    loading.value = true
    try {
      await fetchBranches(true)
      const data = await getDefaultBranchSettings()
      if (data) form.value = formFromTemplate(data)
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  })

  async function save () {
    saving.value = true
    try {
      const saved = await updateDefaultBranchSettings(templatePayload())
      if (saved) form.value = formFromTemplate(saved)
      notify.success('Default settings saved.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  async function applyToAll () {
    applying.value = true
    try {
      const count = await applyDefaultBranchSettingsToAll()
      notify.success(`Queued for ${count} branch(es) — each applies it within a few seconds if online.`)
      confirmApplyOpen.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      applying.value = false
    }
  }
</script>

<style scoped>
.page-root, .dbs-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}
.dbs-detail {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  overflow: hidden;
}
.dbs-section-tabs {
  flex: 0 0 168px;
}
.dbs-section-tabs :deep(.v-tab) {
  justify-content: flex-start;
}
.dbs-section-pane {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
  padding: 20px;
}
.dbs-section {
  max-width: 720px;
  margin-bottom: 24px;
}
.dbs-section-title {
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 12px;
}
.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  align-items: start;
}
.dbs-switch-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 0 24px;
}
</style>
