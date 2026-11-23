<template>
  <div class="page-root">
    <v-card class="au-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-update" />
        <span>{{ t('App Updates') }}</span>
        <v-spacer />
        <v-btn prepend-icon="mdi-refresh" :text="t('Reload')" variant="text" @click="reloadAll" />
      </v-card-title>
      <v-divider />

      <div class="au-content pa-5">
        <p class="text-body-2 text-medium-emphasis mb-4">
          {{ t('Publish a build from the kashi-pos repo with') }} <code>make publish</code>{{ t(', then choose which tills run it. A till downloads its update in the background and installs it when it\'s safe: when the cashier taps "Install now", or by itself while no shift is open. If the new version doesn\'t start, the till goes back to the old one on its own. To roll back on purpose, set the tills to an older release.') }}
        </p>

        <!-- ------------------------------------------------ overview -->
        <div class="au-stats mb-5">
          <div class="au-stat">
            <div class="au-stat-label">{{ t('Latest release') }}</div>
            <div class="au-stat-value">{{ latestRelease ? latestRelease.version : '—' }}</div>
            <div class="au-stat-sub">{{ latestRelease ? t('published {when}', { when: ago(latestRelease.created_at) }) : t('nothing published yet') }}</div>
          </div>
          <div class="au-stat">
            <div class="au-stat-label">{{ t('On their target') }}</div>
            <div class="au-stat-value">{{ upToDateCount }} / {{ targetedRegisters.length }}</div>
            <div class="au-stat-sub">{{ t("tills running what they're set to") }}</div>
          </div>
          <div class="au-stat">
            <div class="au-stat-label">{{ t('Updating') }}</div>
            <div class="au-stat-value" :class="{ 'text-info': updatingCount > 0 }">{{ updatingCount }}</div>
            <div class="au-stat-sub">{{ t('downloading, ready or installing') }}</div>
          </div>
          <div class="au-stat">
            <div class="au-stat-label">{{ t('Problems') }}</div>
            <div class="au-stat-value" :class="{ 'text-error': problemCount > 0 }">{{ problemCount }}</div>
            <div class="au-stat-sub">{{ t('failed, rolled back or blocked') }}</div>
          </div>
        </div>

        <!-- ------------------------------------------------ registers -->
        <div class="d-flex flex-wrap align-center ga-3 mb-2">
          <div class="au-section-title">{{ t('Tills') }}</div>
          <v-spacer />
          <v-btn
            v-if="canEdit"
            :disabled="selectedRegisterIds.length === 0"
            prepend-icon="mdi-rocket-launch-outline"
            :text="selectedRegisterIds.length > 0 ? t('Set version for {n}', { n: selectedRegisterIds.length }) : t('Set version')"
            variant="tonal"
            @click="openTargetDialog(selectedRegisterIds)"
          />
        </div>
        <v-data-table
          v-model="selectedRegisterIds"
          class="mb-6"
          density="compact"
          :headers="registerHeaders"
          item-value="id"
          :items="activeRegisters"
          :items-per-page="-1"
          :loading="loading"
          :no-data-text="t('No registers yet.')"
          :show-select="canEdit"
        >
          <template #item.register="{ item }">
            <div class="d-flex align-center">
              <span class="au-dot me-2" :class="item.connected ? 'is-online' : 'is-offline'" />
              <div>
                <div class="font-weight-medium">{{ item.branch_name }} · {{ item.code }}</div>
                <div class="text-caption text-medium-emphasis">
                  {{ item.connected ? t('online') : t('seen {when}', { when: item.last_seen_at?.Valid ? ago(item.last_seen_at.Time) : t('never') }) }}
                </div>
              </div>
            </div>
          </template>
          <template #item.app_version="{ item }">
            <span v-if="item.app_version">{{ item.app_version }}</span>
            <span v-else class="text-medium-emphasis" :title="t(&quot;This till hasn't reported a version: it runs a build from before remote updates, so it needs one manual install.&quot;)">{{ t('unknown') }}</span>
            <v-chip v-if="item.app_flavor === 'compact'" class="ms-1" size="x-small" variant="outlined">{{ t('compact') }}</v-chip>
          </template>
          <template #item.target="{ item }">
            <span v-if="item.target_version?.Valid">{{ item.target_version.String }}</span>
            <span v-else class="text-medium-emphasis">—</span>
          </template>
          <template #item.status="{ item }">
            <div class="au-status">
              <v-chip :color="stateInfo(item).color" size="small" variant="tonal">{{ stateInfo(item).label }}</v-chip>
              <v-progress-linear
                v-if="item.update_state === 'downloading'"
                class="mt-1"
                color="info"
                height="4"
                :model-value="item.update_progress"
                rounded
              />
              <div v-if="item.update_detail && showDetail(item)" class="text-caption text-medium-emphasis au-detail" :title="item.update_detail">
                {{ item.update_detail }}
              </div>
            </div>
          </template>
          <template v-if="canEdit" #item.actions="{ item }">
            <v-btn size="small" :text="t('Set version')" variant="text" @click="openTargetDialog([item.id])" />
            <v-btn
              v-if="previousReleaseFor(item)"
              prepend-icon="mdi-backup-restore"
              size="small"
              :text="t('Roll back to {version}', { version: previousReleaseFor(item).version })"
              variant="text"
              @click="confirmRollback(item)"
            />
          </template>
        </v-data-table>

        <!-- ------------------------------------------------ releases -->
        <div class="mb-2 d-flex align-center ga-3 flex-wrap">
          <div class="au-section-title">{{ t('Releases') }}</div>
          <v-spacer />
          <v-select
            v-model="keep"
            class="au-keep"
            density="compact"
            :disabled="!canEdit"
            hide-details
            :items="KEEP_OPTIONS"
            :label="t('Keep old builds')"
            variant="outlined"
            @update:model-value="saveKeep"
          />
        </div>
        <v-data-table
          class="mb-6"
          density="compact"
          :headers="releaseHeaders"
          :items="releases"
          :items-per-page="-1"
          :loading="loading"
          :no-data-text="t('Nothing published yet. In the kashi-pos repo run: git tag v1.2.3 && make publish NOTES=&quot;what changed&quot;')"
        >
          <template #item.version="{ item }">
            <span class="font-weight-medium">{{ item.version }}</span>
            <v-chip class="ms-1" size="x-small" variant="outlined">{{ item.flavor }}</v-chip>
          </template>
          <template #item.notes="{ item }">
            <div class="au-notes" :title="item.notes">{{ item.notes || '—' }}</div>
          </template>
          <template #item.size_bytes="{ item }">
            <span v-if="item.file_removed" class="text-medium-emphasis" :title="t('The file was deleted once its tills installed it. Publish this version again to send it to other tills.')">{{ t('Deleted') }}</span>
            <template v-else>
              <!-- What a till downloads: the compressed build, or a patch
                   from the build it runs (api/app_update_compact.go). -->
              <span :title="`The program is ${formatSize(item.size_bytes)}`">
                {{ formatSize(item.encoding ? item.download_bytes : item.size_bytes) }}
              </span>
              <div v-if="item.patches" class="text-caption text-medium-emphasis" :title="t('Tills on a build this was patched from download only the difference')">
                {{ t('patch') }} {{ formatSize(item.patch_bytes) }}{{ item.patches > 1 ? ` · ${t('{n} builds', { n: item.patches })}` : '' }}
              </div>
            </template>
          </template>
          <template #item.created_at="{ item }">
            <span :title="formatDate(item.created_at)">{{ ago(item.created_at) }}</span>
          </template>
          <template #item.usage="{ item }">
            {{ t('{running} running', { running: item.running_count }) }} · {{ t('{set} set', { set: item.target_count }) }}
          </template>
          <template v-if="canEdit" #item.actions="{ item }">
            <v-btn
              :disabled="item.file_removed"
              prepend-icon="mdi-rocket-launch-outline"
              size="small"
              :text="t('Deploy')"
              :title="item.file_removed ? t('The file was deleted; publish this version again first') : undefined"
              variant="text"
              @click="openDeployDialog(item)"
            />
            <v-btn
              icon="mdi-pencil"
              size="small"
              :title="t('Edit notes')"
              variant="text"
              @click="openNotesDialog(item)"
            />
            <v-btn
              :disabled="item.target_count > 0"
              icon="mdi-delete"
              size="small"
              :title="item.target_count > 0 ? t('Tills are still set to this release') : t('Delete')"
              variant="text"
              @click="deleteRelease(item)"
            />
          </template>
        </v-data-table>

        <!-- ------------------------------------------------ history -->
        <div class="au-section-title mb-2">{{ t('History') }}</div>
        <v-data-table
          density="compact"
          :headers="eventHeaders"
          :items="events"
          :loading="loading"
          :no-data-text="t('No update activity yet.')"
        >
          <template #item.created_at="{ item }">
            <span :title="formatDate(item.created_at)">{{ ago(item.created_at) }}</span>
          </template>
          <template #item.register="{ item }">{{ item.branch_name }} · {{ item.register_code }}</template>
          <template #item.state="{ item }">
            <v-chip :color="STATES[item.state]?.color" size="small" variant="tonal">{{ stateLabel(item.state) }}</v-chip>
          </template>
          <template #item.version="{ item }">
            {{ item.app_version || '—' }}<span v-if="item.release_version?.Valid && item.release_version.String !== item.app_version"> → {{ item.release_version.String }}</span>
          </template>
          <template #item.detail="{ item }">
            <span class="text-medium-emphasis">{{ item.detail }}</span>
          </template>
        </v-data-table>
      </div>
    </v-card>

    <!-- set the version for some tills -->
    <v-dialog v-model="targetDialog" max-width="480">
      <v-card>
        <v-card-title class="text-subtitle-1">{{ t('Set the version for {n} till(s)', { n: targetRegisterIds.length }) }}</v-card-title>
        <v-card-text>
          <v-select
            v-model="targetReleaseId"
            density="compact"
            item-title="label"
            item-value="id"
            :items="targetOptions"
            :label="t('Version')"
            variant="outlined"
          />
          <v-checkbox
            v-model="forceInstall"
            color="error"
            density="compact"
            :disabled="!targetReleaseId"
            hide-details
            :label="t('Install now, even during a shift')"
          />
          <p v-if="forceInstall" class="text-caption text-error mb-2">
            {{ t('The cashier gets a one-minute countdown to finish the sale, then the till closes and updates.') }}
          </p>
          <p class="text-body-2 text-medium-emphasis">
            {{ t("Each till downloads it right away (or when it's next online) and installs it at a safe moment. \"Keep what it runs\" stops any update that hasn't installed yet.") }}
          </p>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn :text="t('Cancel')" variant="text" @click="targetDialog = false" />
          <v-btn
            color="primary"
            :loading="saving"
            :text="t('Save')"
            variant="flat"
            @click="submitTarget"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- deploy one release -->
    <v-dialog v-model="deployDialog" max-width="520">
      <v-card v-if="deployRelease">
        <v-card-title class="text-subtitle-1">{{ t('Deploy {version} ({flavor})', { version: deployRelease.version, flavor: deployRelease.flavor }) }}</v-card-title>
        <v-card-text>
          <v-checkbox
            v-for="r in deployCandidates"
            :key="r.id"
            v-model="deployRegisterIds"
            density="compact"
            hide-details
            :label="`${r.branch_name} · ${r.code}${r.app_version ? ` ${t('(runs {version})', { version: r.app_version })}` : ''}`"
            :value="r.id"
          />
          <p v-if="deployCandidates.length === 0" class="text-body-2">
            {{ t('No till runs the {flavor} build.', { flavor: deployRelease.flavor }) }}
          </p>
          <v-checkbox
            v-model="forceInstall"
            class="mt-2"
            color="error"
            density="compact"
            hide-details
            :label="t('Install now, even during a shift')"
          />
          <p v-if="forceInstall" class="text-caption text-error mb-2">
            {{ t('The cashier gets a one-minute countdown to finish the sale, then the till closes and updates.') }}
          </p>
        </v-card-text>
        <v-card-actions>
          <v-btn
            v-if="deployCandidates.length > 0"
            :text="deployRegisterIds.length === deployCandidates.length ? t('None') : t('All')"
            variant="text"
            @click="toggleAllDeploy"
          />
          <v-spacer />
          <v-btn :text="t('Cancel')" variant="text" @click="deployDialog = false" />
          <v-btn
            color="primary"
            :disabled="deployRegisterIds.length === 0"
            :loading="saving"
            :text="t('Deploy to {n}', { n: deployRegisterIds.length })"
            variant="flat"
            @click="submitDeploy"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- edit release notes -->
    <v-dialog v-model="notesDialog" max-width="520">
      <v-card v-if="notesRelease">
        <v-card-title class="text-subtitle-1">{{ t("What's new in {version}", { version: notesRelease.version }) }}</v-card-title>
        <v-card-text>
          <v-textarea
            v-model="notesText"
            auto-grow
            :hint="t('Cashiers see this after the update installs.')"
            persistent-hint
            rows="4"
            variant="outlined"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn :text="t('Cancel')" variant="text" @click="notesDialog = false" />
          <v-btn
            color="primary"
            :loading="saving"
            :text="t('Save')"
            variant="flat"
            @click="submitNotes"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { t } from '@/i18n'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useAppUpdates } from '@/composables/useAppUpdates'
  import { confirmDialog } from '@/composables/useDialog'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const api = useAppUpdates()
  const socket = useAdminSocket()

  // label is a function so the text follows the language toggle.
  const STATES = {
    up_to_date: { label: () => t('Up to date'), color: 'success' },
    downloading: { label: () => t('Downloading'), color: 'info' },
    ready: { label: () => t('Ready to install'), color: 'info' },
    installing: { label: () => t('Installing'), color: 'info' },
    updated: { label: () => t('Updated'), color: 'success' },
    failed: { label: () => t('Failed'), color: 'error' },
    rolled_back: { label: () => t('Rolled back'), color: 'warning' },
    blocked: { label: () => t('Needs manual install'), color: 'error' },
    targeted: { label: () => t('Version set'), color: undefined },
    target_cleared: { label: () => t('Target cleared'), color: undefined },
  }
  function stateLabel (key) {
    return STATES[key] ? STATES[key].label() : key
  }
  const PROBLEM_STATES = new Set(['failed', 'rolled_back', 'blocked'])
  const UPDATING_STATES = new Set(['downloading', 'ready', 'installing'])

  const loading = ref(false)
  const saving = ref(false)
  const releases = ref([])
  // Old build files kept on the server: 0 none (each goes once its tills
  // installed it), N the newest N per flavor, -1 all. See
  // api/app_release_cleanup.go.
  const keep = ref(0)
  const KEEP_OPTIONS = computed(() => [
    { value: 0, title: t('None (delete once installed)') },
    { value: 1, title: t('Newest {n}', { n: 1 }) },
    { value: 2, title: t('Newest {n}', { n: 2 }) },
    { value: 3, title: t('Newest {n}', { n: 3 }) },
    { value: 5, title: t('Newest {n}', { n: 5 }) },
    { value: 10, title: t('Newest {n}', { n: 10 }) },
    { value: -1, title: t('All') },
  ])
  const registers = ref([])
  const events = ref([])
  const selectedRegisterIds = ref([])

  const registerHeaders = computed(() => [
    { title: t('Till'), key: 'register', sortable: false },
    { title: t('Runs'), key: 'app_version', sortable: false },
    { title: t('Set to'), key: 'target', sortable: false },
    { title: t('Status'), key: 'status', sortable: false },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ])
  const releaseHeaders = computed(() => [
    { title: t('Version'), key: 'version', sortable: false },
    { title: t("What's new"), key: 'notes', sortable: false },
    { title: t('Size'), key: 'size_bytes', sortable: false },
    { title: t('Published'), key: 'created_at', sortable: false },
    { title: t('Tills'), key: 'usage', sortable: false },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ])
  const eventHeaders = computed(() => [
    { title: t('When'), key: 'created_at', sortable: false },
    { title: t('Till'), key: 'register', sortable: false },
    { title: t('What'), key: 'state', sortable: false },
    { title: t('Version'), key: 'version', sortable: false },
    { title: t('Detail'), key: 'detail', sortable: false },
  ])

  const activeRegisters = computed(() => registers.value.filter(r => r.is_active))
  const latestRelease = computed(() => releases.value[0] ?? null)
  const targetedRegisters = computed(() => activeRegisters.value.filter(r => r.target_version?.Valid))
  const upToDateCount = computed(() => targetedRegisters.value.filter(r => r.app_version === r.target_version.String).length)
  const updatingCount = computed(() => activeRegisters.value.filter(r => UPDATING_STATES.has(r.update_state)).length)
  const problemCount = computed(() => activeRegisters.value.filter(r => PROBLEM_STATES.has(r.update_state)).length)

  // What the Status column says. A till set to a version it already runs is
  // up to date whatever it last reported; one that hasn't reported at all
  // runs a build from before remote updates.
  function stateInfo (r) {
    if (!r.app_version) return { label: t('Not reporting'), color: undefined }
    if (r.update_state === 'downloading') return { label: t('Downloading {percent}%', { percent: r.update_progress }), color: 'info' }
    if (r.target_version?.Valid && r.app_version === r.target_version.String && !PROBLEM_STATES.has(r.update_state)) {
      return { color: STATES.up_to_date.color, label: STATES.up_to_date.label() }
    }
    if (r.target_version?.Valid && r.app_version !== r.target_version.String && !r.update_state) {
      return { label: t('Waiting'), color: undefined }
    }
    const state = STATES[r.update_state]
    return state ? { color: state.color, label: state.label() } : { label: t('Up to date'), color: 'success' }
  }
  function showDetail (r) {
    return PROBLEM_STATES.has(r.update_state) || r.update_state === 'ready'
  }

  // The newest release older than what the till runs, of its flavor -- the
  // one-click rollback.
  function previousReleaseFor (r) {
    if (!r.app_version) return null
    const flavor = r.app_flavor || 'standard'
    return releases.value
      .filter(rel => !rel.file_removed && rel.flavor === flavor && compareVersions(rel.version, r.app_version) < 0)
      .toSorted((a, b) => compareVersions(b.version, a.version))[0] ?? null
  }
  function compareVersions (a, b) {
    const pa = a.split('.').map(Number)
    const pb = b.split('.').map(Number)
    for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
      const d = (pa[i] ?? 0) - (pb[i] ?? 0)
      if (d !== 0) return d
    }
    return 0
  }

  async function reloadAll () {
    loading.value = true
    try {
      const [rel, regs, evs] = await Promise.all([api.listReleases(), api.listRegisters(), api.listEvents()])
      releases.value = rel.releases
      keep.value = rel.keep
      registers.value = regs
      events.value = evs
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  // ---- set version ---------------------------------------------------------
  const targetDialog = ref(false)
  // "Install now, even during a shift" in either dialog; off each time one opens.
  const forceInstall = ref(false)
  const targetRegisterIds = ref([])
  const targetReleaseId = ref(null)
  const targetOptions = computed(() => {
    const chosen = registers.value.filter(r => targetRegisterIds.value.includes(r.id))
    const flavors = new Set(chosen.map(r => r.app_flavor).filter(Boolean))
    return [
      { id: null, label: t('Keep what it runs (no update)') },
      ...releases.value
        .filter(rel => !rel.file_removed || rel.id === targetReleaseId.value)
        .filter(rel => flavors.size === 0 || (flavors.size === 1 && flavors.has(rel.flavor)))
        .map(rel => ({ id: rel.id, label: t('{version} ({flavor})', { version: rel.version, flavor: rel.flavor }) })),
    ]
  })
  function openTargetDialog (ids) {
    targetRegisterIds.value = [...ids]
    const first = registers.value.find(r => r.id === ids[0])
    targetReleaseId.value = first?.target_release_id?.Valid ? first.target_release_id.Int64 : null
    forceInstall.value = false
    targetDialog.value = true
  }
  async function submitTarget () {
    await saveTarget(targetReleaseId.value, targetRegisterIds.value, forceInstall.value)
    targetDialog.value = false
    selectedRegisterIds.value = []
  }
  async function saveTarget (releaseId, registerIds, force = false) {
    saving.value = true
    try {
      await api.setTarget(releaseId, registerIds, force && !!releaseId)
      notify.success(releaseId ? t('Saved. The tills are fetching it now.') : t('Saved.'))
      await reloadAll()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }
  async function confirmRollback (r) {
    const prev = previousReleaseFor(r)
    if (!prev) return
    const text = t('Roll {till} back from {from} to {to}? It installs at the next safe moment, like an update.', { till: `${r.branch_name} · ${r.code}`, from: r.app_version, to: prev.version })
    if (!(await confirmDialog({ text, okText: t('Roll back') }))) return
    await saveTarget(prev.id, [r.id])
  }

  // ---- deploy one release --------------------------------------------------
  const deployDialog = ref(false)
  const deployRelease = ref(null)
  const deployRegisterIds = ref([])
  const deployCandidates = computed(() => activeRegisters.value.filter(r =>
    deployRelease.value && (r.app_flavor || 'standard') === deployRelease.value.flavor))
  function openDeployDialog (release) {
    deployRelease.value = release
    deployRegisterIds.value = deployCandidates.value.map(r => r.id)
    forceInstall.value = false
    deployDialog.value = true
  }
  function toggleAllDeploy () {
    deployRegisterIds.value = deployRegisterIds.value.length === deployCandidates.value.length
      ? []
      : deployCandidates.value.map(r => r.id)
  }
  async function submitDeploy () {
    await saveTarget(deployRelease.value.id, deployRegisterIds.value, forceInstall.value)
    deployDialog.value = false
  }

  // ---- notes and delete ----------------------------------------------------
  const notesDialog = ref(false)
  const notesRelease = ref(null)
  const notesText = ref('')
  function openNotesDialog (release) {
    notesRelease.value = release
    notesText.value = release.notes
    notesDialog.value = true
  }
  async function submitNotes () {
    saving.value = true
    try {
      await api.updateNotes(notesRelease.value.id, notesText.value)
      notesDialog.value = false
      await reloadAll()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }
  async function saveKeep (value) {
    try {
      await api.setKeep(value)
      notify.success(value === 0 ? t('Build files are deleted once their tills install them') : t('Saved'))
    } catch (error) {
      notify.error(error.message)
      reloadAll()
    }
  }
  async function deleteRelease (release) {
    const text = t('Delete release {version} ({flavor})? Tills that already run it keep it.', { version: release.version, flavor: release.flavor })
    if (!(await confirmDialog({ text, okText: t('Delete'), danger: true }))) return
    try {
      await api.deleteRelease(release.id)
      await reloadAll()
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- formatting ----------------------------------------------------------
  function formatDate (value) {
    return value ? new Date(value).toLocaleString() : '—'
  }
  function ago (value) {
    const seconds = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 1000))
    if (seconds < 60) return t('just now')
    if (seconds < 3600) return t('{n} min ago', { n: Math.floor(seconds / 60) })
    if (seconds < 86_400) return t('{n} h ago', { n: Math.floor(seconds / 3600) })
    return t('{n} d ago', { n: Math.floor(seconds / 86_400) })
  }
  function formatSize (bytes) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  // Tills report progress as they go; refetch on each push (debounced), with
  // a slow poll as the fallback if the socket is down.
  let offSocket = null
  let poll = null
  let refetchTimer = null
  onMounted(() => {
    reloadAll()
    socket.ensureConnected()
    offSocket = socket.onMessage(message => {
      if (message.type !== 'app_update_changed' && message.type !== 'branch_connection_changed') return
      clearTimeout(refetchTimer)
      refetchTimer = setTimeout(() => {
        Promise.all([api.listReleases(), api.listRegisters(), api.listEvents()])
          .then(([rel, regs, evs]) => {
            releases.value = rel.releases
            keep.value = rel.keep
            registers.value = regs
            events.value = evs
          })
          .catch(() => {})
      }, 400)
    })
    poll = setInterval(() => {
      api.listRegisters().then(regs => (registers.value = regs)).catch(() => {})
    }, 30_000)
  })
  onUnmounted(() => {
    offSocket?.()
    clearInterval(poll)
    clearTimeout(refetchTimer)
  })
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.au-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.au-content {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
}
.au-keep {
  max-width: 260px;
}
.au-section-title,
.au-stat-label {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.au-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}
.au-stat {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
  padding: 12px 14px;
}
.au-stat-value {
  font-size: 1.6rem;
  font-weight: 600;
  line-height: 1.3;
}
.au-stat-sub {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.au-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  display: inline-block;
  flex-shrink: 0;
}
.au-dot.is-online {
  background: rgb(var(--v-theme-success));
}
.au-dot.is-offline {
  background: rgba(128, 128, 128, 0.5);
}
.au-status {
  min-width: 140px;
  padding: 4px 0;
}
.au-detail,
.au-notes {
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
