<template>
  <div class="page-root">
    <v-alert
      v-if="listError"
      class="mb-4"
      density="compact"
      type="error"
      variant="tonal"
    >
      {{ listError }}
    </v-alert>

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
            </div>
            <p class="text-body-2 text-medium-emphasis mb-4">
              Saving enqueues a remote command &mdash; the branch applies it itself within a few
              seconds if it's online, or on its next sync otherwise. Nothing here writes to the
              branch database directly.
            </p>

            <v-alert
              v-if="settingsError"
              class="mb-4"
              density="compact"
              type="error"
              variant="tonal"
            >
              {{ settingsError }}
            </v-alert>
            <v-alert
              v-if="saveOutcome"
              class="mb-4"
              density="compact"
              :type="saveOutcome.ok ? 'success' : 'error'"
              variant="tonal"
            >
              {{ saveOutcome.message }}
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
                class="bs-section-tabs"
                color="primary"
                direction="vertical"
              >
                <v-tab class="text-none" text="General" value="general" />
                <v-tab class="text-none" text="Behaviour" value="behaviour" />
                <v-tab class="text-none" text="Page access" value="pages" />
                <v-tab class="text-none" text="Printing" value="printing" />
                <v-tab class="text-none" text="Attendance" value="attendance" />
                <v-tab class="text-none" text="Users" value="users" />
                <v-tab class="text-none" text="Payments" value="payments" />
                <v-tab class="text-none" value="conflicts">
                  <span class="text-none">Conflicts</span>
                  <v-badge
                    v-if="conflictCount > 0"
                    class="ms-2"
                    color="warning"
                    :content="conflictCount"
                    inline
                  />
                </v-tab>
                <v-tab class="text-none" text="Commands" value="commands" />
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
                      </div>
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
                      </div>
                    </section>

                    <section class="bs-section">
                      <div class="bs-group-head">
                        <div class="bs-section-title mb-0">Device identifiers</div>
                        <v-btn-toggle
                          v-model="deviceIdsSource"
                          color="primary"
                          density="compact"
                          divided
                          mandatory
                          @update:model-value="v => setGroupSource('device_ids', v)"
                        >
                          <v-btn size="small" value="central">Centrally</v-btn>
                          <v-btn size="small" value="local">At the branch</v-btn>
                        </v-btn-toggle>
                      </div>
                      <p class="text-caption text-medium-emphasis mb-3">
                        Windows printer name and customer-screen COM port are specific to the physical
                        till. Leave these managed at the branch unless it runs a single machine.
                      </p>
                      <div class="field-grid">
                        <v-text-field
                          v-model="settingsForm.printer_id"
                          density="compact"
                          :disabled="isLocal('device_ids')"
                          hide-details="auto"
                          label="Printer ID / name"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="settingsForm.receipt_printer"
                          density="compact"
                          :disabled="isLocal('device_ids')"
                          hide-details="auto"
                          label="Receipt printer"
                          variant="outlined"
                        />
                        <v-text-field
                          v-model="settingsForm.screen_port"
                          density="compact"
                          :disabled="isLocal('device_ids')"
                          hide-details="auto"
                          label="Customer screen COM port"
                          variant="outlined"
                        />
                      </div>
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
                    v-show="canEdit && sectionTab !== 'commands' && sectionTab !== 'users' && sectionTab !== 'conflicts'"
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
                  <v-alert
                    v-if="usersError"
                    class="mb-3"
                    density="compact"
                    type="error"
                    variant="tonal"
                  >
                    {{ usersError }}
                  </v-alert>
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

                <div v-show="sectionTab === 'payments'">
                  <div class="bs-section-title mb-0">Payment methods</div>
                  <p class="text-body-2 text-medium-emphasis mb-3">
                    Every branch inherits the central payment methods. Switch one off to stop this
                    branch offering it, or pin a position, colour or currency to differ from central
                    here only. Anything left blank keeps following central.
                  </p>
                  <v-alert
                    v-if="paymentMethodsError"
                    class="mb-3"
                    density="compact"
                    type="error"
                    variant="tonal"
                  >
                    {{ paymentMethodsError }}
                  </v-alert>
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

                <div v-show="sectionTab === 'conflicts'">
                  <div class="d-flex align-center justify-space-between mb-2">
                    <div class="bs-section-title mb-0">Sync conflicts</div>
                    <div class="d-flex align-center ga-2">
                      <v-btn
                        v-if="canEdit && selectedConflicts.length > 0"
                        color="primary"
                        size="small"
                        variant="tonal"
                        @click="bulkTakeBranch"
                      >
                        Take branch spelling ({{ selectedConflicts.length }})
                      </v-btn>
                      <v-icon-btn
                        icon="mdi-refresh"
                        size="small"
                        variant="text"
                        @click="reloadConflicts"
                      />
                    </div>
                  </div>

                  <v-alert
                    v-if="conflictOutcome"
                    class="mb-3"
                    density="compact"
                    :type="conflictOutcome.ok ? 'success' : 'error'"
                    variant="tonal"
                  >
                    {{ conflictOutcome.message }}
                  </v-alert>

                  <ServerSideTable
                    v-if="selectedId != null"
                    ref="conflictTableRef"
                    :api-u-r-l="`${apiBase}/branch_sync_conflicts`"
                    density="compact"
                    expandable
                    :headers="conflictHeaders"
                    :query-params="conflictQuery"
                    root-key="conflicts"
                    :selectable="canEdit"
                    :show-search-icon="false"
                    @update:selected="v => (selectedConflicts = v)"
                  >
                    <template #item.data-table-select="{ item, internalItem, isSelected, toggleSelect }">
                      <v-checkbox-btn
                        :disabled="item.kind !== 'phone_name_mismatch'"
                        :model-value="isSelected(internalItem)"
                        @click.stop
                        @update:model-value="toggleSelect(internalItem)"
                      />
                    </template>
                    <template #item.kind="{ item }">
                      <v-chip size="small" variant="tonal">{{ conflictKindLabel(item.kind) }}</v-chip>
                    </template>
                    <template #item.created_at="{ item }">
                      {{ new Date(item.created_at).toLocaleString() }}
                    </template>
                    <template #item.actions="{ item }">
                      <v-btn v-if="canEdit" size="small" variant="text" @click.stop="openResolve(item)">Resolve</v-btn>
                      <v-btn v-if="canEdit" size="small" variant="text" @click.stop="dismiss(item)">Dismiss</v-btn>
                    </template>
                    <template #expanded-row="{ columns, item }">
                      <tr>
                        <td :colspan="columns.length">
                          <div class="py-3 px-2">
                            <table class="conflict-diff">
                              <tr><th /><th>Branch says</th><th>kashi has</th></tr>
                              <tr>
                                <td>Name</td>
                                <td :class="diffClass(branchPayload(item).name, item.central_name)">
                                  {{ branchPayload(item).name || '—' }}
                                </td>
                                <td>{{ item.central_name || '—' }}</td>
                              </tr>
                              <tr>
                                <td>Phone</td>
                                <td>{{ branchPayload(item).phone || '—' }}</td>
                                <td>{{ item.central_phone || '—' }}</td>
                              </tr>
                              <tr v-if="item.entity === 'product'">
                                <td>Barcode</td>
                                <td colspan="2">{{ item.ref }}</td>
                              </tr>
                            </table>
                          </div>
                        </td>
                      </tr>
                    </template>
                  </ServerSideTable>
                </div>

                <div v-show="sectionTab === 'commands'">
                  <div class="d-flex align-center justify-space-between mb-2">
                    <div class="bs-section-title mb-0">Recent Commands</div>
                    <v-icon-btn
                      icon="mdi-refresh"
                      size="small"
                      variant="text"
                      @click="loadCommandHistory"
                    />
                  </div>
                  <v-table density="compact">
                    <thead>
                      <tr>
                        <th>Type</th>
                        <th>Status</th>
                        <th>Issued</th>
                        <th>Error</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="cmd in commandHistory" :key="cmd.id">
                        <td>{{ cmd.type }}</td>
                        <td>
                          <v-chip
                            :color="statusColor(cmd.status)"
                            size="small"
                          >
                            {{ cmd.status }}
                          </v-chip>
                        </td>
                        <td>{{ new Date(cmd.created_at).toLocaleString() }}</td>
                        <td>{{ cmd.error?.Valid ? cmd.error.String : '' }}</td>
                      </tr>
                      <tr v-if="commandHistory.length === 0">
                        <td class="text-medium-emphasis py-4" colspan="4">No commands issued yet.</td>
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

    <!-- resolve a sync conflict -->
    <v-dialog v-model="resolveDialog" max-width="480">
      <v-card v-if="resolveTarget">
        <v-card-title class="text-subtitle-1">
          Resolve — {{ conflictKindLabel(resolveTarget.kind) }}
        </v-card-title>
        <v-card-text>
          <div v-if="resolveTarget.kind === 'phone_name_mismatch'">
            <div class="text-caption text-medium-emphasis mb-2">
              Phone {{ branchPayload(resolveTarget).phone }} already belongs to a
              kashi client under a different name.
            </div>
            <v-radio-group v-model="resolveForm.choice" density="compact" hide-details>
              <v-radio
                :label="`Keep kashi's name (${resolveTarget.central_name})`"
                value="central"
              />
              <v-radio
                :label="`Use the branch's name (${branchPayload(resolveTarget).name})`"
                value="branch"
              />
              <v-radio label="Type a name" value="custom" />
            </v-radio-group>
            <v-text-field
              v-if="resolveForm.choice === 'custom'"
              v-model="resolveForm.name"
              class="mt-2"
              density="compact"
              label="Name"
              variant="outlined"
            />
          </div>

          <div v-else-if="resolveTarget.kind === 'invalid_phone'">
            <div class="text-caption text-medium-emphasis mb-2">
              The branch reported "{{ branchPayload(resolveTarget).name }}" with no
              usable phone.
            </div>
            <v-radio-group v-model="resolveForm.choice" density="compact" hide-details>
              <v-radio label="Merge into an existing client" value="merge" />
              <v-radio label="Create a new client with a phone" value="create" />
            </v-radio-group>
            <v-autocomplete
              v-if="resolveForm.choice === 'merge'"
              v-model="resolveForm.clientId"
              class="mt-2"
              density="compact"
              :items="clientOptions"
              label="Existing client"
              variant="outlined"
            />
            <v-text-field
              v-else
              v-model="resolveForm.phone"
              class="mt-2"
              density="compact"
              label="Phone"
              variant="outlined"
            />
          </div>

          <div v-else class="text-body-2">
            Adopt "{{ branchPayload(resolveTarget).name || resolveTarget.ref }}"
            (barcode {{ resolveTarget.ref }}) into the central catalog. Colour and
            size are created if missing; brand / season / year can be filled in
            afterwards from the product editor.
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="resolveDialog = false" />
          <v-btn
            color="primary"
            :loading="resolveSaving"
            text="Resolve"
            variant="flat"
            @click="submitResolve"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- add / edit POS user -->
    <v-dialog v-model="userDialog" max-width="440">
      <v-card>
        <v-card-title class="text-subtitle-1">{{ editingUser ? 'Edit user' : 'Add user' }}</v-card-title>
        <v-card-text>
          <v-alert
            v-if="userFormError"
            class="mb-3"
            density="compact"
            type="error"
            variant="tonal"
          >
            {{ userFormError }}
          </v-alert>
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
          <v-alert
            v-if="pinError"
            class="mb-3"
            density="compact"
            type="error"
            variant="tonal"
          >
            {{ pinError }}
          </v-alert>
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
  import CurrencySelect from '@/components/CurrencySelect.vue'
  import PageSidebar from '@/components/PageSidebar.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchPaymentMethods } from '@/composables/useBranchPaymentMethods'
  import { useBranchSettings } from '@/composables/useBranchSettings'
  import { useBranchSyncConflicts } from '@/composables/useBranchSyncConflicts'
  import { useBranchUsers } from '@/composables/useBranchUsers'
  import { useClients } from '@/composables/useClients'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const route = useRoute()
  const router = useRouter()

  const { branches, fetchBranches } = useBranches()
  const {
    getBranchSettings,
    updateBranchSettings,
    listBranchCommands,
    setBranchSettingsManagedLocally,
  } = useBranchSettings()

  const listLoading = ref(false)
  const listError = ref('')
  const selectedId = ref(null)
  // Which settings section (vertical sub-tab) is showing for the active branch.
  const sectionTab = ref('general')

  const apiBase = API_BASE
  const { resolveConflict, dismissConflict, bulkResolve, openConflictCount } = useBranchSyncConflicts()
  const conflictTableRef = ref(null)
  const conflictCount = ref(0)
  const selectedConflicts = ref([])
  const conflictOutcome = ref(null)
  const conflictHeaders = [
    { title: '', key: 'data-table-select', sortable: false, width: '44px' },
    { title: 'Type', key: 'kind', sortable: false },
    { title: 'Entity', key: 'entity', sortable: false },
    { title: 'Ref', key: 'ref', sortable: false },
    { title: 'Reported', key: 'created_at', sortable: true },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ]
  const conflictQuery = computed(() => ({
    branch_id: selectedId.value ?? 0,
    status: 'open',
    sort_by: 'created_at',
    sort_order: 'desc',
  }))

  function conflictKindLabel (kind) {
    return {
      phone_name_mismatch: 'Name differs',
      invalid_phone: 'No usable phone',
      phone_taken: 'Phone already taken',
    }[kind] ?? kind
  }
  function branchPayload (item) {
    try { return JSON.parse(item.branch_payload) } catch { return {} }
  }
  function diffClass (a, b) {
    return a && b && a !== b ? 'text-error' : ''
  }
  function reloadConflicts () {
    conflictTableRef.value?.reloadFromFirstPage?.()
    refreshConflictCount()
  }
  async function refreshConflictCount () {
    if (selectedId.value == null) return
    try { conflictCount.value = await openConflictCount(selectedId.value) } catch { /* informational */ }
  }
  // A single sync can push hundreds of conflict rows, and each one emits its
  // own branch_sync_conflict WS nudge. Coalesce a burst into one refetch --
  // fanning them straight through hammered the server (thundering herd).
  let conflictRefreshTimer = null
  function refreshConflictCountSoon () {
    if (conflictRefreshTimer) return
    conflictRefreshTimer = setTimeout(() => {
      conflictRefreshTimer = null
      refreshConflictCount()
    }, 2000)
  }
  async function dismiss (item) {
    try {
      await dismissConflict(item.id)
      conflictOutcome.value = { ok: true, message: 'Dismissed.' }
      reloadConflicts()
    } catch (error) {
      conflictOutcome.value = { ok: false, message: String(error.message || error) }
    }
  }
  async function bulkTakeBranch () {
    try {
      const res = await bulkResolve(selectedConflicts.value.map(c => c.id), 'branch')
      conflictOutcome.value = { ok: true, message: `Resolved ${res?.resolved ?? 0} conflict(s).` }
      selectedConflicts.value = []
      reloadConflicts()
    } catch (error) {
      conflictOutcome.value = { ok: false, message: String(error.message || error) }
    }
  }
  const resolveDialog = ref(false)
  const resolveTarget = ref(null)
  const resolveSaving = ref(false)
  const resolveForm = reactive({ choice: 'branch', name: '', phone: '', clientId: null })
  const { clients, fetchClients } = useClients()
  const clientOptions = computed(() =>
    clients.value.map(c => ({ title: `${c.name} — ${c.phone}`, value: c.id })),
  )

  function openResolve (item) {
    resolveTarget.value = item
    const payload = branchPayload(item)
    if (item.kind === 'phone_name_mismatch') {
      resolveForm.choice = 'branch'
      resolveForm.name = payload.name || ''
    } else if (item.kind === 'invalid_phone') {
      resolveForm.choice = 'create'
      resolveForm.phone = ''
      resolveForm.clientId = null
      fetchClients()
    } else {
      resolveForm.choice = 'create'
    }
    resolveDialog.value = true
  }

  async function submitResolve () {
    const item = resolveTarget.value
    if (!item) {
      return
    }
    const body = { choice: resolveForm.choice }
    if (item.kind === 'phone_name_mismatch' && resolveForm.choice === 'custom') {
      body.name = resolveForm.name.trim()
    }
    if (item.kind === 'invalid_phone') {
      if (resolveForm.choice === 'merge') {
        body.client_id = resolveForm.clientId
      } else {
        body.phone = resolveForm.phone.trim()
      }
    }
    resolveSaving.value = true
    try {
      await resolveConflict(item.id, body)
      conflictOutcome.value = { ok: true, message: 'Resolved.' }
      resolveDialog.value = false
      reloadConflicts()
    } catch (error) {
      conflictOutcome.value = { ok: false, message: String(error.message || error) }
    } finally {
      resolveSaving.value = false
    }
  }

  let conflictPoll = null
  let offAdminMessage = null
  const { onMessage: onAdminMessage, ensureConnected: ensureAdminSocket } = useAdminSocket()
  onMounted(() => {
    ensureAdminSocket?.()
    conflictPoll = setInterval(refreshConflictCount, 30_000)
    // onMessage registers into a module-level Set and returns an unsubscribe;
    // without keeping it, every visit to this page stacks another live
    // handler that never goes away.
    offAdminMessage = onAdminMessage?.((msg) => {
      if (msg?.type === 'branch_sync_conflict') refreshConflictCountSoon()
    })
  })
  onUnmounted(() => {
    if (conflictPoll) clearInterval(conflictPoll)
    if (conflictRefreshTimer) clearTimeout(conflictRefreshTimer)
    offAdminMessage?.()
  })
  watch(selectedId, () => { refreshConflictCount() })

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

  function statusColor (status) {
    if (status === 'success') return 'success'
    if (status === 'failed') return 'error'
    return 'warning'
  }

  onMounted(async () => {
    listLoading.value = true
    listError.value = ''
    try {
      await fetchBranches(true)
      const wanted = Number(route.query.branch)
      selectedId.value = branches.value.find(b => b.id === wanted)?.id
        ?? branches.value[0]?.id
        ?? null
    } catch (error) {
      listError.value = error.message
    } finally {
      listLoading.value = false
    }
  })

  // ---- per-branch settings ------------------------------------------------
  const settingsForm = ref(null)
  const secretForm = ref({ akuvox_password: '', attendance_approver_password: '', admin_password: '' })
  const settingsLoading = ref(false)
  const settingsError = ref('')
  const saving = ref(false)
  const saveOutcome = ref(null)
  const socialPlatformsText = ref('[]')
  const socialHandlesText = ref('{}')
  const commandHistory = ref([])

  // Which settings groups the branch owns locally. Persisted centrally the
  // moment a toggle changes -- it's admin state, not a branch command.
  const managedLocally = ref([])
  const isLocal = group => managedLocally.value.includes(group)
  const deviceIdsSource = computed(() => (isLocal('device_ids') ? 'local' : 'central'))
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
    'attendance_enabled',
    'attendance_cashier_history',
    'split_primary_first',
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
    }
    for (const k of BOOL_KEYS) {
      if (k === 'search_button_enabled') continue
      // Like search_button_enabled: absent on a till that predates the field,
      // and the primary currency leading is the shape every branch had when
      // it was hardcoded -- so only an explicit false flips it.
      if (k === 'split_primary_first') {
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
    settingsError.value = ''
    saveOutcome.value = null
    sectionTab.value = 'general'
    settingsForm.value = null
    secretForm.value = { akuvox_password: '', attendance_approver_password: '', admin_password: '' }
    commandHistory.value = []
    try {
      const data = await getBranchSettings(id)
      if (data) {
        settingsForm.value = formFromSettings(data)
        managedLocally.value = Array.isArray(data.managed_locally) ? [...data.managed_locally] : []
        socialPlatformsText.value = JSON.stringify(data.social_platforms ?? [])
        socialHandlesText.value = JSON.stringify(data.social_handles ?? {})
      }
    } catch (error) {
      settingsError.value = error.message
    } finally {
      settingsLoading.value = false
    }
    loadCommandHistory()
    loadBranchUsers(id)
    loadBranchPaymentMethods(id)
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
  const paymentMethodsError = ref('')

  async function loadBranchPaymentMethods (id) {
    branchPaymentMethods.value = []
    paymentMethodsError.value = ''
    try {
      branchPaymentMethods.value = await listBranchPaymentMethods(id)
    } catch (error) {
      paymentMethodsError.value = error.message
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
    paymentMethodsError.value = ''
    try {
      await setBranchPaymentMethod(selectedId.value, m.id, {
        isEnabled: m.is_enabled,
        sortOrder: m.override_sort_order,
        color: m.override_color,
        currencyCode: m.override_currency_code,
      })
    } catch (error) {
      paymentMethodsError.value = error.message
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
  const usersError = ref('')

  async function loadBranchUsers (id) {
    branchUsers.value = []
    usersError.value = ''
    try {
      branchUsers.value = await listBranchUsers(id)
    } catch (error) {
      usersError.value = error.message
    }
  }

  const userDialog = ref(false)
  const editingUser = ref(null)
  const userSaving = ref(false)
  const userFormError = ref('')
  const userForm = ref({ username: '', role: 'cashier', akuvox_user_id: '', pin: '', is_active: true })

  function openUserDialog (u = null) {
    editingUser.value = u
    userFormError.value = ''
    userForm.value = u
      ? { username: u.username, role: u.role, akuvox_user_id: u.akuvox_user_id, pin: '', is_active: u.is_active }
      : { username: '', role: 'cashier', akuvox_user_id: '', pin: '', is_active: true }
    userDialog.value = true
  }

  async function submitUser () {
    const f = userForm.value
    if (!f.username.trim()) {
      userFormError.value = 'Username is required.'
      return
    }
    if (!editingUser.value && !/^\d{4,6}$/.test(f.pin)) {
      userFormError.value = 'Initial PIN must be 4 to 6 digits.'
      return
    }
    userSaving.value = true
    userFormError.value = ''
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
      userFormError.value = error.message
    } finally {
      userSaving.value = false
    }
  }

  async function removeUser (u) {
    usersError.value = ''
    try {
      await deleteBranchUser(u.id)
      await loadBranchUsers(selectedId.value)
    } catch (error) {
      usersError.value = error.message
    }
  }

  const pinDialog = ref(false)
  const pinTarget = ref(null)
  const pinValue = ref('')
  const pinSaving = ref(false)
  const pinError = ref('')

  function openPinDialog (u) {
    pinTarget.value = u
    pinValue.value = ''
    pinError.value = ''
    pinDialog.value = true
  }

  async function submitPin () {
    if (!/^\d{4,6}$/.test(pinValue.value)) {
      pinError.value = 'PIN must be 4 to 6 digits.'
      return
    }
    pinSaving.value = true
    pinError.value = ''
    try {
      await setBranchUserPin(pinTarget.value.id, pinValue.value)
      pinDialog.value = false
      await loadCommandHistory()
    } catch (error) {
      pinError.value = error.message
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
      settingsError.value = error.message
    }
  }

  async function loadCommandHistory () {
    if (selectedId.value == null) return
    try {
      commandHistory.value = await listBranchCommands(selectedId.value)
    } catch (error) {
      settingsError.value = error.message
    }
  }

  async function saveSettings () {
    saving.value = true
    saveOutcome.value = null
    settingsError.value = ''
    try {
      let socialPlatforms
      let socialHandles
      try {
        socialPlatforms = JSON.parse(socialPlatformsText.value)
        socialHandles = JSON.parse(socialHandlesText.value)
      } catch {
        throw new Error('Social Platforms/Handles must be valid JSON')
      }

      const payload = {
        ...settingsForm.value,
        social_platforms: socialPlatforms,
        social_handles: socialHandles,
      }

      // Groups handed back to the branch are never pushed.
      if (isLocal('device_ids')) {
        for (const k of GROUP_KEYS.device_ids) delete payload[k]
      }
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

      await updateBranchSettings(selectedId.value, payload)
      saveOutcome.value = { ok: true, message: 'Command sent -- the branch will apply it within a few seconds if online.' }
      secretForm.value = { akuvox_password: '', attendance_approver_password: '', admin_password: '' }
      await loadCommandHistory()
    } catch (error) {
      saveOutcome.value = { ok: false, message: error.message }
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
