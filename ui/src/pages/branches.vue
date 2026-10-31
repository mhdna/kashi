<template>
  <v-dialog v-model="createDialog" max-width="480">
    <v-card class="px-4">
      <v-card-title>Add a New Branch</v-card-title>
      <v-card-text>
        <form @submit.prevent="submitCreate">
          <v-text-field
            v-model="createName"
            density="compact"
            label="Name"
          />
          <v-text-field
            v-model="createCode"
            density="compact"
            hint="Short unique code, e.g. MAIN"
            label="Code"
            persistent-hint
            @update:model-value="createCode = createCode.toUpperCase()"
          />

          <v-card-actions class="px-0">
            <v-spacer />
            <v-btn text="Cancel" @click="createDialog = false" />
            <v-btn color="primary" :loading="creating" text="Add Branch" type="submit" />
          </v-card-actions>
        </form>
      </v-card-text>
    </v-card>
  </v-dialog>

  <!-- Shown exactly once, right after creating a branch or a register, or
       rotating a register's key -- the plaintext API key is never
       retrievable again afterward, only its bcrypt hash is stored
       server-side. -->
  <v-dialog v-model="keyDialog" max-width="520">
    <v-card class="px-4">
      <v-card-title>API Key</v-card-title>
      <v-card-text>
        <p v-if="revealedCode" class="mb-3">
          Enter code <strong>{{ revealedCode }}</strong> and this key in the till's kashi-pos settings.
        </p>
        <v-text-field
          v-model="revealedKey"
          density="compact"
          label="API key"
          readonly
          variant="outlined"
        >
          <template #append-inner>
            <v-icon-btn icon="mdi-content-copy" size="small" variant="text" @click="copyKey" />
          </template>
        </v-text-field>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn color="primary" text="Done" @click="keyDialog = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- A branch can run several tills ("registers"). Each logs in with its
       own code and key; the code is also the prefix of its invoice numbers
       (SA/<code>/<year>/<n>), so it can't change once the till is in use. -->
  <v-dialog v-model="registersDialog" max-width="720" scrollable>
    <v-card>
      <v-card-title class="d-flex align-center ga-2">
        <v-icon icon="mdi-cash-register" />
        <span>Registers — {{ registersBranch?.name }}</span>
      </v-card-title>
      <v-divider />
      <v-card-text>
        <v-table density="compact">
          <thead>
            <tr>
              <th>Code</th>
              <th>Name</th>
              <th>Status</th>
              <th class="text-center">Active</th>
              <th class="text-center">In reports</th>
              <th />
            </tr>
          </thead>
          <tbody>
            <tr v-for="register in registers" :key="register.id">
              <td><strong>{{ register.code }}</strong></td>
              <td>{{ register.name || '—' }}</td>
              <td>
                <v-chip
                  v-if="connectedRegisterIds.includes(register.id)"
                  color="success"
                  size="small"
                  text="Online"
                  variant="tonal"
                />
                <span v-else class="text-medium-emphasis">
                  {{ register.last_seen_at?.Valid ? `Last seen ${new Date(register.last_seen_at.Time).toLocaleString()}` : 'Never connected' }}
                </span>
              </td>
              <td class="text-center">
                <v-switch
                  class="d-inline-flex"
                  color="primary"
                  density="compact"
                  :disabled="!canEdit"
                  hide-details
                  :model-value="register.is_active"
                  @update:model-value="value => toggleRegisterActive(register, value)"
                />
              </td>
              <td class="text-center">
                <v-switch
                  class="d-inline-flex"
                  color="primary"
                  density="compact"
                  :disabled="!canEdit"
                  hide-details
                  :model-value="!register.hidden_from_reports"
                  @update:model-value="value => toggleRegisterInReports(register, value)"
                />
              </td>
              <td class="text-end">
                <v-icon-btn
                  v-if="canEdit"
                  icon="mdi-key"
                  size="small"
                  variant="text"
                  @click="openRotate(register)"
                />
              </td>
            </tr>
            <tr v-if="!registersLoading && registers.length === 0">
              <td class="text-medium-emphasis" colspan="6">No registers yet.</td>
            </tr>
          </tbody>
        </v-table>

        <form v-if="canEdit" class="d-flex flex-wrap align-start ga-3 mt-4" @submit.prevent="submitRegister">
          <v-text-field
            v-model="registerCode"
            class="register-code-field"
            density="compact"
            hint="1–8 letters/digits, e.g. CT2"
            label="Code"
            maxlength="8"
            persistent-hint
            @update:model-value="registerCode = registerCode.toUpperCase()"
          />
          <v-text-field
            v-model="registerName"
            density="compact"
            hint="Optional, e.g. Second till"
            label="Name"
            persistent-hint
          />
          <v-btn
            color="primary"
            :disabled="!registerCode"
            :loading="addingRegister"
            prepend-icon="mdi-plus"
            text="Add Register"
            type="submit"
          />
        </form>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Close" @click="registersDialog = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="rotateDialog" max-width="420">
    <v-card>
      <v-card-title>Rotate API Key</v-card-title>
      <v-card-text>
        Rotating register {{ rotateTarget?.code }}'s key immediately invalidates its current key.
        That till will need the new key entered before it can sync again.
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Cancel" @click="rotateDialog = false" />
        <v-btn color="warning" :loading="rotating" text="Rotate Key" @click="confirmRotate" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Targets are admin-managed only here -- kashi-pos pulls them down
       read-only via the branch-sync mechanism (GET /branch/sync/changes),
       it never creates or edits them. Fixed-height + scrollable so the
       header/tabs/footer stay put while a long target list scrolls in
       the middle, instead of the whole dialog growing with content. -->
  <v-dialog v-model="targetsDialog" max-width="720" scrollable>
    <v-card class="targets-dialog-card">
      <v-card-title class="d-flex align-center">
        Targets -- {{ targetsBranch?.name }}
        <v-spacer />
        <v-icon-btn icon="mdi-close" size="small" variant="text" @click="targetsDialog = false" />
      </v-card-title>

      <v-tabs v-model="targetsTab" color="primary">
        <v-tab value="targets">Targets</v-tab>
        <v-tab value="recurring">Recurring</v-tab>
      </v-tabs>
      <v-divider />

      <v-card-text class="targets-dialog-body">
        <div v-if="targetsLoading" class="d-flex justify-center pa-8">
          <PreparingState kind="data" min-height="160px" title="Loading targets" />
        </div>

        <v-window v-else v-model="targetsTab">
          <v-window-item value="targets">
            <TargetProgressBars v-if="targetsList.length > 0" class="mb-4" :targets="targetsList" />
            <div v-if="targetsList.length === 0" class="targets-empty">
              <v-icon class="mb-2" icon="mdi-flag-outline" size="32" />
              <div>No targets set for this branch yet.</div>
            </div>
            <v-list v-else class="target-list" lines="two">
              <v-list-item v-for="target in targetsList" :key="target.id" class="target-list-item">
                <template #prepend>
                  <span class="target-color-swatch" :style="{ background: target.color }" />
                </template>
                <v-list-item-title class="d-flex align-center ga-1">
                  ${{ (target.target_amount / 100).toLocaleString() }}
                  <v-icon
                    v-if="target.series_id?.Valid"
                    icon="mdi-repeat"
                    size="14"
                    title="Generated by a recurring target"
                  />
                </v-list-item-title>
                <v-list-item-subtitle>
                  {{ new Date(target.date_from).toLocaleDateString() }} -- {{ new Date(target.date_to).toLocaleDateString() }}
                </v-list-item-subtitle>
                <template #append>
                  <v-icon-btn
                    v-if="canEdit"
                    icon="mdi-pencil"
                    size="small"
                    variant="text"
                    @click="openEditTarget(target)"
                  />
                  <v-icon-btn
                    v-if="canEdit"
                    icon="mdi-delete"
                    size="small"
                    variant="text"
                    @click="removeTarget(target)"
                  />
                </template>
              </v-list-item>
            </v-list>
          </v-window-item>

          <v-window-item value="recurring">
            <div v-if="seriesList.length === 0" class="targets-empty">
              <v-icon class="mb-2" icon="mdi-repeat" size="32" />
              <div>No recurring targets set up yet.</div>
              <div class="text-caption text-medium-emphasis mt-1">
                Each period (e.g. every month) creates its own target automatically, starting fresh at $0.
              </div>
            </div>
            <v-list v-else class="target-list" lines="two">
              <v-list-item v-for="s in seriesList" :key="s.id" class="target-list-item">
                <template #prepend>
                  <span class="target-color-swatch" :style="{ background: s.color }" />
                </template>
                <v-list-item-title>${{ (s.target_amount / 100).toLocaleString() }}</v-list-item-title>
                <v-list-item-subtitle>{{ describeSeries(s) }}</v-list-item-subtitle>
                <template #append>
                  <v-switch
                    class="me-1"
                    color="primary"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    :model-value="s.active"
                    title="Pause/resume -- keeps already-generated targets"
                    @update:model-value="value => toggleSeriesActive(s, value)"
                  />
                  <v-icon-btn
                    v-if="canEdit"
                    icon="mdi-pencil"
                    size="small"
                    variant="text"
                    @click="openEditSeries(s)"
                  />
                  <v-icon-btn
                    v-if="canEdit"
                    icon="mdi-delete"
                    size="small"
                    variant="text"
                    @click="openDeleteSeriesConfirm(s)"
                  />
                </template>
              </v-list-item>
            </v-list>
          </v-window-item>
        </v-window>
      </v-card-text>

      <v-divider />
      <v-card-actions>
        <v-btn
          v-if="canEdit && targetsTab === 'targets'"
          color="primary"
          prepend-icon="mdi-plus"
          text="Add Target"
          variant="tonal"
          @click="openAddTarget"
        />
        <v-btn
          v-if="canEdit && targetsTab !== 'targets'"
          color="primary"
          prepend-icon="mdi-plus"
          text="Add Recurring Target"
          variant="tonal"
          @click="openAddSeries"
        />
        <v-spacer />
        <v-btn text="Close" @click="targetsDialog = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="targetFormDialog" max-width="440" :persistent="true">
    <v-card class="px-4">
      <v-card-title>{{ editingTargetId ? "Edit Target" : "Add Target" }}</v-card-title>
      <v-card-text>
        <form class="field-grid" @submit.prevent="submitTarget">
          <DateRangePicker v-model:from="targetForm.dateFrom" v-model:to="targetForm.dateTo" class="field-full" />
          <v-text-field
            v-model.number="targetForm.targetAmount"
            class="field-full"
            density="compact"
            label="Target Amount ($)"
            min="0"
            prefix="$"
            step="0.01"
            type="number"
            variant="outlined"
          />
          <div class="d-flex align-center ga-2 field-full">
            <v-checkbox
              v-model="targetForm.autoColor"
              density="compact"
              hide-details
              :label="editingTargetId ? 'Keep current color' : 'Auto-assign color'"
            />
            <v-text-field
              v-if="!targetForm.autoColor"
              v-model="targetForm.color"
              density="compact"
              hide-details
              label="Color"
              style="max-width: 100px"
              type="color"
              variant="outlined"
            />
          </div>

          <v-card-actions class="px-0 field-full">
            <v-spacer />
            <v-btn text="Cancel" variant="text" @click="targetFormDialog = false" />
            <v-btn
              color="primary"
              :loading="targetSaving"
              :text="editingTargetId ? 'Save' : 'Add'"
              type="submit"
            />
          </v-card-actions>
        </form>
      </v-card-text>
    </v-card>
  </v-dialog>

  <v-dialog v-model="seriesFormDialog" max-width="440" :persistent="true">
    <v-card class="px-4">
      <v-card-title>{{ editingSeriesId ? "Edit Recurring Target" : "Add Recurring Target" }}</v-card-title>
      <v-card-text>
        <form class="field-grid" @submit.prevent="submitSeries">
          <v-text-field
            v-model.number="seriesForm.targetAmount"
            class="field-full"
            density="compact"
            label="Target Amount ($)"
            min="0"
            prefix="$"
            step="0.01"
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-model.number="seriesForm.startDay"
            density="compact"
            hint="Day of month a period starts on -- clamped in short months"
            label="Starts on day"
            max="31"
            min="1"
            persistent-hint
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-model.number="seriesForm.intervalCount"
            density="compact"
            hint="1 = every month, 3 = quarterly, etc."
            label="Repeat every N month(s)"
            min="1"
            persistent-hint
            type="number"
            variant="outlined"
          />
          <div class="d-flex align-center ga-2 field-full">
            <v-checkbox
              v-model="seriesForm.autoColor"
              density="compact"
              hide-details
              :label="editingSeriesId ? 'Keep current color' : 'Auto-assign color'"
            />
            <v-text-field
              v-if="!seriesForm.autoColor"
              v-model="seriesForm.color"
              density="compact"
              hide-details
              label="Color"
              style="max-width: 100px"
              type="color"
              variant="outlined"
            />
          </div>

          <div v-if="editingSeriesId" class="text-caption text-medium-emphasis field-full">
            Changes only apply to periods generated from now on -- the current period keeps its
            already-set amount and color.
          </div>

          <v-card-actions class="px-0 field-full">
            <v-spacer />
            <v-btn text="Cancel" variant="text" @click="seriesFormDialog = false" />
            <v-btn
              color="primary"
              :loading="seriesSaving"
              :text="editingSeriesId ? 'Save' : 'Add'"
              type="submit"
            />
          </v-card-actions>
        </form>
      </v-card-text>
    </v-card>
  </v-dialog>

  <v-dialog v-model="deleteSeriesDialog" max-width="420">
    <v-card>
      <v-card-title>Delete Recurring Target</v-card-title>
      <v-card-text>
        This removes the recurring schedule and every target it has generated (past and
        current), for every branch install syncing it. This can't be undone.
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Cancel" @click="deleteSeriesDialog = false" />
        <v-btn color="error" :loading="deleteSeriesSaving" text="Delete" @click="confirmDeleteSeries" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- The branch's staff roster is admin-managed only here -- kashi-pos pulls
       it down read-only via the branch-sync mechanism (GET
       /branch/sync/changes). The attendance ID is the person's number on the
       branch's fingerprint device: set here it wins on the till, left empty
       here the till's own value shows. "Sells" puts them in the till's
       checkout picker; off is for staff who only punch attendance. The color
       is the avatar background on the till. -->
  <v-dialog v-model="salespersonsDialog" max-width="720">
    <v-card>
      <v-card-title>Staff -- {{ salespersonsBranch?.name }}</v-card-title>
      <v-card-text>
        <div v-if="salespersonsLoading" class="d-flex justify-center pa-4">
          <PreparingState kind="data" min-height="160px" title="Loading staff" />
        </div>
        <template v-else>
          <v-alert v-if="salespersonsList.length === 0" class="mb-4" type="info" variant="tonal">
            No staff for this branch yet.
          </v-alert>

          <v-table class="mb-4" density="compact">
            <thead>
              <tr><th>Color</th><th>Name</th><th title="User number on the branch's fingerprint device">Attendance ID</th><th title="Offered as a salesperson at checkout">Sells</th><th>Active</th><th /></tr>
            </thead>
            <tbody>
              <tr v-for="person in salespersonsList" :key="person.id">
                <td>
                  <input
                    aria-label="Avatar color"
                    class="salesperson-color-input"
                    :disabled="!canEdit"
                    type="color"
                    :value="person.color || '#546E7A'"
                    @change="event => changeSalespersonColor(person, event.target.value)"
                  >
                </td>
                <td>
                  <v-text-field
                    v-if="editingSalespersonId === person.id"
                    v-model="editingSalespersonName"
                    density="compact"
                    hide-details
                    variant="outlined"
                    @keyup.enter="submitEditSalesperson(person)"
                  />
                  <template v-else>{{ person.name }}</template>
                </td>
                <td>
                  <v-text-field
                    v-if="editingSalespersonId === person.id"
                    v-model="editingSalespersonAttendanceId"
                    density="compact"
                    hide-details
                    placeholder="e.g. 15"
                    style="max-width: 110px"
                    variant="outlined"
                    @keyup.enter="submitEditSalesperson(person)"
                  />
                  <span v-else :class="{ 'text-medium-emphasis': !person.attendance_user_id }">
                    {{ person.attendance_user_id || '—' }}
                  </span>
                </td>
                <td>
                  <v-checkbox-btn
                    density="compact"
                    :disabled="!canEdit"
                    :model-value="person.is_salesperson"
                    @update:model-value="value => toggleSalespersonSells(person, value)"
                  />
                </td>
                <td>
                  <v-switch
                    color="primary"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    :model-value="person.is_active"
                    @update:model-value="value => toggleSalespersonActive(person, value)"
                  />
                </td>
                <td class="text-end">
                  <v-icon-btn
                    v-if="canEdit && editingSalespersonId === person.id"
                    icon="mdi-check"
                    size="small"
                    variant="text"
                    @click="submitEditSalesperson(person)"
                  />
                  <v-icon-btn
                    v-else-if="canEdit"
                    icon="mdi-pencil"
                    size="small"
                    variant="text"
                    @click="openEditSalesperson(person)"
                  />
                </td>
              </tr>
            </tbody>
          </v-table>

          <form v-if="canEdit" class="d-flex ga-2 align-start" @submit.prevent="submitNewSalesperson">
            <v-text-field
              v-model="newSalespersonName"
              density="compact"
              hide-details
              label="Full name"
              variant="outlined"
            />
            <v-text-field
              v-model="newSalespersonAttendanceId"
              density="compact"
              hide-details
              label="Attendance ID"
              style="max-width: 140px"
              variant="outlined"
            />
            <v-checkbox
              v-model="newSalespersonSells"
              density="compact"
              hide-details
              label="Sells"
              title="Offered as a salesperson at checkout"
            />
            <v-btn color="primary" :loading="salespersonSaving" text="Add" type="submit" />
          </form>
        </template>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn text="Close" @click="salespersonsDialog = false" />
      </v-card-actions>
    </v-card>
  </v-dialog>

  <div class="page-root">
    <v-card class="branches-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-store-outline" />
        <span>Branches</span>
        <v-spacer />
        <v-btn
          v-if="canEdit"
          color="primary"
          prepend-icon="mdi-plus"
          text="Add Branch"
          variant="flat"
          @click="openCreate"
        />
      </v-card-title>
      <v-divider />

      <div class="branches-content pa-4">
        <v-data-table class="branches-table" :headers="headers" :items="branches" :loading="listLoading">
          <template #item.is_active="{ item }">
            <v-switch
              color="primary"
              density="compact"
              :disabled="!canEdit"
              hide-details
              :model-value="item.is_active"
              @update:model-value="value => toggleActive(item, value)"
            />
          </template>
          <template #item.hidden_from_reports="{ item }">
            <v-switch
              color="primary"
              density="compact"
              :disabled="!canEdit"
              hide-details
              :model-value="!item.hidden_from_reports"
              @update:model-value="value => toggleInReports(item, value)"
            />
          </template>
          <template #item.last_seen_at="{ item }">
            {{ item.last_seen_at?.Valid ? new Date(item.last_seen_at.Time).toLocaleString() : 'Never' }}
          </template>
          <template #item.actions="{ item }">
            <v-tooltip location="top" text="Targets">
              <template #activator="{ props: tip }">
                <v-icon-btn
                  v-bind="tip"
                  aria-label="Targets"
                  icon="mdi-flag-outline"
                  size="small"
                  variant="text"
                  @click="openTargets(item)"
                />
              </template>
            </v-tooltip>
            <v-tooltip location="top" text="Staff: add salespersons and other staff">
              <template #activator="{ props: tip }">
                <v-icon-btn
                  v-bind="tip"
                  aria-label="Staff"
                  icon="mdi-account-tie-outline"
                  size="small"
                  variant="text"
                  @click="openSalespersons(item)"
                />
              </template>
            </v-tooltip>
            <v-tooltip location="top" text="Registers (tills)">
              <template #activator="{ props: tip }">
                <v-icon-btn
                  v-bind="tip"
                  aria-label="Registers"
                  icon="mdi-cash-register"
                  size="small"
                  variant="text"
                  @click="openRegisters(item)"
                />
              </template>
            </v-tooltip>
          </template>
        </v-data-table>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, onUnmounted, ref } from 'vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSalespersons } from '@/composables/useBranchSalespersons'
  import { useBranchTargets } from '@/composables/useBranchTargets'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'

  const notify = useNotify()

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  const {
    branches,
    fetchBranches,
    createBranch,
    setBranchActive,
    setBranchHiddenFromReports,
    listRegisters,
    createRegister,
    setRegisterActive,
    setRegisterHiddenFromReports,
    rotateRegisterKey,
    fetchConnectedRegisters,
  } = useBranches()
  const { ensureConnected, onMessage } = useAdminSocket()
  const {
    listBranchTargets,
    createBranchTarget,
    updateBranchTarget,
    deleteBranchTarget,
    listBranchTargetSeries,
    createBranchTargetSeries,
    updateBranchTargetSeries,
    deleteBranchTargetSeries,
    setBranchTargetSeriesActive,
  } = useBranchTargets()
  const {
    listBranchSalespersons,
    createBranchSalesperson,
    setBranchSalespersonAttendanceId,
    setBranchSalespersonIsSalesperson,
    updateBranchSalesperson,
    setBranchSalespersonActive,
    setBranchSalespersonColor,
  } = useBranchSalespersons()

  const headers = ref([
    { title: 'Name', key: 'name', align: 'start' },
    { title: 'Code', key: 'code', align: 'start' },
    { title: 'Active', key: 'is_active', align: 'center', sortable: false },
    // Off keeps the branch syncing but out of every report and stat.
    { title: 'In reports', key: 'hidden_from_reports', align: 'center', sortable: false },
    { title: 'Last Seen', key: 'last_seen_at', align: 'start', sortable: false },
    { title: '', key: 'actions', align: 'end', sortable: false },
  ])

  const listLoading = ref(false)

  async function load () {
    listLoading.value = true
    try {
      await fetchBranches(true)
    } catch (error) {
      notify.error(error.message)
    } finally {
      listLoading.value = false
    }
  }

  onMounted(load)

  const createDialog = ref(false)
  const createName = ref('')
  const createCode = ref('')
  const creating = ref(false)

  function openCreate () {
    createName.value = ''
    createCode.value = ''
    createDialog.value = true
  }

  const keyDialog = ref(false)
  const revealedKey = ref('')
  const revealedCode = ref('')

  function revealKey (code, apiKey) {
    revealedCode.value = code
    revealedKey.value = apiKey
    notify.warning('This key is shown once. Copy it into the till\'s kashi-pos settings now — it cannot be retrieved again.')
    keyDialog.value = true
  }

  async function submitCreate () {
    creating.value = true
    try {
      const { register, api_key: apiKey } = await createBranch({ name: createName.value, code: createCode.value })
      createDialog.value = false
      revealKey(register.code, apiKey)
    } catch (error) {
      notify.error(error.message)
    } finally {
      creating.value = false
    }
  }

  async function copyKey () {
    try {
      await navigator.clipboard.writeText(revealedKey.value)
      notify.success('Copied to clipboard')
    } catch {
      notify.error('Failed to copy to clipboard')
    }
  }

  async function toggleActive (item, value) {
    try {
      await setBranchActive(item.id, value)
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function toggleInReports (item, value) {
    try {
      await setBranchHiddenFromReports(item.id, !value)
    } catch (error) {
      notify.error(error.message)
    }
  }

  const targetsDialog = ref(false)
  const targetsTab = ref('targets')
  const targetsBranch = ref(null)
  const targetsList = ref([])
  const targetsLoading = ref(false)

  async function openTargets (item) {
    targetsBranch.value = item
    targetsTab.value = 'targets'
    targetsDialog.value = true
    await Promise.all([loadTargets(), loadSeries()])
  }

  async function loadTargets () {
    targetsLoading.value = true
    try {
      targetsList.value = await listBranchTargets(targetsBranch.value.id)
    } catch (error) {
      notify.error(error.message)
    } finally {
      targetsLoading.value = false
    }
  }

  const targetFormDialog = ref(false)
  const targetSaving = ref(false)
  const targetForm = ref({ dateFrom: '', dateTo: '', targetAmount: null, autoColor: true, color: '#4C6EF5' })
  const editingTargetId = ref(null)

  function openAddTarget () {
    editingTargetId.value = null
    targetForm.value = { dateFrom: '', dateTo: '', targetAmount: null, autoColor: true, color: '#4C6EF5' }
    targetFormDialog.value = true
  }

  function openEditTarget (target) {
    editingTargetId.value = target.id
    targetForm.value = {
      dateFrom: target.date_from.slice(0, 10),
      dateTo: target.date_to.slice(0, 10),
      targetAmount: target.target_amount / 100,
      autoColor: true,
      color: target.color || '#4C6EF5',
    }
    targetFormDialog.value = true
  }

  async function submitTarget () {
    targetSaving.value = true
    try {
      const payload = {
        dateFrom: targetForm.value.dateFrom,
        dateTo: targetForm.value.dateTo,
        targetAmount: Math.round(Number(targetForm.value.targetAmount) * 100),
        color: targetForm.value.autoColor ? '' : targetForm.value.color,
      }
      await (editingTargetId.value ? updateBranchTarget(editingTargetId.value, payload) : createBranchTarget(targetsBranch.value.id, payload))
      targetFormDialog.value = false
      await loadTargets()
    } catch (error) {
      notify.error(error.message)
    } finally {
      targetSaving.value = false
    }
  }

  async function removeTarget (target) {
    try {
      await deleteBranchTarget(target.id)
      await loadTargets()
    } catch (error) {
      notify.error(error.message)
    }
  }

  const seriesList = ref([])

  async function loadSeries () {
    try {
      seriesList.value = await listBranchTargetSeries(targetsBranch.value.id)
    } catch (error) {
      notify.error(error.message)
    }
  }

  function describeSeries (s) {
    const every = s.interval_count === 1 ? 'month' : `${s.interval_count} months`
    return `Every ${every}, starting day ${s.start_day}`
  }

  const seriesFormDialog = ref(false)
  const seriesSaving = ref(false)
  const seriesForm = ref({ targetAmount: null, startDay: 1, intervalCount: 1, autoColor: true, color: '#4C6EF5' })
  const editingSeriesId = ref(null)

  function openAddSeries () {
    editingSeriesId.value = null
    seriesForm.value = { targetAmount: null, startDay: 1, intervalCount: 1, autoColor: true, color: '#4C6EF5' }
    seriesFormDialog.value = true
  }

  function openEditSeries (s) {
    editingSeriesId.value = s.id
    seriesForm.value = {
      targetAmount: s.target_amount / 100,
      startDay: s.start_day,
      intervalCount: s.interval_count,
      autoColor: true,
      color: s.color || '#4C6EF5',
    }
    seriesFormDialog.value = true
  }

  async function submitSeries () {
    seriesSaving.value = true
    try {
      const payload = {
        targetAmount: Math.round(Number(seriesForm.value.targetAmount) * 100),
        startDay: seriesForm.value.startDay,
        intervalCount: seriesForm.value.intervalCount,
        color: seriesForm.value.autoColor ? '' : seriesForm.value.color,
      }
      await (editingSeriesId.value ? updateBranchTargetSeries(editingSeriesId.value, payload) : createBranchTargetSeries(targetsBranch.value.id, payload))
      seriesFormDialog.value = false
      await Promise.all([loadSeries(), loadTargets()])
    } catch (error) {
      notify.error(error.message)
    } finally {
      seriesSaving.value = false
    }
  }

  async function toggleSeriesActive (s, value) {
    try {
      await setBranchTargetSeriesActive(s.id, value)
      await loadSeries()
    } catch (error) {
      notify.error(error.message)
    }
  }

  const deleteSeriesDialog = ref(false)
  const deleteSeriesSaving = ref(false)
  const deleteSeriesTarget = ref(null)

  function openDeleteSeriesConfirm (s) {
    deleteSeriesTarget.value = s
    deleteSeriesDialog.value = true
  }

  async function confirmDeleteSeries () {
    deleteSeriesSaving.value = true
    try {
      await deleteBranchTargetSeries(deleteSeriesTarget.value.id)
      deleteSeriesDialog.value = false
      await Promise.all([loadSeries(), loadTargets()])
    } catch (error) {
      notify.error(error.message)
    } finally {
      deleteSeriesSaving.value = false
    }
  }

  const salespersonsDialog = ref(false)
  const salespersonsBranch = ref(null)
  const salespersonsList = ref([])
  const salespersonsLoading = ref(false)
  const salespersonSaving = ref(false)
  const newSalespersonName = ref('')
  const newSalespersonAttendanceId = ref('')
  const newSalespersonSells = ref(true)
  const editingSalespersonId = ref(null)
  const editingSalespersonName = ref('')
  const editingSalespersonAttendanceId = ref('')

  async function openSalespersons (item) {
    salespersonsBranch.value = item
    salespersonsDialog.value = true
    newSalespersonName.value = ''
    editingSalespersonId.value = null
    await loadSalespersons()
  }

  async function loadSalespersons () {
    salespersonsLoading.value = true
    try {
      salespersonsList.value = await listBranchSalespersons(salespersonsBranch.value.id)
    } catch (error) {
      notify.error(error.message)
    } finally {
      salespersonsLoading.value = false
    }
  }

  async function submitNewSalesperson () {
    if (!newSalespersonName.value.trim()) return
    salespersonSaving.value = true
    try {
      await createBranchSalesperson(salespersonsBranch.value.id, newSalespersonName.value.trim(), newSalespersonAttendanceId.value.trim(), newSalespersonSells.value)
      newSalespersonName.value = ''
      newSalespersonAttendanceId.value = ''
      newSalespersonSells.value = true
      await loadSalespersons()
    } catch (error) {
      notify.error(error.message)
    } finally {
      salespersonSaving.value = false
    }
  }

  function openEditSalesperson (person) {
    editingSalespersonId.value = person.id
    editingSalespersonName.value = person.name
    editingSalespersonAttendanceId.value = person.attendance_user_id || ''
  }

  async function submitEditSalesperson (person) {
    const name = editingSalespersonName.value.trim()
    const attendanceId = editingSalespersonAttendanceId.value.trim()
    if (!name) return
    try {
      if (name !== person.name) await updateBranchSalesperson(person.id, name)
      if (attendanceId !== (person.attendance_user_id || '')) {
        await setBranchSalespersonAttendanceId(person.id, attendanceId)
      }
      editingSalespersonId.value = null
      await loadSalespersons()
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function changeSalespersonColor (person, color) {
    try {
      await setBranchSalespersonColor(person.id, color)
      await loadSalespersons()
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function toggleSalespersonSells (person, value) {
    try {
      await setBranchSalespersonIsSalesperson(person.id, value)
      await loadSalespersons()
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function toggleSalespersonActive (person, value) {
    try {
      await setBranchSalespersonActive(person.id, value)
      await loadSalespersons()
    } catch (error) {
      notify.error(error.message)
    }
  }

  const registersDialog = ref(false)
  const registersBranch = ref(null)
  const registers = ref([])
  const registersLoading = ref(false)
  const connectedRegisterIds = ref([])
  const registerCode = ref('')
  const registerName = ref('')
  const addingRegister = ref(false)

  async function loadRegisters () {
    registersLoading.value = true
    try {
      registers.value = await listRegisters(registersBranch.value.id)
    } catch (error) {
      notify.error(error.message)
    } finally {
      registersLoading.value = false
    }
  }

  async function openRegisters (branch) {
    registersBranch.value = branch
    registers.value = []
    registerCode.value = ''
    registerName.value = ''
    registersDialog.value = true
    await loadRegisters()
    try {
      connectedRegisterIds.value = await fetchConnectedRegisters()
    } catch {
      // Informational only -- the WS push keeps it current from here on.
    }
  }

  let unsubscribe = null
  onMounted(() => {
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (message.type === 'branch_connection_changed') {
        connectedRegisterIds.value = message.register_ids ?? []
      }
    })
  })
  onUnmounted(() => {
    unsubscribe?.()
  })

  async function submitRegister () {
    addingRegister.value = true
    try {
      const { register, api_key: apiKey } = await createRegister(registersBranch.value.id, {
        code: registerCode.value,
        name: registerName.value,
      })
      registerCode.value = ''
      registerName.value = ''
      await loadRegisters()
      revealKey(register.code, apiKey)
    } catch (error) {
      notify.error(error.message)
    } finally {
      addingRegister.value = false
    }
  }

  async function toggleRegisterActive (register, value) {
    try {
      await setRegisterActive(register.id, value)
      register.is_active = value
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function toggleRegisterInReports (register, value) {
    try {
      await setRegisterHiddenFromReports(register.id, !value)
      register.hidden_from_reports = !value
    } catch (error) {
      notify.error(error.message)
    }
  }

  const rotateDialog = ref(false)
  const rotateTarget = ref(null)
  const rotating = ref(false)

  function openRotate (register) {
    rotateTarget.value = register
    rotateDialog.value = true
  }

  async function confirmRotate () {
    rotating.value = true
    try {
      const { api_key: apiKey } = await rotateRegisterKey(rotateTarget.value.id)
      rotateDialog.value = false
      revealKey(rotateTarget.value.code, apiKey)
    } catch (error) {
      notify.error(error.message)
    } finally {
      rotating.value = false
    }
  }
</script>

<style scoped>
.register-code-field {
  max-width: 140px;
}
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.branches-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.branches-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 12px 16px;
  align-items: start;
}

.field-grid > .field-full {
  grid-column: 1 / -1;
}

.salesperson-color-input {
  width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.2);
  border-radius: 50%;
  background: none;
  cursor: pointer;
  overflow: hidden;
}

.salesperson-color-input::-webkit-color-swatch-wrapper {
  padding: 0;
}

.salesperson-color-input::-webkit-color-swatch {
  border: none;
  border-radius: 50%;
}

.salesperson-color-input:disabled {
  cursor: default;
}

.target-color-swatch {
  display: inline-block;
  width: 14px;
  height: 14px;
  border-radius: 4px;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.2);
  flex-shrink: 0;
}

.targets-dialog-card {
  height: 640px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
}

.targets-dialog-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
}

.targets-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
  color: rgba(var(--v-theme-on-surface), 0.6);
  text-align: center;
}

.target-list {
  padding: 0;
}

.target-list-item {
  border: 1px solid rgba(var(--v-theme-on-surface), 0.08);
  border-radius: 8px;
  margin-bottom: 8px;
}

.target-list-item :deep(.v-list-item__prepend) {
  margin-right: 12px;
  align-self: center;
}
</style>
