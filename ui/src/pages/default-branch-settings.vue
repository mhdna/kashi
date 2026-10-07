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
          <v-tab class="text-none" text="Page access" value="pages" />
          <v-tab class="text-none" text="Printing" value="printing" />
          <v-tab class="text-none" text="Attendance" value="attendance" />
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
                </div>
                <p class="text-caption text-medium-emphasis mt-3">
                  The Akuvox device IP/username and its device password are per-machine and
                  never part of this template &mdash; set those per branch from Branch Settings.
                </p>
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
    }
  }

  const form = ref(blankForm())

  onMounted(async () => {
    loading.value = true
    try {
      await fetchBranches(true)
      const data = await getDefaultBranchSettings()
      if (data) form.value = { ...blankForm(), ...data }
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  })

  async function save () {
    saving.value = true
    try {
      const saved = await updateDefaultBranchSettings(form.value)
      if (saved) form.value = { ...blankForm(), ...saved }
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
