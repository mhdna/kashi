<template>
  <div class="page-root">
    <v-card class="au-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-update" />
        <span>App Updates</span>
        <v-spacer />
        <v-btn prepend-icon="mdi-refresh" text="Reload" variant="text" @click="reloadAll" />
      </v-card-title>
      <v-divider />

      <div class="au-content pa-5">
        <p class="text-body-2 text-medium-emphasis mb-4">
          Publish a build from the kashi-pos repo with <code>make publish</code>, then choose which
          tills run it. A till downloads its update in the background and installs it when it's
          safe: when the cashier taps "Install now", or by itself while no shift is open. If the new
          version doesn't start, the till goes back to the old one on its own. To roll back on
          purpose, set the tills to an older release.
        </p>

        <!-- ------------------------------------------------ overview -->
        <div class="au-stats mb-5">
          <div class="au-stat">
            <div class="au-stat-label">Latest release</div>
            <div class="au-stat-value">{{ latestRelease ? latestRelease.version : '—' }}</div>
            <div class="au-stat-sub">{{ latestRelease ? `published ${ago(latestRelease.created_at)}` : 'nothing published yet' }}</div>
          </div>
          <div class="au-stat">
            <div class="au-stat-label">On their target</div>
            <div class="au-stat-value">{{ upToDateCount }} / {{ targetedRegisters.length }}</div>
            <div class="au-stat-sub">tills running what they're set to</div>
          </div>
          <div class="au-stat">
            <div class="au-stat-label">Updating</div>
            <div class="au-stat-value" :class="{ 'text-info': updatingCount > 0 }">{{ updatingCount }}</div>
            <div class="au-stat-sub">downloading, ready or installing</div>
          </div>
          <div class="au-stat">
            <div class="au-stat-label">Problems</div>
            <div class="au-stat-value" :class="{ 'text-error': problemCount > 0 }">{{ problemCount }}</div>
            <div class="au-stat-sub">failed, rolled back or blocked</div>
          </div>
        </div>

        <!-- ------------------------------------------------ registers -->
        <div class="d-flex flex-wrap align-center ga-3 mb-2">
          <div class="au-section-title">Tills</div>
          <v-spacer />
          <v-btn
            v-if="canEdit"
            :disabled="selectedRegisterIds.length === 0"
            prepend-icon="mdi-rocket-launch-outline"
            :text="selectedRegisterIds.length > 0 ? `Set version for ${selectedRegisterIds.length}` : 'Set version'"
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
          no-data-text="No registers yet."
          :show-select="canEdit"
        >
          <template #item.register="{ item }">
            <div class="d-flex align-center">
              <span class="au-dot me-2" :class="item.connected ? 'is-online' : 'is-offline'" />
              <div>
                <div class="font-weight-medium">{{ item.branch_name }} · {{ item.code }}</div>
                <div class="text-caption text-medium-emphasis">
                  {{ item.connected ? 'online' : `seen ${item.last_seen_at?.Valid ? ago(item.last_seen_at.Time) : 'never'}` }}
                </div>
              </div>
            </div>
          </template>
          <template #item.app_version="{ item }">
            <span v-if="item.app_version">{{ item.app_version }}</span>
            <span v-else class="text-medium-emphasis" title="This till hasn't reported a version: it runs a build from before remote updates, so it needs one manual install.">unknown</span>
            <v-chip v-if="item.app_flavor === 'compact'" class="ms-1" size="x-small" variant="outlined">compact</v-chip>
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
            <v-btn size="small" text="Set version" variant="text" @click="openTargetDialog([item.id])" />
            <v-btn
              v-if="previousReleaseFor(item)"
              prepend-icon="mdi-backup-restore"
              size="small"
              :text="`Roll back to ${previousReleaseFor(item).version}`"
              variant="text"
              @click="confirmRollback(item)"
            />
          </template>
        </v-data-table>

        <!-- ------------------------------------------------ releases -->
        <div class="au-section-title mb-2">Releases</div>
        <v-data-table
          class="mb-6"
          density="compact"
          :headers="releaseHeaders"
          :items="releases"
          :items-per-page="-1"
          :loading="loading"
          no-data-text="Nothing published yet. In the kashi-pos repo run: git tag v1.2.3 && make publish NOTES=&quot;what changed&quot;"
        >
          <template #item.version="{ item }">
            <span class="font-weight-medium">{{ item.version }}</span>
            <v-chip class="ms-1" size="x-small" variant="outlined">{{ item.flavor }}</v-chip>
          </template>
          <template #item.notes="{ item }">
            <div class="au-notes" :title="item.notes">{{ item.notes || '—' }}</div>
          </template>
          <template #item.size_bytes="{ item }">{{ formatSize(item.size_bytes) }}</template>
          <template #item.created_at="{ item }">
            <span :title="formatDate(item.created_at)">{{ ago(item.created_at) }}</span>
          </template>
          <template #item.usage="{ item }">
            {{ item.running_count }} running · {{ item.target_count }} set
          </template>
          <template v-if="canEdit" #item.actions="{ item }">
            <v-btn
              prepend-icon="mdi-rocket-launch-outline"
              size="small"
              text="Deploy"
              variant="text"
              @click="openDeployDialog(item)"
            />
            <v-btn
              icon="mdi-pencil"
              size="small"
              title="Edit notes"
              variant="text"
              @click="openNotesDialog(item)"
            />
            <v-btn
              :disabled="item.target_count > 0"
              icon="mdi-delete"
              size="small"
              :title="item.target_count > 0 ? 'Tills are still set to this release' : 'Delete'"
              variant="text"
              @click="deleteRelease(item)"
            />
          </template>
        </v-data-table>

        <!-- ------------------------------------------------ history -->
        <div class="au-section-title mb-2">History</div>
        <v-data-table
          density="compact"
          :headers="eventHeaders"
          :items="events"
          :items-per-page="25"
          :loading="loading"
          no-data-text="No update activity yet."
        >
          <template #item.created_at="{ item }">
            <span :title="formatDate(item.created_at)">{{ ago(item.created_at) }}</span>
          </template>
          <template #item.register="{ item }">{{ item.branch_name }} · {{ item.register_code }}</template>
          <template #item.state="{ item }">
            <v-chip :color="STATES[item.state]?.color" size="small" variant="tonal">{{ STATES[item.state]?.label ?? item.state }}</v-chip>
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
        <v-card-title class="text-subtitle-1">Set the version for {{ targetRegisterIds.length }} till(s)</v-card-title>
        <v-card-text>
          <v-select
            v-model="targetReleaseId"
            density="compact"
            item-title="label"
            item-value="id"
            :items="targetOptions"
            label="Version"
            variant="outlined"
          />
          <v-checkbox
            v-model="forceInstall"
            color="error"
            density="compact"
            :disabled="!targetReleaseId"
            hide-details
            label="Install now, even during a shift"
          />
          <p v-if="forceInstall" class="text-caption text-error mb-2">
            The cashier gets a one-minute countdown to finish the sale, then the till closes and updates.
          </p>
          <p class="text-body-2 text-medium-emphasis">
            Each till downloads it right away (or when it's next online) and installs it at a safe
            moment. "Keep what it runs" stops any update that hasn't installed yet.
          </p>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="targetDialog = false" />
          <v-btn
            color="primary"
            :loading="saving"
            text="Save"
            variant="flat"
            @click="submitTarget"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- deploy one release -->
    <v-dialog v-model="deployDialog" max-width="520">
      <v-card v-if="deployRelease">
        <v-card-title class="text-subtitle-1">Deploy {{ deployRelease.version }} ({{ deployRelease.flavor }})</v-card-title>
        <v-card-text>
          <v-checkbox
            v-for="r in deployCandidates"
            :key="r.id"
            v-model="deployRegisterIds"
            density="compact"
            hide-details
            :label="`${r.branch_name} · ${r.code}${r.app_version ? ` (runs ${r.app_version})` : ''}`"
            :value="r.id"
          />
          <p v-if="deployCandidates.length === 0" class="text-body-2">
            No till runs the {{ deployRelease.flavor }} build.
          </p>
          <v-checkbox
            v-model="forceInstall"
            class="mt-2"
            color="error"
            density="compact"
            hide-details
            label="Install now, even during a shift"
          />
          <p v-if="forceInstall" class="text-caption text-error mb-2">
            The cashier gets a one-minute countdown to finish the sale, then the till closes and updates.
          </p>
        </v-card-text>
        <v-card-actions>
          <v-btn
            v-if="deployCandidates.length > 0"
            :text="deployRegisterIds.length === deployCandidates.length ? 'None' : 'All'"
            variant="text"
            @click="toggleAllDeploy"
          />
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="deployDialog = false" />
          <v-btn
            color="primary"
            :disabled="deployRegisterIds.length === 0"
            :loading="saving"
            :text="`Deploy to ${deployRegisterIds.length}`"
            variant="flat"
            @click="submitDeploy"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- edit release notes -->
    <v-dialog v-model="notesDialog" max-width="520">
      <v-card v-if="notesRelease">
        <v-card-title class="text-subtitle-1">What's new in {{ notesRelease.version }}</v-card-title>
        <v-card-text>
          <v-textarea
            v-model="notesText"
            auto-grow
            hint="Cashiers see this after the update installs."
            persistent-hint
            rows="4"
            variant="outlined"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="notesDialog = false" />
          <v-btn
            color="primary"
            :loading="saving"
            text="Save"
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
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useAppUpdates } from '@/composables/useAppUpdates'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const api = useAppUpdates()
  const socket = useAdminSocket()

  const STATES = {
    up_to_date: { label: 'Up to date', color: 'success' },
    downloading: { label: 'Downloading', color: 'info' },
    ready: { label: 'Ready to install', color: 'info' },
    installing: { label: 'Installing', color: 'info' },
    updated: { label: 'Updated', color: 'success' },
    failed: { label: 'Failed', color: 'error' },
    rolled_back: { label: 'Rolled back', color: 'warning' },
    blocked: { label: 'Needs manual install', color: 'error' },
    targeted: { label: 'Version set', color: undefined },
    target_cleared: { label: 'Target cleared', color: undefined },
  }
  const PROBLEM_STATES = new Set(['failed', 'rolled_back', 'blocked'])
  const UPDATING_STATES = new Set(['downloading', 'ready', 'installing'])

  const loading = ref(false)
  const saving = ref(false)
  const releases = ref([])
  const registers = ref([])
  const events = ref([])
  const selectedRegisterIds = ref([])

  const registerHeaders = [
    { title: 'Till', key: 'register', sortable: false },
    { title: 'Runs', key: 'app_version', sortable: false },
    { title: 'Set to', key: 'target', sortable: false },
    { title: 'Status', key: 'status', sortable: false },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ]
  const releaseHeaders = [
    { title: 'Version', key: 'version', sortable: false },
    { title: "What's new", key: 'notes', sortable: false },
    { title: 'Size', key: 'size_bytes', sortable: false },
    { title: 'Published', key: 'created_at', sortable: false },
    { title: 'Tills', key: 'usage', sortable: false },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ]
  const eventHeaders = [
    { title: 'When', key: 'created_at', sortable: false },
    { title: 'Till', key: 'register', sortable: false },
    { title: 'What', key: 'state', sortable: false },
    { title: 'Version', key: 'version', sortable: false },
    { title: 'Detail', key: 'detail', sortable: false },
  ]

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
    if (!r.app_version) return { label: 'Not reporting', color: undefined }
    if (r.update_state === 'downloading') return { label: `Downloading ${r.update_progress}%`, color: 'info' }
    if (r.target_version?.Valid && r.app_version === r.target_version.String && !PROBLEM_STATES.has(r.update_state)) {
      return STATES.up_to_date
    }
    if (r.target_version?.Valid && r.app_version !== r.target_version.String && !r.update_state) {
      return { label: 'Waiting', color: undefined }
    }
    return STATES[r.update_state] ?? { label: 'Up to date', color: 'success' }
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
      .filter(rel => rel.flavor === flavor && compareVersions(rel.version, r.app_version) < 0)
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
      releases.value = rel
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
      { id: null, label: 'Keep what it runs (no update)' },
      ...releases.value
        .filter(rel => flavors.size === 0 || (flavors.size === 1 && flavors.has(rel.flavor)))
        .map(rel => ({ id: rel.id, label: `${rel.version} (${rel.flavor})` })),
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
      notify.success(releaseId ? 'Saved. The tills are fetching it now.' : 'Saved.')
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
    if (!window.confirm(`Roll ${r.branch_name} · ${r.code} back from ${r.app_version} to ${prev.version}? It installs at the next safe moment, like an update.`)) return
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
  async function deleteRelease (release) {
    if (!window.confirm(`Delete release ${release.version} (${release.flavor})? Tills that already run it keep it.`)) return
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
    if (seconds < 60) return 'just now'
    if (seconds < 3600) return `${Math.floor(seconds / 60)} min ago`
    if (seconds < 86_400) return `${Math.floor(seconds / 3600)} h ago`
    return `${Math.floor(seconds / 86_400)} d ago`
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
            releases.value = rel
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
