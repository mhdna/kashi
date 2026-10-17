<template>
  <div class="page-root">
    <v-card class="bs-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-store-cog-outline" />
        <span>Branch Settings</span>
        <v-spacer />
        <v-btn
          :disabled="!activeBranch"
          prepend-icon="mdi-refresh"
          text="Reload"
          variant="text"
          @click="reloadActive"
        />
      </v-card-title>
      <v-divider />

      <div class="bs-layout">
        <PageSidebar
          v-model="selectedId"
          empty-text="No branches yet."
          :items="branchItems"
          :loading="listLoading"
        >
          <template #prepend="{ item }">
            <span class="bs-dot me-2" :class="onlineClass(item)" />
          </template>
        </PageSidebar>

        <v-divider vertical />

        <div class="bs-content">
          <!-- ------------------------------------------------- no selection -->
          <div v-if="!activeBranch" class="bs-placeholder">
            <v-icon icon="mdi-store-cog-outline" size="40" />
            <div>Pick a branch to view its settings.</div>
          </div>

          <!-- --------------------------------------------------- a branch -->
          <div v-else class="bs-detail-root pa-5">
            <div class="d-flex align-center ga-2 mb-1">
              <h3 class="text-subtitle-1">{{ activeBranch.name }}</h3>
              <v-chip size="x-small" variant="tonal">{{ activeBranch.code }}</v-chip>
              <span class="text-caption text-medium-emphasis">
                Last seen {{ lastSeenLabel(activeBranch) }}
              </span>
              <v-spacer />
              <v-btn
                v-if="canEdit"
                :loading="syncing"
                prepend-icon="mdi-sync"
                size="small"
                text="Sync now"
                variant="text"
                @click="syncNow"
              />
              <v-btn
                prepend-icon="mdi-console-line"
                size="small"
                text="Commands"
                :to="`/branch-commands?branch=${activeBranch.id}`"
                variant="text"
              />
            </div>
            <p class="text-body-2 text-medium-emphasis mb-4">
              Saving enqueues a remote command &mdash; the branch applies it itself within a few
              seconds if it's online, or on its next sync otherwise. Nothing here writes to the
              branch database directly.
            </p>

            <v-alert
              v-if="!settingsLoading && settingsForm && drift.length > 0"
              class="bs-empty-alert mb-4"
              density="compact"
              type="warning"
              variant="tonal"
            >
              <div class="d-flex flex-wrap align-center ga-2">
                <span>
                  {{ drift.length }} setting{{ drift.length === 1 ? '' : 's' }} on the till
                  differ{{ drift.length === 1 ? 's' : '' }} from kashi:
                  {{ drift.slice(0, 6).map(d => `${d.label} (${d.registers.join(', ')})`).join(', ') }}{{ drift.length > 6 ? '…' : '' }}.
                  Saving sends kashi's values; or take the till's values into the form first.
                </span>
                <v-spacer />
                <v-btn size="small" text="Use till values" variant="text" @click="adoptTillValues" />
              </div>
            </v-alert>

            <div v-if="settingsLoading" class="d-flex justify-center pa-8">
              <v-progress-circular color="primary" indeterminate />
            </div>

            <v-alert
              v-else-if="!settingsForm"
              class="bs-empty-alert"
              type="info"
              variant="tonal"
            >
              This branch hasn't pushed its settings yet &mdash; it needs to sync at least once
              before you can view or edit them.
            </v-alert>

            <div v-else class="bs-detail">
              <v-tabs
                v-model="sectionTab"
                class="bs-section-tabs rail-tabs"
                color="primary"
                :direction="$vuetify.display.smAndDown ? 'horizontal' : 'vertical'"
              >
                <v-tab class="text-none" text="General" value="general" />
                <v-tab class="text-none" text="Behaviour" value="behaviour" />
                <v-tab class="text-none" text="Cash &amp; shifts" value="cash" />
                <v-tab class="text-none" text="Page access" value="pages" />
                <v-tab class="text-none" text="Printing" value="printing" />
                <v-tab class="text-none" text="Attendance" value="attendance" />
                <v-tab class="text-none" text="Users" value="users" />
                <v-tab class="text-none" text="Payments" value="payments" />
                <v-tab class="text-none" text="Quick access" value="quick_access" />
                <v-tab class="text-none" text="Till display" value="display" />
              </v-tabs>

              <v-divider vertical />

              <div class="bs-section-pane">
                <v-form :disabled="!canEdit" @submit.prevent="saveSettings">
                  <div v-show="sectionTab === 'general'">
                    <section class="bs-section">
                      <div class="bs-section-title">General</div>
                      <div class="field-grid">
                        <v-text-field
                          v-model="settingsForm.branch_name"
                          density="compact"
                          hide-details="auto"
                          label="Branch Name"
                          variant="outlined"
                        />
                      </div>
                    </section>

                    <section class="bs-section">
                      <div class="bs-section-title">Pricing &amp; Exchange</div>
                      <div class="field-grid">
                        <v-text-field
                          v-model.number="settingsForm.tax_rate"
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
                          v-model="settingsForm.rounding_mode"
                          density="compact"
                          hide-details="auto"
                          :items="['half_up', 'up', 'down']"
                          label="Rounding Mode"
                          variant="outlined"
                        />
                        <CurrencySelect
                          v-model="settingsForm.rounding_currency"
                          label="Rounding Currency"
                          variant="outlined"
                        />
                        <CurrencySelect
                          v-model="settingsForm.split_primary_currency"
                          hint="First half of the till's split total; the rounding currency is the second."
                          label="Split Primary Currency"
                          persistent-hint
                          variant="outlined"
                        />
                        <v-switch
                          v-model="settingsForm.split_primary_first"
                          color="primary"
                          density="compact"
                          hide-details
                          :label="settingsForm.split_primary_first
                            ? 'Primary currency shown first'
                            : 'Rounding currency shown first'"
                        />
                        <v-text-field
                          v-model.number="settingsForm.split_primary_step"
                          density="compact"
                          hide-details="auto"
                          hint="Denomination the split truncates the primary to. 0 = use that currency's own Cash Rounding Unit."
                          label="Split Step"
                          persistent-hint
                          type="number"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.exchange_rate"
                          density="compact"
                          hide-details="auto"
                          label="Exchange Rate"
                          type="number"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.exchange_window_hours"
                          density="compact"
                          hide-details="auto"
                          label="Exchange Window (hours)"
                          type="number"
                          variant="outlined"
                        />
                      </div>
                    </section>

                    <section class="bs-section">
                      <div class="bs-section-title">Storefront</div>
                      <div class="field-grid">
                        <v-text-field
                          v-model="settingsForm.market_name"
                          density="compact"
                          hide-details="auto"
                          label="Market Name"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="settingsForm.market_phone"
                          density="compact"
                          hide-details="auto"
                          label="Market Phone"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="settingsForm.branch_phone"
                          density="compact"
                          hide-details="auto"
                          label="Branch Phone"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="settingsForm.website"
                          density="compact"
                          hide-details="auto"
                          label="Website"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="settingsForm.instagram"
                          density="compact"
                          hide-details="auto"
                          label="Instagram"
                          variant="outlined"
                        />
                        <v-textarea
                          v-model="settingsForm.market_description"
                          class="field-full"
                          density="compact"
                          hide-details="auto"
                          label="Market Description"
                          rows="2"
                          variant="outlined"
                        />
                        <v-textarea
                          v-model="settingsForm.return_policy"
                          class="field-full"
                          density="compact"
                          hide-details="auto"
                          label="Return Policy"
                          rows="2"
                          variant="outlined"
                        />
                      </div>
                    </section>

                    <section class="bs-section">
                      <div class="bs-section-title">Social</div>
                      <div class="field-grid">
                        <v-textarea
                          v-model="socialPlatformsText"
                          class="field-full"
                          density="compact"
                          hint="JSON array, e.g. [&quot;facebook&quot;, &quot;instagram&quot;]"
                          label="Social Platforms"
                          persistent-hint
                          rows="2"
                          variant="outlined"
                        />
                        <v-textarea
                          v-model="socialHandlesText"
                          class="field-full"
                          density="compact"
                          hint="JSON object, e.g. {&quot;instagram&quot;: &quot;@handle&quot;}"
                          label="Social Handles"
                          persistent-hint
                          rows="2"
                          variant="outlined"
                        />
                      </div>
                    </section>

                  </div>

                  <div v-show="sectionTab === 'behaviour'">
                    <section class="bs-section">
                      <div class="bs-section-title">Behaviour</div>
                      <div class="bs-switch-grid">
                        <v-switch
                          v-model="settingsForm.search_button_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Product search button on sale screen"
                        />
                        <v-switch
                          v-model="settingsForm.scan_by_code_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Accept a product code in the scan field"
                        />
                        <v-switch
                          v-model="settingsForm.sounds_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Play sounds (scan, checkout, errors, keypad)"
                        />
                        <v-switch
                          v-model="settingsForm.custom_item_discounts_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Custom per-line discounts"
                        />
                        <v-switch
                          v-model="settingsForm.custom_item_prices_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Custom item prices"
                        />
                        <v-switch
                          v-model="settingsForm.per_unit_item_prices_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Per-unit item pricing"
                        />
                        <v-switch
                          v-model="settingsForm.price_change_manual_override_mode"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Price-change manual override mode"
                        />
                        <v-switch
                          v-model="settingsForm.invoice_keyboard_mode"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Invoice keyboard (number-pad) mode"
                        />
                        <v-switch
                          v-model="settingsForm.client_required"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Require a client on every invoice"
                        />
                        <v-switch
                          v-model="settingsForm.salesperson_per_invoice_required"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Require a salesperson on every invoice"
                        />
                        <v-switch
                          v-model="settingsForm.overchange_confirmation"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Confirm change larger than what's owed"
                        />
                        <v-switch
                          v-model="settingsForm.hide_payment_method_currency"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Hide currency on payment method buttons"
                        />
                        <v-switch
                          v-model="settingsForm.auto_append_lbp_zeros"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Type LBP in thousands (50 → 50,000)"
                        />
                        <v-switch
                          v-model="settingsForm.visitor_counter_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Visitor counter buttons"
                        />
                        <v-switch
                          v-model="settingsForm.non_admin_month_nav_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Cashiers can browse other months' income"
                        />
                      </div>
                      <v-select
                        v-model="settingsForm.oversell_mode"
                        class="mt-3"
                        density="compact"
                        hide-details
                        :items="OVERSELL_MODES"
                        label="Selling more than the till has in stock"
                        variant="outlined"
                      />
                      <div class="text-caption text-medium-emphasis mt-1">
                        Whatever the mode, kashi raises an "oversold" alert on the Conflicts page
                        when a sale takes a product's stock below zero.
                      </div>
                    </section>

                  </div>

                  <div v-show="sectionTab === 'cash'">
                    <section class="bs-section">
                      <div class="bs-section-title">Opening cash</div>
                      <div class="bs-switch-grid">
                        <v-switch
                          v-model="settingsForm.opening_float_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Shifts start with an opening float"
                        />
                        <v-switch
                          v-model="settingsForm.count_opening_cash_on_close"
                          color="primary"
                          density="compact"
                          :disabled="!settingsForm.opening_float_enabled"
                          hide-details
                          label="Count the opening float when closing"
                        />
                      </div>
                      <div class="field-grid mt-3">
                        <v-text-field
                          v-model.number="openingFloatUsd"
                          density="compact"
                          :disabled="!settingsForm.opening_float_enabled"
                          hide-details="auto"
                          label="Default opening float (USD)"
                          min="0"
                          step="0.01"
                          type="number"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.default_opening_float_lbp"
                          density="compact"
                          :disabled="!settingsForm.opening_float_enabled"
                          hide-details="auto"
                          label="Default opening float (LBP)"
                          min="0"
                          step="1000"
                          type="number"
                          variant="outlined"
                        />
                      </div>
                    </section>

                    <section class="bs-section">
                      <div class="bs-section-title">Payments</div>
                      <div class="field-grid">
                        <v-text-field
                          v-model.number="maxOverpayment"
                          density="compact"
                          hint="How much a payment may go over the total, in USD. 0 = no limit."
                          label="Max overpayment (USD)"
                          min="0"
                          persistent-hint
                          step="0.01"
                          type="number"
                          variant="outlined"
                        />
                      </div>
                    </section>

                    <section class="bs-section">
                      <div class="bs-section-title">Closing shifts</div>
                      <v-switch
                        v-model="settingsForm.auto_close_stale_shifts"
                        color="primary"
                        density="compact"
                        hide-details
                        label="Close yesterday's open shift automatically at startup"
                      />
                      <v-select
                        v-model="settingsForm.shift_receipt_upload_mode"
                        class="mt-3"
                        density="compact"
                        hide-details
                        :items="SHIFT_RECEIPT_UPLOAD_MODES"
                        label="Close-shift receipt photo"
                        variant="outlined"
                      />
                      <div class="text-caption text-medium-emphasis mt-1">
                        Hybrid uploads to the office when the branch is online and buffers on
                        the till otherwise; Local always uploads to the till over shop Wi-Fi;
                        Central always goes straight to the office; Off skips the photo.
                      </div>
                      <v-switch
                        v-model="settingsForm.close_shift_requires_sync"
                        class="mt-3"
                        color="primary"
                        density="compact"
                        hide-details
                        label="Upload everything before closing a shift"
                      />
                      <div class="text-caption text-medium-emphasis mt-1">
                        The till uploads every sale, return and expense still waiting for the
                        office before it closes the shift and prints the close receipt, showing
                        progress. Offline, a manager can override on the till.
                      </div>
                    </section>
                  </div>

                  <div v-show="sectionTab === 'display'">
                    <section class="bs-section">
                      <div class="bs-section-title">How the till's screen looks</div>
                      <p class="text-caption text-medium-emphasis mb-3">
                        Save sends these to every register. To change one till only, pick it below
                        and send. A cashier can still change them on the till afterwards; the latest
                        change wins.
                      </p>
                      <div class="text-body-2">
                        Quick access rows:
                        <strong>{{ displayForm.quick_access_rows === 0 ? 'Hidden' : displayForm.quick_access_rows }}</strong>
                      </div>
                      <v-slider
                        v-model="displayForm.quick_access_rows"
                        color="primary"
                        :max="8"
                        :min="0"
                        hint="Rows of cards on the sale screen before it scrolls. 0 hides the panel."
                        persistent-hint
                        show-ticks="always"
                        :step="1"
                      />
                      <div class="text-body-2 mt-4">
                        Quick access text size: <strong>{{ displayForm.quick_access_font_scale }}%</strong>
                      </div>
                      <v-slider
                        v-model="displayForm.quick_access_font_scale"
                        color="primary"
                        hide-details
                        :max="200"
                        :min="70"
                        show-ticks="always"
                        :step="10"
                      />
                      <div class="bs-switch-grid mt-3">
                        <v-switch
                          v-model="displayForm.quick_access_show_code"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Show code on quick access cards"
                        />
                        <v-switch
                          v-model="displayForm.quick_access_show_name"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Show name on quick access cards"
                        />
                        <v-switch
                          v-model="displayForm.quick_access_show_remove"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Show remove (X) on quick access cards"
                        />
                        <v-switch
                          v-model="displayForm.animations_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Animations"
                        />
                      </div>
                      <div class="field-grid mt-3">
                        <v-select
                          v-model="displayForm.theme"
                          density="compact"
                          hide-details
                          :items="[{ title: 'Light', value: 'light' }, { title: 'Dark', value: 'dark' }]"
                          label="Theme"
                          variant="outlined"
                        />
                        <CurrencySelect
                          v-model="displayForm.calculator_total_currency"
                          label="Calculator shows first"
                          variant="outlined"
                        />
                      </div>
                    </section>

                    <section class="bs-section">
                      <div class="bs-section-title">One register only</div>
                      <div class="d-flex flex-wrap align-center ga-3">
                        <v-select
                          v-model="displayRegisterId"
                          density="compact"
                          hide-details
                          item-title="label"
                          item-value="id"
                          :items="registerOptions"
                          label="Register"
                          no-data-text="This branch has no registers"
                          style="max-width: 280px"
                          variant="outlined"
                        />
                        <v-btn
                          color="primary"
                          :disabled="!canEdit || displayRegisterId == null"
                          :loading="displaySending"
                          prepend-icon="mdi-send-outline"
                          :text="`Send to ${registerCodeById(displayRegisterId) || 'register'}`"
                          variant="tonal"
                          @click="sendDisplayToRegister"
                        />
                      </div>
                    </section>
                  </div>

                  <div v-show="sectionTab === 'pages'">
                    <section class="bs-section">
                      <div class="bs-section-title">Page access (non-admin users)</div>
                      <div class="bs-switch-grid">
                        <v-switch
                          v-model="settingsForm.page_unlock_clients"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Clients"
                        />
                        <v-switch
                          v-model="settingsForm.page_unlock_inventory"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Inventory"
                        />
                        <v-switch
                          v-model="settingsForm.page_unlock_products"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Products"
                        />
                        <v-switch
                          v-model="settingsForm.page_unlock_transfers"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Transfers"
                        />
                        <v-switch
                          v-model="settingsForm.page_unlock_attendance"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Attendance"
                        />
                        <v-switch
                          v-model="settingsForm.page_unlock_salespersons"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Salespersons"
                        />
                      </div>
                    </section>

                  </div>

                  <div v-show="sectionTab === 'printing'">
                    <section class="bs-section">
                      <div class="bs-section-title">Printing</div>
                      <div class="field-grid">
                        <v-select
                          v-model="settingsForm.printer_size"
                          density="compact"
                          hide-details="auto"
                          :items="['58mm', '80mm']"
                          label="Printer Size"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.receipt_width"
                          density="compact"
                          hide-details="auto"
                          label="Receipt Width (mm)"
                          type="number"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.receipt_height"
                          density="compact"
                          hide-details="auto"
                          label="Receipt Height (mm)"
                          type="number"
                          variant="outlined"
                        />
                        <v-select
                          v-model="settingsForm.receipt_font"
                          density="compact"
                          hide-details="auto"
                          :items="['Belleza', 'Helvetica', 'Times', 'Courier']"
                          label="Title Font"
                          variant="outlined"
                        />
                        <v-select
                          v-model="settingsForm.receipt_body_font"
                          density="compact"
                          hide-details="auto"
                          :items="['Printer Font A', 'Courier', 'Helvetica', 'Times', 'Belleza']"
                          label="Body Font"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.receipt_title_size"
                          density="compact"
                          hide-details="auto"
                          label="Title Size"
                          type="number"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.receipt_body_size"
                          density="compact"
                          hide-details="auto"
                          label="Body Size"
                          type="number"
                          variant="outlined"
                        />
                      </div>
                      <div class="bs-switch-grid mt-3">
                        <v-switch
                          v-model="settingsForm.receipt_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Print receipt after checkout"
                        />
                        <v-switch
                          v-model="settingsForm.receipt_cutoff"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Cut paper after printing"
                        />
                        <v-switch
                          v-model="settingsForm.shift_report_signatures"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Cashier and manager signature lines on end-of-shift receipt"
                        />
                        <v-switch
                          v-model="settingsForm.shift_report_uppercase_header"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Store and branch name in capitals on end-of-shift receipt"
                        />
                        <v-switch
                          v-model="settingsForm.auto_cover_negative_lbp"
                          color="primary"
                          density="compact"
                          hide-details
                          label="At close shift, cover a negative Cash LL balance from Cash $$"
                        />
                      </div>
                    </section>

                    <!-- Printers and the customer screen belong to one physical till, so
                         they're edited per register and never sent branch-wide. -->
                    <section class="bs-section">
                      <div class="bs-section-title">Device identifiers (per register)</div>
                      <p class="text-caption text-medium-emphasis mb-3">
                        Windows printer name and customer-screen COM port are specific to each till.
                        Pick a register to see what it reports and send it new values.
                      </p>
                      <v-select
                        v-model="deviceRegisterId"
                        class="mb-3"
                        density="compact"
                        hide-details
                        item-title="label"
                        item-value="id"
                        :items="registerOptions"
                        label="Register"
                        no-data-text="This branch has no registers"
                        variant="outlined"
                      />
                      <template v-if="deviceRegisterId != null">
                        <div class="field-grid">
                          <v-text-field
                            v-model="deviceForm.printer_id"
                            density="compact"
                            hide-details="auto"
                            label="Printer ID / name"
                            variant="outlined"
                          />
                          <v-text-field
                            v-model="deviceForm.receipt_printer"
                            density="compact"
                            hide-details="auto"
                            label="Receipt printer"
                            variant="outlined"
                          />
                          <v-text-field
                            v-model="deviceForm.screen_port"
                            density="compact"
                            hide-details="auto"
                            label="Customer screen COM port"
                            variant="outlined"
                          />
                        </div>
                        <div class="d-flex justify-end mt-3">
                          <v-btn
                            color="primary"
                            :disabled="!canEdit"
                            :loading="deviceSaving"
                            prepend-icon="mdi-send-outline"
                            :text="`Send to ${selectedRegister?.code ?? 'register'}`"
                            variant="tonal"
                            @click="saveDeviceSettings"
                          />
                        </div>
                      </template>
                    </section>

                  </div>

                  <div v-show="sectionTab === 'attendance'">
                    <section class="bs-section">
                      <div class="bs-section-title">Attendance</div>
                      <div class="field-grid">
                        <v-text-field
                          v-model="settingsForm.akuvox_ip"
                          density="compact"
                          hide-details="auto"
                          label="Akuvox IP"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="settingsForm.akuvox_username"
                          density="compact"
                          hide-details="auto"
                          label="Akuvox Username"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model.number="settingsForm.attendance_duplicate_interval_seconds"
                          density="compact"
                          hide-details="auto"
                          label="Duplicate interval (sec)"
                          min="0"
                          type="number"
                          variant="outlined"
                        />
                      </div>
                      <div class="bs-switch-grid mt-3">
                        <v-switch
                          v-model="settingsForm.attendance_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Attendance enabled"
                        />
                        <v-switch
                          v-model="settingsForm.attendance_cashier_history"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Cashiers can browse previous days"
                        />
                        <v-switch
                          v-model="settingsForm.attendance_manager_approval_enabled"
                          color="primary"
                          density="compact"
                          hide-details
                          label="Manager approves requests on the till (manager password)"
                        />
                      </div>

                      <div class="bs-group-head mt-4">
                        <div class="bs-section-title mb-0">Secrets</div>
                        <v-btn-toggle
                          v-model="secretsSource"
                          color="primary"
                          density="compact"
                          divided
                          mandatory
                          @update:model-value="v => setGroupSource('secrets', v)"
                        >
                          <v-btn size="small" value="central">Centrally</v-btn>
                          <v-btn size="small" value="local">At the branch</v-btn>
                        </v-btn-toggle>
                      </div>
                      <p class="text-caption text-medium-emphasis mb-3">
                        Write-only &mdash; kashi never displays these back. Leave blank to keep the
                        branch's current value; type a new one to push it.
                      </p>
                      <div class="field-grid">
                        <v-text-field
                          v-model="secretForm.akuvox_password"
                          autocomplete="new-password"
                          density="compact"
                          :disabled="isLocal('secrets')"
                          hide-details="auto"
                          label="Akuvox device password"
                          type="password"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="secretForm.attendance_approver_password"
                          autocomplete="new-password"
                          density="compact"
                          :disabled="isLocal('secrets')"
                          hide-details="auto"
                          label="Attendance manager password"
                          type="password"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="secretForm.admin_password"
                          autocomplete="new-password"
                          density="compact"
                          :disabled="isLocal('secrets')"
                          hint="Resets the branch's POS &quot;admin&quot; login"
                          label="POS admin password"
                          persistent-hint
                          type="password"
                          variant="outlined"
                        />
                      </div>
                    </section>
                  </div>

                  <div
                    v-show="canEdit && FORM_TABS.includes(sectionTab)"
                    class="bs-savebar d-flex justify-end"
                  >
                    <v-btn
                      color="primary"
                      :loading="saving"
                      text="Save"
                      type="submit"
                      variant="flat"
                    />
                  </div>
                </v-form>

                <div v-show="sectionTab === 'users'">
                  <div class="d-flex align-center justify-space-between mb-2">
                    <div class="bs-section-title mb-0">POS Users</div>
                    <v-btn
                      v-if="canEdit"
                      color="primary"
                      prepend-icon="mdi-plus"
                      size="small"
                      text="Add user"
                      variant="tonal"
                      @click="openUserDialog()"
                    />
                  </div>
                  <p class="text-body-2 text-medium-emphasis mb-3">
                    Roster is managed here and synced down to the till read-only. Adding a user or
                    changing a PIN sends a command the branch applies on its next sync.
                  </p>
                  <v-table density="compact">
                    <thead>
                      <tr>
                        <th>Username</th>
                        <th>Role</th>
                        <th>Akuvox ID</th>
                        <th>Active</th>
                        <th class="text-end" />
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="u in branchUsers" :key="u.id">
                        <td>{{ u.username }}</td>
                        <td>{{ u.role }}</td>
                        <td>{{ u.akuvox_user_id || '—' }}</td>
                        <td>
                          <v-chip :color="u.is_active ? 'success' : 'grey'" size="x-small" variant="tonal">
                            {{ u.is_active ? 'Active' : 'Inactive' }}
                          </v-chip>
                        </td>
                        <td class="text-end">
                          <v-icon-btn
                            v-if="canEdit"
                            icon="mdi-key"
                            size="small"
                            title="Set PIN"
                            variant="text"
                            @click="openPinDialog(u)"
                          />
                          <v-icon-btn
                            v-if="canEdit"
                            icon="mdi-pencil"
                            size="small"
                            variant="text"
                            @click="openUserDialog(u)"
                          />
                          <v-icon-btn
                            v-if="canEdit"
                            color="error"
                            icon="mdi-delete"
                            size="small"
                            variant="text"
                            @click="removeUser(u)"
                          />
                        </td>
                      </tr>
                      <tr v-if="branchUsers.length === 0">
                        <td class="text-medium-emphasis py-4" colspan="5">No POS users yet.</td>
                      </tr>
                    </tbody>
                  </v-table>
                </div>

                <div v-if="sectionTab === 'quick_access'">
                  <BranchQuickAccess :branch-id="selectedId" :can-edit="canEdit" />
                </div>

                <div v-show="sectionTab === 'payments'">
                  <div class="bs-section-title mb-0">Payment methods</div>
                  <p class="text-body-2 text-medium-emphasis mb-3">
                    Every branch inherits the central payment methods. Switch one off to stop this
                    branch offering it, or pin a position, colour or currency to differ from central
                    here only. Anything left blank keeps following central.
                  </p>
                  <v-table density="compact">
                    <thead>
                      <tr>
                        <th>Method</th>
                        <th>Offered</th>
                        <th>Order</th>
                        <th>Currency</th>
                        <th>Colour</th>
                        <th class="text-end" />
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="m in branchPaymentMethods" :key="m.id">
                        <td>{{ m.name }}</td>
                        <td>
                          <v-switch
                            color="primary"
                            density="compact"
                            :disabled="!canEdit"
                            hide-details
                            :model-value="m.is_enabled"
                            @update:model-value="setMethodEnabled(m, $event)"
                          />
                        </td>
                        <td style="width: 120px">
                          <v-text-field
                            density="compact"
                            :disabled="!canEdit"
                            hide-details
                            :model-value="m.override_sort_order"
                            :placeholder="String(m.sort_order)"
                            type="number"
                            variant="outlined"
                            @blur="saveMethod(m)"
                            @update:model-value="setMethodOverride(m, 'sortOrder', $event)"
                          />
                        </td>
                        <td style="width: 160px">
                          <CurrencySelect
                            clearable
                            :disabled="!canEdit"
                            hide-details
                            label=""
                            :model-value="m.override_currency_code || ''"
                            :placeholder="m.currency_code"
                            @update:model-value="setMethodOverride(m, 'currencyCode', $event)"
                          />
                        </td>
                        <td style="width: 150px">
                          <v-text-field
                            density="compact"
                            :disabled="!canEdit"
                            hide-details
                            :model-value="m.override_color"
                            :placeholder="m.color || '—'"
                            variant="outlined"
                            @blur="saveMethod(m)"
                            @update:model-value="setMethodOverride(m, 'color', $event)"
                          />
                        </td>
                        <td class="text-end">
                          <v-chip
                            v-if="methodIsOverridden(m)"
                            size="x-small"
                            variant="tonal"
                          >
                            Overridden
                          </v-chip>
                          <span v-else class="text-medium-emphasis text-caption">Inherited</span>
                          <v-icon-btn
                            v-if="canEdit && methodIsOverridden(m)"
                            icon="mdi-restore"
                            size="small"
                            title="Back to central"
                            variant="text"
                            @click="resetMethod(m)"
                          />
                        </td>
                      </tr>
                      <tr v-if="branchPaymentMethods.length === 0">
                        <td class="text-medium-emphasis py-4" colspan="6">
                          No payment methods defined centrally yet.
                        </td>
                      </tr>
                    </tbody>
                  </v-table>
                </div>

              </div>
            </div>
          </div>
        </div>
      </div>
    </v-card>

    <!-- add / edit POS user -->
    <v-dialog v-model="userDialog" max-width="440">
      <v-card>
        <v-card-title class="text-subtitle-1">{{ editingUser ? 'Edit user' : 'Add user' }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="userForm.username"
            density="compact"
            label="Username"
            variant="outlined"
          />
          <v-select
            v-model="userForm.role"
            class="mt-3"
            density="compact"
            :items="['cashier', 'admin']"
            label="Role"
            variant="outlined"
          />
          <v-text-field
            v-model="userForm.akuvox_user_id"
            class="mt-3"
            density="compact"
            hide-details="auto"
            label="Akuvox UserID (optional)"
            variant="outlined"
          />
          <v-text-field
            v-if="!editingUser"
            v-model="userForm.pin"
            class="mt-3"
            density="compact"
            hint="4 to 6 digits"
            inputmode="numeric"
            label="Initial PIN"
            maxlength="6"
            persistent-hint
            variant="outlined"
          />
          <v-switch
            v-model="userForm.is_active"
            color="primary"
            hide-details
            label="Active"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="userDialog = false" />
          <v-btn
            color="primary"
            :loading="userSaving"
            text="Save"
            variant="flat"
            @click="submitUser"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- set PIN -->
    <v-dialog v-model="pinDialog" max-width="380">
      <v-card>
        <v-card-title class="text-subtitle-1">Set PIN — {{ pinTarget?.username }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="pinValue"
            autofocus
            density="compact"
            hint="4 to 6 digits — the branch applies it on its next sync"
            inputmode="numeric"
            label="New PIN"
            maxlength="6"
            persistent-hint
            variant="outlined"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="pinDialog = false" />
          <v-btn
            color="primary"
            :loading="pinSaving"
            text="Queue PIN change"
            variant="flat"
            @click="submitPin"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import BranchQuickAccess from '@/components/BranchQuickAccess.vue'
  import CurrencySelect from '@/components/CurrencySelect.vue'
  import PageSidebar from '@/components/PageSidebar.vue'
  import { useBranchCommands } from '@/composables/useBranchCommands'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchPaymentMethods } from '@/composables/useBranchPaymentMethods'
  import { useBranchSettings } from '@/composables/useBranchSettings'
  import { useBranchUsers } from '@/composables/useBranchUsers'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'

  const notify = useNotify()

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const route = useRoute()
  const router = useRouter()

  const { branches, fetchBranches, listRegisters } = useBranches()
  const {
    getBranchSettings,
    updateBranchSettings,
    setBranchSettingsManagedLocally,
  } = useBranchSettings()
  const { sendCommand } = useBranchCommands()

  const listLoading = ref(false)
  const selectedId = ref(null)
  // Which settings section (vertical sub-tab) is showing for the active branch.
  const sectionTab = ref('general')


  const activeBranch = computed(() => branches.value.find(b => b.id === selectedId.value) ?? null)

  const branchItems = computed(() => branches.value.map(b => ({
    value: b.id,
    title: b.name,
    subtitle: b.code,
    raw: b,
  })))

  function onlineClass (branch) {
    const t = branch.last_seen_at?.Valid ? new Date(branch.last_seen_at.Time).getTime() : 0
    return Date.now() - t < 2 * 60 * 1000 ? 'is-online' : 'is-offline'
  }

  function lastSeenLabel (branch) {
    return branch.last_seen_at?.Valid
      ? new Date(branch.last_seen_at.Time).toLocaleString()
      : 'never'
  }

  const syncing = ref(false)
  async function syncNow () {
    syncing.value = true
    try {
      const sent = await sendCommand(selectedId.value, 'sync_now')
      notify.success(`Asked ${sent.length} register${sent.length === 1 ? '' : 's'} to sync now.`)
    } catch (error) {
      notify.error(error.message)
    } finally {
      syncing.value = false
    }
  }

  onMounted(async () => {
    listLoading.value = true
    try {
      await fetchBranches(true)
      const wanted = Number(route.query.branch)
      selectedId.value = branches.value.find(b => b.id === wanted)?.id
        ?? branches.value[0]?.id
        ?? null
    } catch (error) {
      notify.error(error.message)
    } finally {
      listLoading.value = false
    }
  })

  // ---- per-branch settings ------------------------------------------------
  const settingsForm = ref(null)
  const secretForm = ref({ akuvox_password: '', attendance_approver_password: '', admin_password: '' })
  const settingsLoading = ref(false)
  const saving = ref(false)
  const socialPlatformsText = ref('[]')
  const socialHandlesText = ref('{}')

  // Tabs whose fields belong to the settings form (and so show Save).
  const FORM_TABS = ['general', 'behaviour', 'cash', 'pages', 'printing', 'attendance', 'display']

  // The form as it was loaded (or last saved). Save sends only the fields
  // that differ from it: the rest stay exactly as the tills have them, so a
  // value kashi hasn't caught up with yet is never pushed back over the
  // till's real one.
  const loadedForm = ref(null)

  // The till screen's look, sent as one display_prefs object. Defaults are
  // the till's own defaults (kashi-pos lib/display.ts).
  const DISPLAY_DEFAULTS = {
    quick_access_rows: 3,
    quick_access_font_scale: 100,
    quick_access_show_code: true,
    quick_access_show_name: true,
    quick_access_show_remove: true,
    animations_enabled: true,
    calculator_total_currency: 'USD',
    theme: 'light',
  }
  const displayForm = ref({ ...DISPLAY_DEFAULTS })
  const displayRegisterId = ref(null)
  const displaySending = ref(false)

  async function sendDisplayToRegister () {
    displaySending.value = true
    try {
      await updateBranchSettings(selectedId.value, { display_prefs: { ...displayForm.value } }, displayRegisterId.value)
      notify.success(`Sent to ${registerCodeById(displayRegisterId.value)} — it applies within a few seconds if online.`)
    } catch (error) {
      notify.error(error.message)
    } finally {
      displaySending.value = false
    }
  }

  // Money fields are stored in cents; the form edits dollars.
  function centsField (key) {
    return computed({
      get: () => (settingsForm.value?.[key] ?? 0) / 100,
      set: v => {
        settingsForm.value[key] = Math.round((Number(v) || 0) * 100)
      },
    })
  }
  const openingFloatUsd = centsField('default_opening_float_usd')
  const maxOverpayment = centsField('max_overpayment_cents')

  // Which settings groups the branch owns locally. Persisted centrally the
  // moment a toggle changes -- it's admin state, not a branch command.
  const managedLocally = ref([])
  const isLocal = group => managedLocally.value.includes(group)
  const secretsSource = computed(() => (isLocal('secrets') ? 'local' : 'central'))

  // Keys that make up each toggleable group, for stripping the command payload.
  const GROUP_KEYS = {
    device_ids: ['printer_id', 'receipt_printer', 'screen_port'],
    secrets: ['akuvox_password', 'attendance_approver_password', 'admin_password'],
  }

  const SHIFT_RECEIPT_UPLOAD_MODES = [
    { title: 'Hybrid (office when online, else the till)', value: 2 },
    { title: 'Local only (the till, over shop Wi-Fi)', value: 3 },
    { title: 'Central only (straight to the office)', value: 1 },
    { title: 'Off (no photo)', value: 0 },
  ]

  const OVERSELL_MODES = [
    { title: 'Allow — sell, fix stock in kashi later', value: 'allow' },
    { title: 'Warn — cashier confirms before selling', value: 'warn' },
    { title: 'Block — refuse the sale', value: 'block' },
  ]

  const BOOL_KEYS = [
    'search_button_enabled',
    'custom_item_discounts_enabled',
    'custom_item_prices_enabled',
    'per_unit_item_prices_enabled',
    'price_change_manual_override_mode',
    'invoice_keyboard_mode',
    'client_required',
    'page_unlock_clients',
    'page_unlock_inventory',
    'page_unlock_transfers',
    'page_unlock_attendance',
    'page_unlock_salespersons',
    'receipt_enabled',
    'receipt_cutoff',
    'shift_report_signatures',
    'shift_report_uppercase_header',
    'auto_cover_negative_lbp',
    'attendance_enabled',
    'attendance_cashier_history',
    'attendance_manager_approval_enabled',
    'split_primary_first',
    'close_shift_requires_sync',
    'scan_by_code_enabled',
    'sounds_enabled',
    'salesperson_per_invoice_required',
    'non_admin_month_nav_enabled',
    'opening_float_enabled',
    'count_opening_cash_on_close',
    'overchange_confirmation',
    'page_unlock_products',
    'hide_payment_method_currency',
    'auto_append_lbp_zeros',
    'auto_close_stale_shifts',
    'visitor_counter_enabled',
  ]

  function formFromSettings (d) {
    const form = {
      branch_name: d.branch_name,
      tax_rate: d.tax_rate,
      rounding_mode: d.rounding_mode,
      rounding_currency: d.rounding_currency,
      exchange_rate: d.exchange_rate,
      exchange_window_hours: d.exchange_window_hours,
      market_name: d.market_name,
      market_phone: d.market_phone,
      market_description: d.market_description,
      return_policy: d.return_policy,
      website: d.website,
      instagram: d.instagram,
      // Absent on branches that haven't pushed since this field was added --
      // treat only an explicit false as "off".
      search_button_enabled: d.search_button_enabled !== false,
      printer_size: d.printer_size ?? '',
      receipt_width: d.receipt_width ?? 0,
      receipt_height: d.receipt_height ?? 0,
      receipt_font: d.receipt_font ?? '',
      receipt_body_font: d.receipt_body_font ?? '',
      receipt_title_size: d.receipt_title_size ?? 0,
      receipt_body_size: d.receipt_body_size ?? 0,
      printer_id: d.printer_id ?? '',
      receipt_printer: d.receipt_printer ?? '',
      screen_port: d.screen_port ?? '',
      akuvox_ip: d.akuvox_ip ?? '',
      akuvox_username: d.akuvox_username ?? '',
      attendance_duplicate_interval_seconds: d.attendance_duplicate_interval_seconds ?? 0,
      // Not reported back up by the branch yet -- defaults to hybrid; sending
      // the command sets it on the till.
      shift_receipt_upload_mode: d.shift_receipt_upload_mode ?? 2,
      split_primary_currency: d.split_primary_currency ?? '',
      split_primary_step: d.split_primary_step ?? 0,
      oversell_mode: d.oversell_mode ?? 'allow',
      branch_phone: d.branch_phone ?? '',
      max_overpayment_cents: d.max_overpayment_cents ?? 0,
      default_opening_float_usd: d.default_opening_float_usd ?? 0,
      default_opening_float_lbp: d.default_opening_float_lbp ?? 0,
    }
    for (const k of BOOL_KEYS) {
      if (k === 'search_button_enabled') continue
      // Like search_button_enabled: absent on a till that predates the field,
      // and on is what every branch did before it existed (the primary
      // currency leading; codes accepted in the scan field) -- so only an
      // explicit false flips it.
      if (k === 'split_primary_first' || k === 'scan_by_code_enabled' || k === 'attendance_manager_approval_enabled' || k === 'shift_report_signatures') {
        form[k] = d[k] !== false
        continue
      }
      form[k] = d[k] === true
    }
    return form
  }

  watch(selectedId, id => {
    if (id == null) return
    // Keep the URL shareable / refresh-stable without stacking history entries.
    router.replace({ query: { ...route.query, branch: String(id) } })
    loadSettings(id)
  })

  async function loadSettings (id) {
    settingsLoading.value = true
    sectionTab.value = 'general'
    settingsForm.value = null
    secretForm.value = { akuvox_password: '', attendance_approver_password: '', admin_password: '' }
    loadedForm.value = null
    try {
      const data = await getBranchSettings(id)
      if (data) {
        settingsForm.value = formFromSettings(data)
        displayForm.value = { ...DISPLAY_DEFAULTS, ...data.display_prefs }
        managedLocally.value = Array.isArray(data.managed_locally) ? [...data.managed_locally] : []
        socialPlatformsText.value = JSON.stringify(data.social_platforms ?? [])
        socialHandlesText.value = JSON.stringify(data.social_handles ?? {})
      }
    } catch (error) {
      notify.error(error.message)
    } finally {
      settingsLoading.value = false
    }
    snapshotLoaded()
    loadBranchUsers(id)
    loadBranchPaymentMethods(id)
    loadBranchRegisters(id)
  }

  // ---- per-register device identifiers -----------------------------------
  const branchRegisters = ref([])
  const deviceRegisterId = ref(null)
  const deviceForm = ref({ printer_id: '', receipt_printer: '', screen_port: '' })
  const deviceSaving = ref(false)
  const registerOptions = computed(() => branchRegisters.value.map(r => ({
    id: r.id,
    label: r.name ? `${r.code} — ${r.name}` : r.code,
  })))
  const selectedRegister = computed(() => branchRegisters.value.find(r => r.id === deviceRegisterId.value) ?? null)
  function registerCodeById (id) {
    return branchRegisters.value.find(r => r.id === id)?.code ?? ''
  }

  async function loadBranchRegisters (id) {
    branchRegisters.value = []
    deviceRegisterId.value = null
    try {
      branchRegisters.value = await listRegisters(id)
      deviceRegisterId.value = branchRegisters.value[0]?.id ?? null
      displayRegisterId.value = branchRegisters.value[0]?.id ?? null
    } catch (error) {
      notify.error(error.message)
    }
  }

  // What the till last reported is the best starting point for an edit.
  watch(selectedRegister, register => {
    const reported = register?.reported_settings ?? {}
    deviceForm.value = {
      printer_id: reported.printer_id ?? '',
      receipt_printer: reported.receipt_printer ?? '',
      screen_port: reported.screen_port ?? '',
    }
  })

  async function saveDeviceSettings () {
    deviceSaving.value = true
    try {
      await updateBranchSettings(selectedId.value, { ...deviceForm.value }, deviceRegisterId.value)
      notify.success(`Command sent to ${selectedRegister.value?.code} — it applies within a few seconds if online.`)
    } catch (error) {
      notify.error(error.message)
    } finally {
      deviceSaving.value = false
    }
  }

  function reloadActive () {
    if (selectedId.value != null) loadSettings(selectedId.value)
  }

  // ---- payment method overrides -----------------------------------------
  // The rows carry both central's value and this branch's override, so an
  // edit only has to send the override side. An empty string from a cleared
  // field means "stop overriding", which is null on the wire -- the server
  // drops the row entirely once nothing is pinned and the method is offered.
  const { listBranchPaymentMethods, setBranchPaymentMethod } = useBranchPaymentMethods()
  const branchPaymentMethods = ref([])

  async function loadBranchPaymentMethods (id) {
    branchPaymentMethods.value = []
    try {
      branchPaymentMethods.value = await listBranchPaymentMethods(id)
    } catch (error) {
      notify.error(error.message)
    }
  }

  function methodIsOverridden (m) {
    return !m.is_enabled
      || m.override_sort_order != null
      || m.override_color != null
      || m.override_currency_code != null
  }

  // Every write sends the method's whole override state, because the server
  // replaces the row rather than patching it -- sending one field alone would
  // clear the others.
  async function saveMethod (m) {
    try {
      await setBranchPaymentMethod(selectedId.value, m.id, {
        isEnabled: m.is_enabled,
        sortOrder: m.override_sort_order,
        color: m.override_color,
        currencyCode: m.override_currency_code,
      })
    } catch (error) {
      notify.error(error.message)
    }
    await loadBranchPaymentMethods(selectedId.value)
  }

  function setMethodEnabled (m, value) {
    m.is_enabled = !!value
    saveMethod(m)
  }

  // Records the edit without saving. The free-text fields save on blur, so a
  // half-typed colour is not sent (and not echoed back mid-keystroke by the
  // reload); the currency picker commits immediately, since a selection is a
  // complete value the moment it is made.
  function setMethodOverride (m, field, value) {
    const blank = value === '' || value == null
    if (field === 'sortOrder') {
      m.override_sort_order = blank ? null : Number(value)
    } else if (field === 'color') {
      m.override_color = blank ? null : value
    } else {
      m.override_currency_code = blank ? null : value
      saveMethod(m)
    }
  }

  function resetMethod (m) {
    m.is_enabled = true
    m.override_sort_order = null
    m.override_color = null
    m.override_currency_code = null
    saveMethod(m)
  }

  // ---- POS users --------------------------------------------------------
  const { listBranchUsers, createBranchUser, updateBranchUser, setBranchUserPin, deleteBranchUser } = useBranchUsers()
  const branchUsers = ref([])

  async function loadBranchUsers (id) {
    branchUsers.value = []
    try {
      branchUsers.value = await listBranchUsers(id)
    } catch (error) {
      notify.error(error.message)
    }
  }

  const userDialog = ref(false)
  const editingUser = ref(null)
  const userSaving = ref(false)
  const userForm = ref({ username: '', role: 'cashier', akuvox_user_id: '', pin: '', is_active: true })

  function openUserDialog (u = null) {
    editingUser.value = u
    userForm.value = u
      ? { username: u.username, role: u.role, akuvox_user_id: u.akuvox_user_id, pin: '', is_active: u.is_active }
      : { username: '', role: 'cashier', akuvox_user_id: '', pin: '', is_active: true }
    userDialog.value = true
  }

  async function submitUser () {
    const f = userForm.value
    if (!f.username.trim()) {
      notify.error('Username is required.')
      return
    }
    if (!editingUser.value && !/^\d{4,6}$/.test(f.pin)) {
      notify.error('Initial PIN must be 4 to 6 digits.')
      return
    }
    userSaving.value = true
    try {
      await (editingUser.value ? updateBranchUser(editingUser.value.id, {
        username: f.username.trim(),
        role: f.role,
        akuvox_user_id: f.akuvox_user_id,
        is_active: f.is_active,
      }) : createBranchUser(selectedId.value, {
        username: f.username.trim(),
        role: f.role,
        akuvox_user_id: f.akuvox_user_id,
        is_active: f.is_active,
        pin: f.pin,
      }));
      userDialog.value = false
      await loadBranchUsers(selectedId.value)
    } catch (error) {
      notify.error(error.message)
    } finally {
      userSaving.value = false
    }
  }

  async function removeUser (u) {
    try {
      await deleteBranchUser(u.id)
      await loadBranchUsers(selectedId.value)
    } catch (error) {
      notify.error(error.message)
    }
  }

  const pinDialog = ref(false)
  const pinTarget = ref(null)
  const pinValue = ref('')
  const pinSaving = ref(false)

  function openPinDialog (u) {
    pinTarget.value = u
    pinValue.value = ''
    pinDialog.value = true
  }

  async function submitPin () {
    if (!/^\d{4,6}$/.test(pinValue.value)) {
      notify.error('PIN must be 4 to 6 digits.')
      return
    }
    pinSaving.value = true
    try {
      await setBranchUserPin(pinTarget.value.id, pinValue.value)
      pinDialog.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      pinSaving.value = false
    }
  }

  async function setGroupSource (group, source) {
    const local = source === 'local'
    const next = managedLocally.value.filter(g => g !== group)
    if (local) next.push(group)
    const prev = managedLocally.value
    managedLocally.value = next
    try {
      await setBranchSettingsManagedLocally(selectedId.value, next)
    } catch (error) {
      managedLocally.value = prev
      notify.error(error.message)
    }
  }

  // What the form holds as the payload it would send, for diffing.
  function formPayload () {
    let socialPlatforms
    let socialHandles
    try {
      socialPlatforms = JSON.parse(socialPlatformsText.value)
      socialHandles = JSON.parse(socialHandlesText.value)
    } catch {
      throw new Error('Social Platforms/Handles must be valid JSON')
    }
    return {
      ...settingsForm.value,
      social_platforms: socialPlatforms,
      social_handles: socialHandles,
      display_prefs: { ...displayForm.value },
    }
  }

  function snapshotLoaded () {
    if (!settingsForm.value) return
    try {
      loadedForm.value = structuredClone(formPayload())
    } catch {
      loadedForm.value = null
    }
  }

  // ---- drift: what the tills report vs what kashi holds --------------------
  // Labels for the settings worth comparing (per-machine fields excluded).
  const DRIFT_LABELS = {
    tax_rate: 'Tax rate',
    rounding_mode: 'Rounding mode',
    rounding_currency: 'Rounding currency',
    exchange_rate: 'Exchange rate',
    exchange_window_hours: 'Exchange window',
    search_button_enabled: 'Search button',
    scan_by_code_enabled: 'Code in scan field',
    sounds_enabled: 'Sounds',
    attendance_manager_approval_enabled: 'Manager approval on till',
    custom_item_discounts_enabled: 'Custom discounts',
    custom_item_prices_enabled: 'Custom prices',
    per_unit_item_prices_enabled: 'Per-unit pricing',
    invoice_keyboard_mode: 'Keyboard mode',
    client_required: 'Client required',
    salesperson_per_invoice_required: 'Salesperson required',
    non_admin_month_nav_enabled: 'Month browsing',
    opening_float_enabled: 'Opening float',
    count_opening_cash_on_close: 'Count opening cash',
    overchange_confirmation: 'Overchange confirmation',
    close_shift_requires_sync: 'Upload before close',
    hide_payment_method_currency: 'Hide payment currency',
    auto_append_lbp_zeros: 'LBP thousands',
    auto_close_stale_shifts: 'Auto-close shifts',
    visitor_counter_enabled: 'Visitor counter',
    shift_receipt_upload_mode: 'Receipt photo',
    oversell_mode: 'Oversell mode',
    max_overpayment_cents: 'Max overpayment',
    default_opening_float_usd: 'Opening float USD',
    default_opening_float_lbp: 'Opening float LBP',
    page_unlock_clients: 'Clients page',
    page_unlock_inventory: 'Inventory page',
    page_unlock_products: 'Products page',
    page_unlock_transfers: 'Transfers page',
    page_unlock_attendance: 'Attendance page',
    page_unlock_salespersons: 'Salespersons page',
    receipt_enabled: 'Print receipt',
    receipt_cutoff: 'Cut paper',
    shift_report_signatures: 'Shift receipt signatures',
    shift_report_uppercase_header: 'Shift receipt capitals',
    auto_cover_negative_lbp: 'Cover negative LBP at close',
    printer_size: 'Printer size',
    branch_phone: 'Branch phone',
  }

  // Compared against what was loaded, not the live form, so editing a field
  // doesn't make the warning come and go.
  const drift = computed(() => {
    if (!loadedForm.value) return []
    const out = []
    for (const [key, label] of Object.entries(DRIFT_LABELS)) {
      const registers = branchRegisters.value
        .filter(r => r.reported_settings && key in r.reported_settings
          && JSON.stringify(r.reported_settings[key]) !== JSON.stringify(loadedForm.value[key]))
        .map(r => r.code)
      if (registers.length > 0) out.push({ key, label, registers })
    }
    return out
  })

  // Takes the first differing register's value for each drifted field into
  // the form; nothing is sent until Save.
  function adoptTillValues () {
    for (const { key } of drift.value) {
      const register = branchRegisters.value.find(r => r.reported_settings && key in r.reported_settings)
      settingsForm.value[key] = register.reported_settings[key]
    }
    notify.info('Till values copied into the form. Save to keep them.')
  }

  async function saveSettings () {
    saving.value = true
    try {
      const full = formPayload()
      const payload = {}
      for (const [key, value] of Object.entries(full)) {
        if (JSON.stringify(value) !== JSON.stringify(loadedForm.value?.[key])) payload[key] = value
      }
      // A drifted field is one the till holds differently from kashi; send
      // kashi's value even if the admin didn't touch it, as the banner says.
      for (const { key } of drift.value) {
        if (key in full) payload[key] = full[key]
      }

      // Per-register device fields are sent from their own section.
      for (const k of GROUP_KEYS.device_ids) delete payload[k]
      if (!isLocal('secrets')) {
        // Write-only: only send a secret the admin actually typed.
        if (secretForm.value.akuvox_password) {
          payload.akuvox_password = secretForm.value.akuvox_password
        }
        if (secretForm.value.attendance_approver_password) {
          payload.attendance_approver_password = secretForm.value.attendance_approver_password
        }
        if (secretForm.value.admin_password) {
          payload.admin_password = secretForm.value.admin_password
        }
      }

      if (Object.keys(payload).length === 0) {
        notify.success('Nothing changed.')
        return
      }
      await updateBranchSettings(selectedId.value, payload)
      notify.success('Saved — every register of this branch applies it within a few seconds if online.')
      secretForm.value = { akuvox_password: '', attendance_approver_password: '', admin_password: '' }
      loadedForm.value = structuredClone(full)
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }
</script>

<style scoped>
/* Keep intrinsic height inside the layout's flex-column scroll wrapper so the
   scroll lives inside the content pane, not the whole page. */
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bs-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bs-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.bs-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
/* activeBranch wrapper: header stays put, the section pane scrolls */
.bs-detail-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bs-detail {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
}
/* v-alert defaults to flex: 1 1; keep the empty-state notice at its natural height
   instead of stretching to fill the detail pane. */
.bs-empty-alert {
  flex: 0 0 auto;
}
.bs-section-tabs {
  flex: 0 0 168px;
}
.bs-section-tabs :deep(.v-tab) {
  justify-content: flex-start;
}
.bs-section-pane {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
  padding-left: 20px;
}
.bs-savebar {
  position: sticky;
  bottom: 0;
  padding: 12px 0 4px;
  margin-top: 8px;
  background: rgb(var(--v-theme-surface));
}
.bs-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  flex-shrink: 0;
}
.bs-dot.is-online {
  background: rgb(var(--v-theme-success));
}
.bs-dot.is-offline {
  background: rgba(128, 128, 128, 0.5);
}
.bs-placeholder {
  font-size: 0.875rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: 100%;
  color: rgba(var(--v-theme-on-surface), 0.55);
  text-align: center;
}
.bs-section {
  max-width: 720px;
  margin-bottom: 24px;
}
.bs-section-title {
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 12px;
}
.bs-group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;
}
.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  align-items: start;
}
.field-grid > .field-full {
  grid-column: 1 / -1;
}
.bs-switch-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 0 24px;
}
</style>
