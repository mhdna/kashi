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
          class="set-tabs"
          color="primary"
          direction="vertical"
        >
          <v-tab prepend-icon="mdi-cog-outline" text="General" value="general" />
          <v-tab prepend-icon="mdi-chart-line" text="Financials" value="financials" />
          <v-tab prepend-icon="mdi-barcode" text="Barcode labels" value="barcodes" />
          <v-tab prepend-icon="mdi-account-group" text="Users" value="users" />
        </v-tabs>

        <v-divider vertical />

        <div class="set-content pa-5">
          <!-- ------------------------------------------------------- GENERAL -->
          <section v-show="tab === 'general'" class="set-section">
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

            <h3 class="text-subtitle-1 mb-1 mt-6">Your avatar</h3>
            <p class="text-body-2 text-medium-emphasis mb-3">
              The initials badge in the top bar — your name's first two letters.
              The colour is saved on this device and applies right away.
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
                  @click="setAvatarColor(c)"
                />
              </div>
            </div>
            <v-btn
              v-if="avatarIsCustom"
              class="px-1"
              size="small"
              variant="text"
              @click="resetAvatarColor"
            >
              Reset to default colour
            </v-btn>
          </section>

          <!-- ---------------------------------------------------- FINANCIALS -->
          <section v-show="tab === 'financials'" class="set-section">
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
          </section>

          <!-- ------------------------------------------------------ BARCODES -->
          <section v-show="tab === 'barcodes'" class="set-section">
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
              POS operators and their login PINs. Changes take effect on the next login.
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
                  <th class="text-left">Name</th>
                  <th class="text-left">Status</th>
                  <th class="text-left">Created</th>
                  <th class="text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="u in users" :key="u.id">
                  <td>{{ u.name }}</td>
                  <td>
                    <v-chip :color="u.activated ? 'success' : 'default'" size="x-small" variant="tonal">
                      {{ u.activated ? 'Active' : 'Inactive' }}
                    </v-chip>
                  </td>
                  <td>{{ formatDate(u.created_at) }}</td>
                  <td class="text-right">
                    <v-btn
                      icon="mdi-key"
                      size="small"
                      title="Change PIN"
                      variant="text"
                      @click="openPin(u)"
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
                  <td class="text-medium-emphasis py-4" colspan="4">No users yet.</td>
                </tr>
              </tbody>
            </v-table>
          </section>
        </div>
      </div>

      <template v-if="tab !== 'users'">
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
            :disabled="!dirty || !draftValid"
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
    <v-dialog v-model="createDialog" max-width="420">
      <v-card>
        <v-card-title class="text-subtitle-1">Add user</v-card-title>
        <v-card-text>
          <v-alert
            v-if="formError"
            class="mb-3"
            density="compact"
            type="error"
            variant="tonal"
          >
            {{ formError }}
          </v-alert>
          <v-text-field
            v-model="form.name"
            density="compact"
            label="Name"
            :rules="[nameRule]"
          />
          <v-text-field
            v-model="form.pin"
            density="compact"
            hint="4 to 6 digits"
            inputmode="numeric"
            label="PIN"
            maxlength="6"
            persistent-hint
            :rules="[pinRule]"
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

    <!-- Change PIN -->
    <v-dialog v-model="pinDialog" max-width="420">
      <v-card>
        <v-card-title class="text-subtitle-1">Change PIN — {{ pinTarget?.name }}</v-card-title>
        <v-card-text>
          <v-alert
            v-if="formError"
            class="mb-3"
            density="compact"
            type="error"
            variant="tonal"
          >
            {{ formError }}
          </v-alert>
          <v-text-field
            v-model="newPin"
            autofocus
            density="compact"
            hint="4 to 6 digits"
            inputmode="numeric"
            label="New PIN"
            maxlength="6"
            persistent-hint
            :rules="[pinRule]"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="pinDialog = false">Cancel</v-btn>
          <v-btn
            color="primary"
            :disabled="!isPin(newPin)"
            :loading="saving"
            variant="tonal"
            @click="submitPin"
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
  import { useUserAvatar } from '@/composables/useUserAvatar'
  import { useUsers } from '@/composables/useUsers'
  import { useSettingsStore } from '@/stores/settings'

  const tab = ref('general')

  const {
    initials: avatarInitials,
    color: avatarColor,
    colors: avatarColors,
    isCustom: avatarIsCustom,
    setColor: setAvatarColor,
    resetColor: resetAvatarColor,
  } = useUserAvatar()

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
  })

  function seedDraft () {
    draft.activityDisplayMode = settingsStore.activityDisplayMode
    draft.activityMessageSeconds = settingsStore.activityMessageSeconds
    draft.upcomingEventsDays = settingsStore.upcomingEventsDays
    draft.upcomingEventsMenuDays = settingsStore.upcomingEventsMenuDays
    draft.financialsHighSeasonMonths = [...settingsStore.financialsHighSeasonMonths]
    draft.barcodeLabelWidth = settingsStore.barcodeLabelWidth
    draft.barcodeLabelHeight = settingsStore.barcodeLabelHeight
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

  const usersError = ref('')
  const formError = ref('')
  const saving = ref(false)

  onMounted(async () => {
    try {
      await fetchUsers()
    } catch (error) {
      usersError.value = error.message
    }
  })

  function formatDate (value) {
    if (!value) return '—'
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? '—' : d.toLocaleDateString()
  }

  const isPin = v => /^\d{4,6}$/.test(String(v ?? ''))
  const pinRule = v => isPin(v) || 'PIN must be 4 to 6 digits'
  const nameRule = v => /^[a-zA-Z0-9]+$/.test(String(v ?? '')) || 'Letters and numbers only'

  // Add user
  const createDialog = ref(false)
  const form = reactive({ name: '', pin: '', activated: true })
  const canSubmitCreate = computed(() => nameRule(form.name) === true && isPin(form.pin))

  function openCreate () {
    form.name = ''
    form.pin = ''
    form.activated = true
    formError.value = ''
    createDialog.value = true
  }

  async function submitCreate () {
    saving.value = true
    formError.value = ''
    try {
      await createUser({ name: form.name, password: form.pin, activated: form.activated })
      await fetchUsers(true)
      createDialog.value = false
    } catch (error) {
      formError.value = error.message
    } finally {
      saving.value = false
    }
  }

  // Change PIN
  const pinDialog = ref(false)
  const pinTarget = ref(null)
  const newPin = ref('')

  function openPin (user) {
    pinTarget.value = user
    newPin.value = ''
    formError.value = ''
    pinDialog.value = true
  }

  async function submitPin () {
    saving.value = true
    formError.value = ''
    try {
      await updateUser({
        id: pinTarget.value.id,
        name: pinTarget.value.name,
        activated: pinTarget.value.activated,
        password: newPin.value,
      })
      await fetchUsers(true)
      pinDialog.value = false
    } catch (error) {
      formError.value = error.message
    } finally {
      saving.value = false
    }
  }

  // Delete
  const deleteDialog = ref(false)
  const deleteTarget = ref(null)

  function confirmDelete (user) {
    deleteTarget.value = user
    formError.value = ''
    deleteDialog.value = true
  }

  async function submitDelete () {
    saving.value = true
    try {
      await deleteUser(deleteTarget.value.id)
      await fetchUsers(true)
      deleteDialog.value = false
    } catch (error) {
      usersError.value = error.message
      deleteDialog.value = false
    } finally {
      saving.value = false
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
/* Save / Discard bar pinned under the tab content. */
.set-actions {
  flex: 0 0 auto;
}
.field-md {
  max-width: 420px;
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
