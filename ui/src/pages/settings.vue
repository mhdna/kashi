<template>
  <div class="page-root">
    <v-card class="set-card" flat>
      <v-card-title class="page-heading d-flex align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-cog-outline" />
        <span>Settings</span>
      </v-card-title>
      <v-divider />

      <div class="set-layout">
        <v-tabs
          v-model="tab"
          class="set-tabs rail-tabs"
          color="primary"
          :direction="$vuetify.display.smAndDown ? 'horizontal' : 'vertical'"
        >
          <v-tab prepend-icon="mdi-cog-outline" text="General" value="general" />
          <v-tab prepend-icon="mdi-chart-line" text="Financials" value="financials" />
          <v-tab prepend-icon="mdi-barcode" text="Barcode labels" value="barcodes" />
          <v-tab prepend-icon="mdi-package-variant" text="Products" value="products" />
          <v-tab v-if="auth.isAdmin" prepend-icon="mdi-account-group" text="Users" value="users" />
          <v-tab v-if="auth.isAdmin" prepend-icon="mdi-shield-account-outline" text="Roles" value="roles" />
          <v-tab v-if="auth.canViewPage('system', 'settings')" prepend-icon="mdi-whatsapp" text="WhatsApp" value="whatsapp" />
          <v-tab v-if="auth.canViewPage('system', 'settings')" prepend-icon="mdi-star-outline" text="Reviews" value="reviews" />
          <v-tab v-if="auth.canViewPage('system', 'settings')" prepend-icon="mdi-email-outline" text="Email" value="email" />
        </v-tabs>

        <v-divider vertical />

        <div class="set-content pa-5">
          <!-- ------------------------------------------------------- GENERAL -->
          <section v-show="tab === 'general'" class="set-section">
            <v-form :disabled="!canEditSystem">
              <h3 class="text-subtitle-1 mb-1">Branch activity</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                How live branch sales / returns / expenses are shown in the appbar.
              </p>

              <div class="text-body-2 mb-2">Display mode</div>
              <v-btn-toggle
                v-model="draft.activityDisplayMode"
                class="mb-4"
                color="primary"
                divided
                mandatory
              >
                <v-btn value="notification">Notification</v-btn>
                <v-btn value="appbar">Appbar messages</v-btn>
              </v-btn-toggle>

              <v-text-field
                v-model.number="draft.activityMessageSeconds"
                class="field-md"
                density="compact"
                hint="How long each message is shown before moving to the next one queued behind it"
                label="Message duration (seconds)"
                min="1"
                persistent-hint
                type="number"
                variant="outlined"
              />

              <h3 class="text-subtitle-1 mb-1 mt-6">Upcoming events</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                The appbar &ldquo;Upcoming event&rdquo; card scans the retail calendar this
                far ahead.
              </p>

              <v-text-field
                v-model.number="draft.upcomingEventsDays"
                class="field-md"
                density="compact"
                hint="Days ahead the inline card text looks for the next event"
                label="Card look-ahead (days)"
                max="365"
                min="1"
                persistent-hint
                type="number"
                variant="outlined"
              />

              <v-text-field
                v-model.number="draft.upcomingEventsMenuDays"
                class="field-md mt-4"
                density="compact"
                hint="Days ahead the dropdown lists events (default 180 ≈ 6 months)"
                label="Dropdown look-ahead (days)"
                max="730"
                min="1"
                persistent-hint
                type="number"
                variant="outlined"
              />

            </v-form>

            <h3 class="text-subtitle-1 mb-1 mt-6">Your avatar</h3>
            <p class="text-body-2 text-medium-emphasis mb-3">
              The initials badge in the top bar — your name's first two letters.
              Everyone sees you in this colour, on every device.
            </p>

            <div class="d-flex align-center ga-4 mb-2">
              <v-avatar class="text-white font-weight-medium" :color="avatarColor" size="48">
                <span class="text-subtitle-1">{{ avatarInitials }}</span>
              </v-avatar>
              <div class="d-flex flex-wrap ga-2">
                <button
                  v-for="c in avatarColors"
                  :key="c"
                  class="swatch"
                  :class="{ 'swatch--active': c === avatarColor }"
                  :style="{ background: c }"
                  type="button"
                  @click="pickAvatarColor(c)"
                />
              </div>
            </div>
            <v-btn
              v-if="avatarIsCustom"
              class="px-1"
              size="small"
              variant="text"
              @click="pickAvatarColor('')"
            >
              Reset to default colour
            </v-btn>

            <h3 class="text-subtitle-1 mt-8 mb-1">Phone upload pages</h3>
            <p class="text-body-2 text-medium-emphasis mb-2">
              The pages a phone opens from a QR code (receipts, transfers, unknown items) are always light.
              Turn this on to let them follow the phone's dark mode.
            </p>
            <v-switch
              v-model="draft.uploadPagesFollowSystem"
              color="primary"
              density="compact"
              hide-details
              inset
              label="Follow the phone's dark mode"
            />
          </section>

          <!-- ---------------------------------------------------- FINANCIALS -->
          <section v-show="tab === 'financials'" class="set-section">
            <v-form :disabled="!canEditSystem">
              <h3 class="text-subtitle-1 mb-1">Revenue chart</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                Dashboard &ldquo;Financials&rdquo; tab revenue candles.
              </p>

              <v-select
                v-model="draft.financialsHighSeasonMonths"
                chips
                class="field-md"
                density="compact"
                hint="Months that earn ~1.5x a normal month, shaded on the revenue candles chart"
                :items="monthItems"
                label="High season months"
                multiple
                persistent-hint
                variant="outlined"
              />
            </v-form>
          </section>

          <!-- ------------------------------------------------------ BARCODES -->
          <section v-show="tab === 'barcodes'" class="set-section">
            <v-form :disabled="!canEditSystem">
              <h3 class="text-subtitle-1 mb-1">Barcode labels</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                Default page size for generated barcode-label PDFs.
              </p>

              <div class="d-flex ga-3 field-md">
                <v-text-field
                  v-model.number="draft.barcodeLabelWidth"
                  density="compact"
                  hide-details
                  label="Width (pt)"
                  type="number"
                  variant="outlined"
                />
                <v-text-field
                  v-model.number="draft.barcodeLabelHeight"
                  density="compact"
                  hide-details
                  label="Height (pt)"
                  type="number"
                  variant="outlined"
                />
              </div>
              <div class="text-caption text-medium-emphasis mt-2">
                72 pt = 1 inch. Common thermal label: 288 &times; 144 (4 &times; 2 in).
              </div>
            </v-form>
          </section>

          <!-- ------------------------------------------------------ PRODUCTS -->
          <section v-show="tab === 'products'" class="set-section">
            <v-form :disabled="!canEditSystem">
              <h3 class="text-subtitle-1 mb-1">Smart auto-fill</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                Shows an Auto-fill button on the New Product dialog that fills
                attributes and price from the product's code and name.
              </p>

              <v-switch
                v-model="draft.productAutofillEnabled"
                color="primary"
                density="compact"
                hide-details
                inset
                label="Enable smart auto-fill"
              />
              <v-btn
                class="mt-3"
                prepend-icon="mdi-auto-fix"
                size="small"
                text="Edit auto-fill rules"
                to="/autofill-rules"
                variant="tonal"
              />
            </v-form>
          </section>

          <!-- --------------------------------------------------------- USERS -->
          <section v-show="tab === 'users'" class="set-section">
            <div class="d-flex align-center justify-space-between mb-1">
              <h3 class="text-subtitle-1">Users</h3>
              <v-btn
                color="primary"
                prepend-icon="mdi-plus"
                size="small"
                variant="tonal"
                @click="openCreate"
              >
                Add user
              </v-btn>
            </div>
            <p class="text-body-2 text-medium-emphasis mb-4">
              People who can sign in to this site (cloud.example.com), with their own passwords and the role that decides what they can see and change (see the Roles tab). POS operators are separate: they are managed per branch under Branch Settings.
            </p>

            <v-table density="compact">
              <thead>
                <tr>
                  <th class="text-left">Name</th>
                  <th class="text-left">Username</th>
                  <th class="text-left">Role</th>
                  <th class="text-left">Status</th>
                  <th class="text-left">Created</th>
                  <th class="text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="u in users" :key="u.id">
                  <td>{{ u.name }}</td>
                  <td class="text-medium-emphasis">{{ u.username }}</td>
                  <td>{{ roleName(u.role_id) }}</td>
                  <td>
                    <v-chip :color="u.activated ? 'success' : 'default'" size="x-small" variant="tonal">
                      {{ u.activated ? 'Active' : 'Inactive' }}
                    </v-chip>
                  </td>
                  <td>{{ formatDate(u.created_at) }}</td>
                  <td class="text-right">
                    <v-btn
                      icon="mdi-pencil"
                      size="small"
                      title="Edit name, role and status"
                      variant="text"
                      @click="openEdit(u)"
                    />
                    <v-btn
                      icon="mdi-key"
                      size="small"
                      title="Change password"
                      variant="text"
                      @click="openPassword(u)"
                    />
                    <v-btn
                      color="error"
                      icon="mdi-delete"
                      size="small"
                      title="Delete user"
                      variant="text"
                      @click="confirmDelete(u)"
                    />
                  </td>
                </tr>
                <tr v-if="users.length === 0">
                  <td class="text-medium-emphasis py-4" colspan="5">No users yet.</td>
                </tr>
              </tbody>
            </v-table>
          </section>

          <!-- --------------------------------------------------------- ROLES -->
          <section v-if="auth.isAdmin" v-show="tab === 'roles'" class="set-section">
            <RolesManager />
          </section>

          <!-- ------------------------------------------------------ WHATSAPP -->
          <section v-show="tab === 'whatsapp'" class="set-section set-section--wide">
            <v-form :disabled="!canEditSystem">
              <h3 class="text-subtitle-1 mb-1">Post-sale thank-you message</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                Sent after a sale to a client with a phone number: from your own WhatsApp numbers, or
                through Meta's Cloud API — chosen under <strong>Send with</strong> below.
              </p>
              <p v-if="auth.canViewPage('system', 'whatsapp')" class="text-body-2 text-medium-emphasis mb-4">
                Rules for which sales get a message, and what was sent, to whom and what it cost,
                are on the <router-link to="/whatsapp">WhatsApp page</router-link>.
              </p>

              <v-switch
                v-model="whatsappDraft.enabled"
                class="mb-2"
                color="primary"
                hide-details
                label="Send a thank-you WhatsApp after each sale"
              />

              <h4 class="text-subtitle-2 mt-4 mb-1">Meta Cloud API</h4>
              <p class="text-body-2 text-medium-emphasis mb-4">
                Only used when sending with Meta. Cloud API only allows a pre-approved message
                template for business-initiated messages, so this sets which one to use and how
                to fill it — not free-form text.
              </p>

              <v-text-field
                v-model="whatsappDraft.phoneNumberId"
                class="field-md mb-4"
                density="compact"
                hide-details
                label="Phone Number ID"
                variant="outlined"
              />
              <v-text-field
                v-model="whatsappDraft.accessToken"
                :append-inner-icon="showToken ? 'mdi-eye-off' : 'mdi-eye'"
                class="field-md mb-4"
                density="compact"
                hide-details
                label="Access token"
                :type="showToken ? 'text' : 'password'"
                variant="outlined"
                @click:append-inner="showToken = !showToken"
              />
              <v-text-field
                v-model="whatsappDraft.templateName"
                class="field-md mb-4"
                density="compact"
                hide-details
                label="Template name"
                variant="outlined"
              />
              <v-text-field
                v-model="whatsappDraft.templateLang"
                class="field-md mb-4"
                density="compact"
                hide-details
                label="Template language"
                variant="outlined"
              />
              <v-text-field
                v-model="whatsappDraft.templateParams"
                class="field-md"
                density="compact"
                hint="Comma-separated, in order, filling the template's {{1}}, {{2}}, ... body parameters. Supported: name, market, branch, invoice, total"
                label="Body params"
                persistent-hint
                variant="outlined"
              />

              <h3 class="text-subtitle-1 mt-8 mb-1">Webhook: delivery, reads and replies</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                Lets Meta tell kashi when a message is delivered or read, when it couldn't be
                delivered, and when a client writes back. In the Meta app, open
                <strong>WhatsApp &gt; Configuration</strong>, enter this callback URL and verify
                token, then subscribe to the <code>messages</code> field. The app secret is under
                <strong>App settings &gt; Basic</strong>; kashi uses it to make sure each call
                really comes from Meta.
              </p>
              <v-text-field
                append-inner-icon="mdi-content-copy"
                class="field-md mb-4"
                density="compact"
                hide-details
                label="Callback URL"
                :model-value="webhookUrl"
                readonly
                variant="outlined"
                @click:append-inner="copyText(webhookUrl)"
              />
              <v-text-field
                v-model="whatsappDraft.webhookVerifyToken"
                append-inner-icon="mdi-content-copy"
                class="field-md mb-4"
                density="compact"
                hide-details
                label="Verify token"
                variant="outlined"
                @click:append-inner="copyText(whatsappDraft.webhookVerifyToken)"
              />
              <v-text-field
                v-model="whatsappDraft.appSecret"
                :append-inner-icon="showAppSecret ? 'mdi-eye-off' : 'mdi-eye'"
                class="field-md"
                density="compact"
                hide-details
                label="App secret"
                :type="showAppSecret ? 'text' : 'password'"
                variant="outlined"
                @click:append-inner="showAppSecret = !showAppSecret"
              />

              <div class="d-flex align-center ga-3 mt-6">
                <span class="text-caption text-medium-emphasis">
                  {{ whatsappDirty ? 'Unsaved changes' : 'All changes saved' }}
                </span>
                <v-spacer />
                <v-btn
                  :disabled="!whatsappDirty || savingWhatsApp"
                  size="small"
                  text="Discard"
                  variant="text"
                  @click="seedWhatsAppDraft"
                />
                <v-btn
                  color="primary"
                  :disabled="!whatsappDirty || !canEditSystem"
                  :loading="savingWhatsApp"
                  size="small"
                  text="Save"
                  variant="flat"
                  @click="saveWhatsAppSettings"
                />
              </div>
            </v-form>

            <v-divider class="my-6" />
            <h3 class="text-subtitle-1 mb-1">Your numbers and the message</h3>
            <WhatsAppSending v-if="tab === 'whatsapp'" :settings="{ whatsapp_enabled: whatsappSaved.enabled }" />
          </section>

          <!-- ------------------------------------------------------- REVIEWS -->
          <section v-if="tab === 'reviews'" class="set-section set-section--wide">
            <ReviewSettings />
          </section>

          <!-- --------------------------------------------------------- EMAIL -->
          <section v-show="tab === 'email'" class="set-section">
            <v-form :disabled="!canEditSystem">
              <h3 class="text-subtitle-1 mb-1">Invoice report emails</h3>
              <p class="text-body-2 text-medium-emphasis mb-4">
                A day's, a date range's or a shift's invoices as a CSV, sent with the
                <strong>Email</strong> button on a Daily Income day. Mail goes out through
                Resend from any address on a domain verified there (example.com).
                <template v-if="auth.canViewPage('system', 'emails')">
                  Everything sent is on the <router-link to="/emails">Emails page</router-link>.
                </template>
              </p>
              <v-alert
                v-if="emailConfigured === false"
                class="mb-4"
                density="compact"
                type="warning"
                variant="tonal"
              >
                Sending is off: paste a Resend API key below.
              </v-alert>

              <v-text-field
                v-model="emailDraft.apiKey"
                :append-inner-icon="showEmailKey ? 'mdi-eye-off' : 'mdi-eye'"
                autocomplete="off"
                class="field-md mb-4"
                density="compact"
                :hint="emailKeyHint"
                label="Resend API key"
                persistent-hint
                :placeholder="emailKeyLast4 ? `re_••••${emailKeyLast4}` : 're_…'"
                :type="showEmailKey ? 'text' : 'password'"
                variant="outlined"
                @click:append-inner="showEmailKey = !showEmailKey"
              />

              <v-text-field
                v-model="emailDraft.senderName"
                class="field-md mb-4"
                density="compact"
                hide-details
                label="Sender name"
                variant="outlined"
              />
              <v-text-field
                v-model="emailDraft.senderAddress"
                class="field-md mb-4"
                density="compact"
                hint="Must be on the verified domain, e.g. reports@example.com"
                label="Sender address"
                persistent-hint
                variant="outlined"
              />
              <v-text-field
                v-model="emailDraft.replyTo"
                class="field-md mb-4"
                density="compact"
                hint="Where replies go; example.com has no inbox of its own"
                label="Reply-to"
                persistent-hint
                variant="outlined"
              />
              <v-text-field
                v-model="emailDraft.defaultRecipients"
                class="field-md mb-4"
                density="compact"
                hint="Pre-filled when sending; comma-separated"
                label="Default recipients"
                persistent-hint
                variant="outlined"
              />
              <v-text-field
                v-model="emailDraft.businessName"
                class="field-md"
                density="compact"
                hint="In the subject (“Demo Store Sales Report — …”) and the file name"
                label="Business name"
                persistent-hint
                variant="outlined"
              />

              <h3 class="text-subtitle-1 mt-8 mb-1">Attachment</h3>
              <p class="text-body-2 text-medium-emphasis mb-3">
                The default when sending; the send dialog can switch it per email. Excel is a
                formatted workbook with totals and a per-register summary sheet.
              </p>
              <v-btn-toggle
                v-model="emailDraft.attachmentFormat"
                class="mb-3"
                color="primary"
                density="compact"
                divided
                mandatory
                variant="outlined"
              >
                <v-btn prepend-icon="mdi-file-delimited-outline" text="CSV" value="csv" />
                <v-btn prepend-icon="mdi-microsoft-excel" text="Excel" value="xlsx" />
              </v-btn-toggle>
              <v-switch
                v-model="emailDraft.groupByRegister"
                color="primary"
                density="compact"
                hide-details
                label="Group invoices by register (Main · CT1 first, then CT2, …)"
              />
              <v-switch
                v-model="emailDraft.oldestFirst"
                color="primary"
                density="compact"
                hide-details
                label="Oldest invoice first"
              />

              <h3 class="text-subtitle-1 mt-8 mb-1">Columns</h3>
              <p class="text-body-2 text-medium-emphasis mb-3">
                In order, one row per invoice. The default is the old kashi-admin report.
              </p>
              <v-list class="email-columns mb-3" density="compact">
                <v-list-item
                  v-for="(key, index) in emailDraft.csvColumns"
                  :key="key"
                  :title="columnHeader(key)"
                >
                  <template #append>
                    <v-btn
                      :disabled="index === 0"
                      icon="mdi-arrow-up"
                      size="x-small"
                      variant="text"
                      @click="moveColumn(index, -1)"
                    />
                    <v-btn
                      :disabled="index === emailDraft.csvColumns.length - 1"
                      icon="mdi-arrow-down"
                      size="x-small"
                      variant="text"
                      @click="moveColumn(index, 1)"
                    />
                    <v-btn
                      :disabled="emailDraft.csvColumns.length === 1"
                      icon="mdi-close"
                      size="x-small"
                      variant="text"
                      @click="emailDraft.csvColumns.splice(index, 1)"
                    />
                  </template>
                </v-list-item>
              </v-list>
              <div class="d-flex align-center ga-3">
                <v-select
                  class="field-md"
                  density="compact"
                  :disabled="unusedColumns.length === 0"
                  hide-details
                  item-title="header"
                  item-value="key"
                  :items="unusedColumns"
                  label="Add a column"
                  :model-value="null"
                  variant="outlined"
                  @update:model-value="key => key && emailDraft.csvColumns.push(key)"
                />
                <v-btn size="small" text="Reset to default" variant="text" @click="emailDraft.csvColumns = [...DEFAULT_EMAIL_COLUMNS]" />
              </div>

              <div class="d-flex align-center ga-3 mt-6">
                <span class="text-caption text-medium-emphasis">
                  {{ emailDirty ? 'Unsaved changes' : 'All changes saved' }}
                </span>
                <v-spacer />
                <v-btn
                  :disabled="!emailDirty || savingEmail"
                  size="small"
                  text="Discard"
                  variant="text"
                  @click="seedEmailDraft"
                />
                <v-btn
                  color="primary"
                  :disabled="!emailDirty || !canEditSystem"
                  :loading="savingEmail"
                  size="small"
                  text="Save"
                  variant="flat"
                  @click="saveEmailSettings"
                />
              </div>
            </v-form>
          </section>
        </div>
      </div>

      <template v-if="tab !== 'users' && tab !== 'roles' && tab !== 'whatsapp' && tab !== 'email' && tab !== 'reviews'">
        <v-divider />
        <div class="set-actions px-5 py-3 d-flex align-center ga-3">
          <span class="text-caption text-medium-emphasis">
            {{ dirty ? 'Unsaved changes' : 'All changes saved' }}
          </span>
          <v-spacer />
          <v-btn
            :disabled="!dirty || savingSettings"
            size="small"
            text="Discard"
            variant="text"
            @click="seedDraft"
          />
          <v-btn
            color="primary"
            :disabled="!dirty || !draftValid || !canEditSystem"
            :loading="savingSettings"
            size="small"
            text="Save"
            variant="flat"
            @click="saveSettings"
          />
        </div>
      </template>
    </v-card>

    <!-- Add user -->
    <v-dialog v-model="createDialog" max-width="420" persistent>
      <v-card>
        <v-card-title class="text-subtitle-1">Add user</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="form.username"
            autocapitalize="none"
            autocorrect="off"
            density="compact"
            hint="What they sign in with"
            label="Username"
            persistent-hint
            :rules="[usernameRule]"
          />
          <v-text-field
            v-model="form.name"
            autocapitalize="words"
            density="compact"
            hint="Shown to others; each word is capitalised"
            label="Display name"
            persistent-hint
            :rules="[nameRule]"
          />
          <v-text-field
            v-model="form.password"
            autocomplete="new-password"
            density="compact"
            hint="8 to 72 characters"
            label="Password"
            persistent-hint
            :rules="[passwordRule]"
            type="password"
          />
          <v-select
            v-model="form.roleId"
            density="compact"
            item-title="name"
            item-value="id"
            :items="roles"
            label="Role"
          />
          <v-switch
            v-model="form.activated"
            color="primary"
            hide-details
            label="Active"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="createDialog = false">Cancel</v-btn>
          <v-btn
            color="primary"
            :disabled="!canSubmitCreate"
            :loading="saving"
            variant="tonal"
            @click="submitCreate"
          >
            Create
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Edit user -->
    <v-dialog v-model="editDialog" max-width="420" persistent>
      <v-card>
        <v-card-title class="text-subtitle-1">Edit user — {{ editTarget?.name }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="editForm.username"
            autocapitalize="none"
            autocorrect="off"
            density="compact"
            label="Username"
            :rules="[usernameRule]"
          />
          <v-text-field
            v-model="editForm.name"
            autocapitalize="words"
            density="compact"
            label="Display name"
            :rules="[nameRule]"
          />
          <v-select
            v-model="editForm.roleId"
            density="compact"
            item-title="name"
            item-value="id"
            :items="roles"
            label="Role"
          />
          <v-switch
            v-model="editForm.activated"
            color="primary"
            hide-details
            label="Active"
          />
          <div class="text-body-2 text-medium-emphasis mt-2 mb-2">Avatar colour</div>
          <div class="d-flex align-center ga-3">
            <v-avatar class="text-white font-weight-medium" :color="editAvatarColor" size="36">
              <span class="text-caption">{{ initialsOf(editForm.name) }}</span>
            </v-avatar>
            <div class="d-flex flex-wrap ga-2">
              <button
                v-for="c in avatarColors"
                :key="c"
                class="swatch"
                :class="{ 'swatch--active': c === editAvatarColor }"
                :style="{ background: c }"
                type="button"
                @click="editForm.avatarColor = c"
              />
            </div>
          </div>
          <v-btn
            v-if="editForm.avatarColor"
            class="px-1 mt-1"
            size="small"
            variant="text"
            @click="editForm.avatarColor = ''"
          >
            Reset to default colour
          </v-btn>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="editDialog = false">Cancel</v-btn>
          <v-btn
            color="primary"
            :disabled="usernameRule(editForm.username) !== true || nameRule(editForm.name) !== true || !editForm.roleId"
            :loading="saving"
            variant="tonal"
            @click="submitEdit"
          >
            Save
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Change password -->
    <v-dialog v-model="passwordDialog" max-width="420" persistent>
      <v-card>
        <v-card-title class="text-subtitle-1">Change password — {{ passwordTarget?.name }}</v-card-title>
        <v-card-text>
          <v-text-field
            v-model="newPassword"
            autocomplete="new-password"
            autofocus
            density="compact"
            hint="8 to 72 characters"
            label="New password"
            persistent-hint
            :rules="[passwordRule]"
            type="password"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="passwordDialog = false">Cancel</v-btn>
          <v-btn
            color="primary"
            :disabled="!isPassword(newPassword)"
            :loading="saving"
            variant="tonal"
            @click="submitPassword"
          >
            Save
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete confirm -->
    <v-dialog v-model="deleteDialog" max-width="380">
      <v-card>
        <v-card-title class="text-subtitle-1">Delete user</v-card-title>
        <v-card-text>
          Delete <strong>{{ deleteTarget?.name }}</strong>? This cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteDialog = false">Cancel</v-btn>
          <v-btn color="error" :loading="saving" variant="tonal" @click="submitDelete">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import ReviewSettings from '@/components/Reviews/ReviewSettings.vue'
  import RolesManager from '@/components/RolesManager.vue'
  import WhatsAppSending from '@/components/WhatsApp/WhatsAppSending.vue'
  import { useEmailReports } from '@/composables/useEmailReports'
  import { useNotify } from '@/composables/useNotify'
  import { useRoles } from '@/composables/useRoles'
  import { hashColor, initialsOf, useUserAvatar } from '@/composables/useUserAvatar'
  import { useUsers } from '@/composables/useUsers'
  import { useWhatsAppSettings } from '@/composables/useWhatsAppSettings'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'
  import { useSettingsStore } from '@/stores/settings'

  const notify = useNotify()

  // ?tab=whatsapp opens a tab directly (the WhatsApp page links here).
  const route = useRoute()
  const tab = ref(typeof route.query.tab === 'string' ? route.query.tab : 'general')
  const auth = useAuthStore()
  // Saving app settings / WhatsApp needs Edit on System; without it the forms
  // render read-only. (Your own avatar colour is per-device and always yours.)
  const canEditSystem = computed(() => auth.canEditPage('system', 'settings'))

  const {
    initials: avatarInitials,
    color: avatarColor,
    colors: avatarColors,
    isCustom: avatarIsCustom,
    setColor: setAvatarColor,
    resetColor: resetAvatarColor,
  } = useUserAvatar()

  async function pickAvatarColor (value) {
    try {
      await (value ? setAvatarColor(value) : resetAvatarColor())
    } catch (error) {
      notify.error(error.message)
    }
  }

  const settingsStore = useSettingsStore()

  // The page edits a local draft and only writes to the store (and the DB) when
  // the user hits Save. `seedDraft` copies the current persisted values in;
  // `dirty` compares field by field so Save/Discard only light up on a real
  // change.
  const draft = reactive({
    activityDisplayMode: settingsStore.activityDisplayMode,
    activityMessageSeconds: settingsStore.activityMessageSeconds,
    upcomingEventsDays: settingsStore.upcomingEventsDays,
    upcomingEventsMenuDays: settingsStore.upcomingEventsMenuDays,
    financialsHighSeasonMonths: [...settingsStore.financialsHighSeasonMonths],
    barcodeLabelWidth: settingsStore.barcodeLabelWidth,
    barcodeLabelHeight: settingsStore.barcodeLabelHeight,
    productAutofillEnabled: settingsStore.productAutofillEnabled,
    uploadPagesFollowSystem: settingsStore.uploadPagesFollowSystem,
  })

  function seedDraft () {
    draft.activityDisplayMode = settingsStore.activityDisplayMode
    draft.activityMessageSeconds = settingsStore.activityMessageSeconds
    draft.upcomingEventsDays = settingsStore.upcomingEventsDays
    draft.upcomingEventsMenuDays = settingsStore.upcomingEventsMenuDays
    draft.financialsHighSeasonMonths = [...settingsStore.financialsHighSeasonMonths]
    draft.barcodeLabelWidth = settingsStore.barcodeLabelWidth
    draft.barcodeLabelHeight = settingsStore.barcodeLabelHeight
    draft.productAutofillEnabled = settingsStore.productAutofillEnabled
    draft.uploadPagesFollowSystem = settingsStore.uploadPagesFollowSystem
  }

  const savingSettings = ref(false)

  onMounted(async () => {
    await settingsStore.init()
    seedDraft()
  })

  function sameMonths (a, b) {
    return a.length === b.length && a.toSorted().join(',') === b.toSorted().join(',')
  }

  const dirty = computed(() =>
    draft.activityDisplayMode !== settingsStore.activityDisplayMode
    || Number(draft.activityMessageSeconds) !== settingsStore.activityMessageSeconds
    || Number(draft.upcomingEventsDays) !== settingsStore.upcomingEventsDays
    || Number(draft.upcomingEventsMenuDays) !== settingsStore.upcomingEventsMenuDays
    || Number(draft.barcodeLabelWidth) !== settingsStore.barcodeLabelWidth
    || Number(draft.barcodeLabelHeight) !== settingsStore.barcodeLabelHeight
    || draft.productAutofillEnabled !== settingsStore.productAutofillEnabled
    || draft.uploadPagesFollowSystem !== settingsStore.uploadPagesFollowSystem
    || !sameMonths(draft.financialsHighSeasonMonths, settingsStore.financialsHighSeasonMonths),
  )

  // Bounds mirror the old per-field setters that silently rejected out-of-range
  // input; here they gate the Save button instead.
  const inRange = (v, lo, hi) => Number.isFinite(Number(v)) && Number(v) >= lo && Number(v) <= hi
  const draftValid = computed(() =>
    inRange(draft.activityMessageSeconds, 1, Number.POSITIVE_INFINITY)
    && inRange(draft.upcomingEventsDays, 1, 365)
    && inRange(draft.upcomingEventsMenuDays, 1, 730)
    && Number(draft.barcodeLabelWidth) > 0
    && Number(draft.barcodeLabelHeight) > 0,
  )

  async function saveSettings () {
    if (!dirty.value || !draftValid.value) {
      return
    }
    savingSettings.value = true
    try {
      await settingsStore.saveAll({
        activityDisplayMode: draft.activityDisplayMode,
        activityMessageSeconds: Number(draft.activityMessageSeconds),
        upcomingEventsDays: Number(draft.upcomingEventsDays),
        upcomingEventsMenuDays: Number(draft.upcomingEventsMenuDays),
        financialsHighSeasonMonths: [...draft.financialsHighSeasonMonths],
        barcodeLabelWidth: Number(draft.barcodeLabelWidth),
        barcodeLabelHeight: Number(draft.barcodeLabelHeight),
        productAutofillEnabled: draft.productAutofillEnabled,
        uploadPagesFollowSystem: draft.uploadPagesFollowSystem,
      })
    } finally {
      savingSettings.value = false
    }
  }

  // If init() resolves after mount, re-seed so the draft reflects real values
  // rather than the store defaults.
  watch(() => settingsStore.loaded, loaded => {
    if (loaded && !dirty.value) {
      seedDraft()
    }
  })

  const monthItems = [
    'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
    'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
  ].map((title, i) => ({ title, value: i + 1 }))

  // ---- Users ----
  const { users, fetchUsers, createUser, updateUser, deleteUser } = useUsers()

  const saving = ref(false)

  const { roles, fetchRoles } = useRoles()

  // Users and roles are admin-only endpoints; other roles never see the tab.
  onMounted(async () => {
    if (!auth.isAdmin) {return}
    try {
      await Promise.all([fetchUsers(), fetchRoles()])
    } catch (error) {
      notify.error(error.message)
    }
  })

  function roleName (id) {
    return roles.value.find(r => r.id === id)?.name ?? '—'
  }

  function formatDate (value) {
    if (!value) return '—'
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? '—' : d.toLocaleDateString()
  }

  const isPassword = v => String(v ?? '').length >= 8 && String(v).length <= 72
  const passwordRule = v => isPassword(v) || 'Password must be 8 to 72 characters'
  // Mirrors normalizeUserNames (api/user.go): the server lowercases the
  // username and capitalises each word of the display name.
  const usernameRule = v => /^[a-z0-9._-]{1,32}$/.test(String(v ?? '').trim().toLowerCase()) || 'Up to 32 letters, digits, ".", "_" or "-"'
  const nameRule = v => String(v ?? '').trim() !== '' || 'Required'

  // Add user
  const createDialog = ref(false)
  const form = reactive({ username: '', name: '', password: '', activated: true, roleId: null })
  const canSubmitCreate = computed(() => usernameRule(form.username) === true && nameRule(form.name) === true && isPassword(form.password) && !!form.roleId)

  function openCreate () {
    form.username = ''
    form.name = ''
    form.password = ''
    form.activated = true
    form.roleId = null
    createDialog.value = true
  }

  async function submitCreate () {
    saving.value = true
    try {
      await createUser({ username: form.username, name: form.name, password: form.password, activated: form.activated, role_id: form.roleId })
      await fetchUsers(true)
      createDialog.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  // Edit name / role / status
  const editDialog = ref(false)
  const editTarget = ref(null)
  const editForm = reactive({ username: '', name: '', roleId: null, activated: true, avatarColor: '' })
  // '' means "derived from the name", same as everywhere else.
  const editAvatarColor = computed(() => editForm.avatarColor || hashColor(editForm.name))

  function openEdit (user) {
    editTarget.value = user
    editForm.username = user.username
    editForm.name = user.name
    editForm.roleId = user.role_id
    editForm.activated = user.activated
    editForm.avatarColor = user.avatar_color ?? ''
    editDialog.value = true
  }

  async function submitEdit () {
    saving.value = true
    try {
      await updateUser({
        id: editTarget.value.id,
        username: editForm.username,
        name: editForm.name,
        activated: editForm.activated,
        role_id: editForm.roleId,
        avatar_color: editForm.avatarColor,
      })
      // Your own colour also lives on the signed-in user (top bar, presence).
      if (editTarget.value.id === auth.user?.id) {
        auth.user = { ...auth.user, avatar_color: editForm.avatarColor }
        try {
          localStorage.setItem('user', JSON.stringify(auth.user))
        } catch { /* the in-memory user is updated either way */ }
      }
      await Promise.all([fetchUsers(true), fetchRoles()])
      editDialog.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  // Change password
  const passwordDialog = ref(false)
  const passwordTarget = ref(null)
  const newPassword = ref('')

  function openPassword (user) {
    passwordTarget.value = user
    newPassword.value = ''
    passwordDialog.value = true
  }

  async function submitPassword () {
    saving.value = true
    try {
      await updateUser({
        id: passwordTarget.value.id,
        username: passwordTarget.value.username,
        name: passwordTarget.value.name,
        activated: passwordTarget.value.activated,
        role_id: passwordTarget.value.role_id,
        password: newPassword.value,
      })
      await fetchUsers(true)
      passwordDialog.value = false
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  // Delete
  const deleteDialog = ref(false)
  const deleteTarget = ref(null)

  function confirmDelete (user) {
    deleteTarget.value = user
    deleteDialog.value = true
  }

  async function submitDelete () {
    saving.value = true
    try {
      await deleteUser(deleteTarget.value.id)
      await fetchUsers(true)
      deleteDialog.value = false
    } catch (error) {
      notify.error(error.message)
      deleteDialog.value = false
    } finally {
      saving.value = false
    }
  }

  // ---- WhatsApp (Cloud API) ----
  // Self-contained, like Users above: its own load/dirty/save rather than
  // going through settingsStore, since it's a different set of app_settings
  // columns with its own admin endpoint (GET/PUT /whatsapp_settings).
  const { getWhatsAppSettings, updateWhatsAppSettings } = useWhatsAppSettings()

  const savingWhatsApp = ref(false)
  const showToken = ref(false)
  const showAppSecret = ref(false)
  const whatsappLoaded = ref(false)
  // Where Meta calls kashi (api/whatsapp_webhook.go), as the world sees it.
  const webhookUrl = `${API_BASE}/whatsapp/webhook`
  async function copyText (text: string) {
    try {
      await navigator.clipboard.writeText(text)
      notify.success('Copied.')
    } catch {
      notify.error('Could not copy; select the text instead.')
    }
  }

  const whatsappSaved = reactive({
    enabled: false,
    phoneNumberId: '',
    accessToken: '',
    templateName: '',
    templateLang: 'en_US',
    templateParams: 'name,invoice,total',
    webhookVerifyToken: '',
    appSecret: '',
  })
  const whatsappDraft = reactive({ ...whatsappSaved })

  function seedWhatsAppDraft () {
    Object.assign(whatsappDraft, whatsappSaved)
  }

  const whatsappDirty = computed(() =>
    whatsappDraft.enabled !== whatsappSaved.enabled
    || whatsappDraft.phoneNumberId !== whatsappSaved.phoneNumberId
    || whatsappDraft.accessToken !== whatsappSaved.accessToken
    || whatsappDraft.templateName !== whatsappSaved.templateName
    || whatsappDraft.templateLang !== whatsappSaved.templateLang
    || whatsappDraft.templateParams !== whatsappSaved.templateParams
    || whatsappDraft.webhookVerifyToken !== whatsappSaved.webhookVerifyToken
    || whatsappDraft.appSecret !== whatsappSaved.appSecret,
  )

  onMounted(async () => {
    try {
      const settings = await getWhatsAppSettings()
      if (settings) {
        Object.assign(whatsappSaved, {
          enabled: Boolean(settings.whatsapp_enabled),
          phoneNumberId: settings.whatsapp_phone_number_id ?? '',
          accessToken: settings.whatsapp_access_token ?? '',
          templateName: settings.whatsapp_template_name ?? '',
          templateLang: settings.whatsapp_template_lang ?? 'en_US',
          templateParams: settings.whatsapp_template_params ?? 'name,invoice,total',
          webhookVerifyToken: settings.whatsapp_webhook_verify_token ?? '',
          appSecret: settings.whatsapp_app_secret ?? '',
        })
        seedWhatsAppDraft()
      }
    } catch (error) {
      notify.error(error.message)
    } finally {
      whatsappLoaded.value = true
    }
  })

  async function saveWhatsAppSettings () {
    if (!whatsappDirty.value) {
      return
    }
    savingWhatsApp.value = true
    try {
      await updateWhatsAppSettings({
        whatsapp_enabled: whatsappDraft.enabled,
        whatsapp_phone_number_id: whatsappDraft.phoneNumberId,
        whatsapp_access_token: whatsappDraft.accessToken,
        whatsapp_template_name: whatsappDraft.templateName,
        whatsapp_template_lang: whatsappDraft.templateLang,
        whatsapp_template_params: whatsappDraft.templateParams,
        whatsapp_webhook_verify_token: whatsappDraft.webhookVerifyToken,
        whatsapp_app_secret: whatsappDraft.appSecret,
      })
      Object.assign(whatsappSaved, whatsappDraft)
    } catch (error) {
      notify.error(error.message)
    } finally {
      savingWhatsApp.value = false
    }
  }

  // ---- Email (invoice reports via Resend) ----
  // Same shape as WhatsApp above: its own app_settings columns and endpoint
  // (GET/PUT /email_settings). The Resend key itself stays in app.env.
  const { getEmailSettings, updateEmailSettings } = useEmailReports()
  const DEFAULT_EMAIL_COLUMNS = ['invoice_no', 'type', 'date', 'net_usd', 'net_lbp']

  const savingEmail = ref(false)
  const showEmailKey = ref(false)
  const emailConfigured = ref(null)
  // The saved key never comes back from the API, only where it's from and
  // its last 4 characters; the field stays empty unless replacing it.
  const emailKeySource = ref('')
  const emailKeyLast4 = ref('')
  const emailKeyHint = computed(() => {
    if (emailKeySource.value === 'settings') return `Saved, ending in ${emailKeyLast4.value}. Paste a new one to replace it.`
    if (emailKeySource.value === 'env') return `Using RESEND_API_KEY from the server's app.env (ending in ${emailKeyLast4.value}).`
    return 'From resend.com > API Keys, with sending access to example.com'
  })
  const emailColumns = ref([])
  const emailSaved = reactive({
    senderName: '',
    senderAddress: '',
    replyTo: '',
    defaultRecipients: '',
    businessName: '',
    csvColumns: [...DEFAULT_EMAIL_COLUMNS],
    attachmentFormat: 'csv',
    groupByRegister: true,
    oldestFirst: false,
    apiKey: '',
  })
  const emailDraft = reactive({ ...emailSaved, csvColumns: [...emailSaved.csvColumns] })

  function seedEmailDraft () {
    Object.assign(emailDraft, emailSaved, { csvColumns: [...emailSaved.csvColumns] })
  }

  const emailDirty = computed(() =>
    emailDraft.senderName !== emailSaved.senderName
    || emailDraft.senderAddress !== emailSaved.senderAddress
    || emailDraft.replyTo !== emailSaved.replyTo
    || emailDraft.defaultRecipients !== emailSaved.defaultRecipients
    || emailDraft.businessName !== emailSaved.businessName
    || emailDraft.csvColumns.join(',') !== emailSaved.csvColumns.join(',')
    || emailDraft.attachmentFormat !== emailSaved.attachmentFormat
    || emailDraft.groupByRegister !== emailSaved.groupByRegister
    || emailDraft.oldestFirst !== emailSaved.oldestFirst
    || emailDraft.apiKey.trim() !== '',
  )

  const unusedColumns = computed(() => emailColumns.value.filter(c => !emailDraft.csvColumns.includes(c.key)))
  function columnHeader (key) {
    return emailColumns.value.find(c => c.key === key)?.header ?? key
  }
  function moveColumn (index, by) {
    const columns = emailDraft.csvColumns
    ;[columns[index], columns[index + by]] = [columns[index + by], columns[index]]
  }

  function applyEmailSettings (data) {
    if (!data?.settings) return
    emailConfigured.value = Boolean(data.configured)
    emailKeySource.value = data.api_key_source ?? ''
    emailKeyLast4.value = data.api_key_hint ?? ''
    emailColumns.value = data.columns ?? []
    const settings = data.settings
    Object.assign(emailSaved, {
      senderName: settings.email_sender_name ?? '',
      senderAddress: settings.email_sender_address ?? '',
      replyTo: settings.email_reply_to ?? '',
      defaultRecipients: settings.email_default_recipients ?? '',
      businessName: settings.email_business_name ?? '',
      csvColumns: Array.isArray(settings.email_csv_columns) && settings.email_csv_columns.length > 0
        ? [...settings.email_csv_columns]
        : [...DEFAULT_EMAIL_COLUMNS],
      attachmentFormat: settings.email_attachment_format || 'csv',
      groupByRegister: settings.email_group_by_register ?? true,
      oldestFirst: Boolean(settings.email_oldest_first),
      apiKey: '',
    })
    seedEmailDraft()
  }

  onMounted(async () => {
    try {
      applyEmailSettings(await getEmailSettings())
    } catch (error) {
      notify.error(error.message)
    }
  })

  async function saveEmailSettings () {
    if (!emailDirty.value) {
      return
    }
    savingEmail.value = true
    try {
      applyEmailSettings(await updateEmailSettings({
        email_sender_name: emailDraft.senderName,
        email_sender_address: emailDraft.senderAddress,
        email_reply_to: emailDraft.replyTo,
        email_default_recipients: emailDraft.defaultRecipients,
        email_business_name: emailDraft.businessName,
        email_csv_columns: emailDraft.csvColumns,
        email_attachment_format: emailDraft.attachmentFormat,
        email_group_by_register: emailDraft.groupByRegister,
        email_oldest_first: emailDraft.oldestFirst,
        // Only sent when replacing the key; absent keeps the saved one.
        ...(emailDraft.apiKey.trim() ? { email_resend_api_key: emailDraft.apiKey.trim() } : {}),
      }))
    } catch (error) {
      notify.error(error.message)
    } finally {
      savingEmail.value = false
    }
  }
</script>

<style scoped>
/* Fill the layout's flex-column scroll wrapper so the scroll lives inside the
   settings pane, not the whole page. */
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.set-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.set-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.set-tabs {
  flex: 0 0 180px;
}
.set-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
}
.set-section {
  max-width: 640px;
}
/* WhatsApp's numbers and message preview, and Reviews' two-column form. */
.set-section--wide {
  max-width: 1100px;
}
/* Save / Discard bar pinned under the tab content. */
.set-actions {
  flex: 0 0 auto;
}
.field-md {
  max-width: 420px;
}
.email-columns {
  max-width: 420px;
  border: thin solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 4px;
}
.swatch {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  outline-offset: 2px;
  transition: transform 0.1s ease;
}
.swatch:hover {
  transform: scale(1.12);
}
.swatch--active {
  border-color: rgb(var(--v-theme-on-surface));
}
</style>
