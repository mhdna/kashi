<template>
  <div class="page-root">
    <v-card class="bk-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-source-branch-sync" />
        <span>{{ t('Conflicts') }}</span>
        <v-spacer />
        <v-btn
          v-if="canEdit && selectedConflicts.length > 0"
          color="primary"
          variant="tonal"
          @click="bulkTakeBranch"
        >
          {{ t('Take branch spelling ({n})', { n: selectedConflicts.length }) }}
        </v-btn>
        <v-btn prepend-icon="mdi-refresh" :text="t('Reload')" variant="text" @click="reload" />
      </v-card-title>
      <v-divider />

      <div class="bk-layout">
        <PageSidebar
          v-model="selectedId"
          :all-title="t('All branches')"
          :empty-text="t('No branches yet.')"
          :items="branchItems"
          :loading="branchesLoading"
          show-all
        >
          <template #append="{ item }">
            <v-badge
              v-if="summaryFor(item.id).open_oversold > 0"
              color="error"
              :content="summaryFor(item.id).open_oversold"
              inline
            />
            <v-badge
              v-if="summaryFor(item.id).open - summaryFor(item.id).open_oversold > 0"
              color="warning"
              :content="summaryFor(item.id).open - summaryFor(item.id).open_oversold"
              inline
            />
          </template>
        </PageSidebar>

        <v-divider vertical />

        <div class="bk-content pa-5">
          <p class="text-body-2 text-medium-emphasis mb-4">
            {{ t('Things a till sent that kashi couldn\'t apply on its own: a client whose phone already belongs to someone with another name, a client with no usable phone, or a sale that took a product below zero stock. Red means stock went negative and needs fixing; yellow is everything else.') }}
          </p>

          <div class="bk-stats mb-4">
            <div class="bk-stat">
              <div class="bk-stat-label">{{ t('Open') }}</div>
              <div class="bk-stat-value" :class="{ 'text-warning': totals.open > 0 }">{{ totals.open }}</div>
              <div class="bk-stat-sub">{{ totals.oldestOpenAt ? t('oldest {time}', { time: ago(totals.oldestOpenAt) }) : t('nothing to review') }}</div>
            </div>
            <div class="bk-stat">
              <div class="bk-stat-label">{{ t('Oversold stock') }}</div>
              <div class="bk-stat-value" :class="{ 'text-error': totals.oversold > 0 }">{{ totals.oversold }}</div>
              <div class="bk-stat-sub">{{ t('fix with an adjustment, transfer or count') }}</div>
            </div>
            <div class="bk-stat">
              <div class="bk-stat-label">{{ t('Client conflicts') }}</div>
              <div class="bk-stat-value">{{ totals.open - totals.oversold }}</div>
              <div class="bk-stat-sub">{{ t('names and phones to reconcile') }}</div>
            </div>
          </div>

          <div class="d-flex flex-wrap align-center ga-3 mb-2">
            <v-chip-group v-model="statusFilter" color="primary" mandatory>
              <v-chip
                v-for="s in STATUS_FILTERS"
                :key="s.value"
                size="small"
                :value="s.value"
                variant="outlined"
              >
                {{ s.title }}
              </v-chip>
            </v-chip-group>
            <v-spacer />
            <v-select
              v-model="entityFilter"
              clearable
              density="compact"
              hide-details
              :items="ENTITY_FILTERS"
              :placeholder="t('Any kind')"
              style="max-width: 180px"
              variant="outlined"
            />
            <v-text-field
              v-model="search"
              clearable
              density="compact"
              hide-details
              :placeholder="t('Search ref, name, phone')"
              prepend-inner-icon="mdi-magnify"
              style="max-width: 240px"
              variant="outlined"
            />
          </div>

          <ServerSideTable
            ref="tableRef"
            :api-u-r-l="`${API_BASE}/branch_sync_conflicts`"
            density="compact"
            expandable
            :external-search="search ?? ''"
            :headers="headers"
            :query-params="queryParams"
            root-key="conflicts"
            :selectable="canEdit && statusFilter === 'open'"
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
              <v-chip :color="item.kind === 'oversold' ? 'error' : 'warning'" size="small" variant="tonal">
                {{ conflictKindLabel(item.kind) }}
              </v-chip>
            </template>
            <template #item.branch_id="{ item }">{{ branchName(item.branch_id) }}</template>
            <template #item.summary="{ item }">{{ conflictSummaryText(item) }}</template>
            <template #item.status="{ item }">
              <v-chip :color="item.status === 'open' ? 'warning' : 'default'" size="small" variant="tonal">
                {{ t(item.status) }}
              </v-chip>
            </template>
            <template #item.created_at="{ item }">
              <span :title="new Date(item.created_at).toLocaleString()">{{ ago(item.created_at) }}</span>
            </template>
            <template #item.actions="{ item }">
              <template v-if="canEdit && item.status === 'open'">
                <v-btn size="small" variant="text" @click.stop="openResolve(item)">{{ t('Resolve') }}</v-btn>
                <v-btn size="small" variant="text" @click.stop="dismiss(item)">{{ t('Dismiss') }}</v-btn>
              </template>
            </template>
            <template #expanded-row="{ columns, item }">
              <tr>
                <td :colspan="columns.length">
                  <div v-if="item.kind === 'oversold'" class="py-3 px-2 text-body-2">
                    {{ oversoldSummary(item) }}
                  </div>
                  <div v-else class="py-3 px-2">
                    <table class="conflict-diff">
                      <tr><th /><th>{{ t('Branch says') }}</th><th>{{ t('kashi has') }}</th></tr>
                      <tr>
                        <td>{{ t('Name') }}</td>
                        <td :class="diffClass(branchPayload(item).name, item.central_name?.String)">
                          {{ branchPayload(item).name || '—' }}
                        </td>
                        <td>{{ item.central_name?.String || '—' }}</td>
                      </tr>
                      <tr>
                        <td>{{ t('Phone') }}</td>
                        <td>{{ branchPayload(item).phone || '—' }}</td>
                        <td>{{ item.central_phone?.String || '—' }}</td>
                      </tr>
                    </table>
                  </div>
                  <div v-if="item.status !== 'open'" class="px-2 pb-3 text-caption text-medium-emphasis">
                    {{ item.status === 'dismissed' ? t('Dismissed') : t('Resolved') }} {{ ago(item.updated_at) }}
                  </div>
                </td>
              </tr>
            </template>
          </ServerSideTable>
        </div>
      </div>
    </v-card>

    <!-- resolve a sync conflict -->
    <v-dialog v-model="resolveDialog" max-width="480">
      <v-card v-if="resolveTarget">
        <v-card-title class="text-subtitle-1">
          {{ t('Resolve — {kind}', { kind: conflictKindLabel(resolveTarget.kind) }) }}
        </v-card-title>
        <v-card-text>
          <div v-if="resolveTarget.kind === 'phone_name_mismatch'">
            <div class="text-caption text-medium-emphasis mb-2">
              {{ t('Phone {phone} already belongs to a kashi client under a different name.', { phone: branchPayload(resolveTarget).phone }) }}
            </div>
            <v-radio-group v-model="resolveForm.choice" density="compact" hide-details>
              <v-radio
                :label="t('Keep kashi\'s name ({name})', { name: resolveTarget.central_name?.String })"
                value="central"
              />
              <v-radio
                :label="t('Use the branch\'s name ({name})', { name: branchPayload(resolveTarget).name })"
                value="branch"
              />
              <v-radio :label="t('Type a name')" value="custom" />
            </v-radio-group>
            <v-text-field
              v-if="resolveForm.choice === 'custom'"
              v-model="resolveForm.name"
              class="mt-2"
              density="compact"
              :label="t('Name')"
              variant="outlined"
            />
          </div>

          <div v-else-if="resolveTarget.kind === 'invalid_phone'">
            <div class="text-caption text-medium-emphasis mb-2">
              {{ t('The branch reported "{name}" with no usable phone.', { name: branchPayload(resolveTarget).name }) }}
            </div>
            <v-radio-group v-model="resolveForm.choice" density="compact" hide-details>
              <v-radio :label="t('Merge into an existing client')" value="merge" />
              <v-radio :label="t('Create a new client with a phone')" value="create" />
            </v-radio-group>
            <v-autocomplete
              v-if="resolveForm.choice === 'merge'"
              v-model="resolveForm.clientId"
              class="mt-2"
              density="compact"
              :items="clientOptions"
              :label="t('Existing client')"
              variant="outlined"
            />
            <v-text-field
              v-else
              v-model="resolveForm.phone"
              class="mt-2"
              density="compact"
              :label="t('Phone')"
              variant="outlined"
            />
          </div>

          <div v-else-if="resolveTarget.kind === 'oversold'" class="text-body-2">
            <p class="mb-2">{{ oversoldSummary(resolveTarget) }}</p>
            <p class="mb-0">
              {{ t('The sale stands. Correct the stock the usual way (a stock adjustment, a transfer in, or a stock count), then mark this resolved. It reopens if the product goes below zero again.') }}
            </p>
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn :text="t('Cancel')" variant="text" @click="resolveDialog = false" />
          <v-btn
            color="primary"
            :loading="resolveSaving"
            :text="t('Resolve')"
            variant="flat"
            @click="submitResolve"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { t } from '@/i18n'
  import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import PageSidebar from '@/components/PageSidebar.vue'
  import ServerSideTable from '@/components/Tables/ServerSideTable.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSyncConflicts } from '@/composables/useBranchSyncConflicts'
  import { useClients } from '@/composables/useClients'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const route = useRoute()
  const router = useRouter()

  const { branches, fetchBranches, listRegisters } = useBranches()
  const {
    conflictSummary,
    refreshConflictSummary,
    resolveConflict,
    dismissConflict,
    bulkResolve,
  } = useBranchSyncConflicts()

  // Computed so the labels follow the EN/FR toggle.
  const STATUS_FILTERS = computed(() => [
    { title: t('Open'), value: 'open' },
    { title: t('Resolved'), value: 'resolved' },
    { title: t('Dismissed'), value: 'dismissed' },
    { title: t('All'), value: '' },
  ])
  const ENTITY_FILTERS = computed(() => [
    { title: t('Clients'), value: 'client' },
    { title: t('Stock'), value: 'stock' },
  ])

  const branchesLoading = ref(false)
  const selectedId = ref(null)
  const statusFilter = ref('open')
  const entityFilter = ref(null)
  const search = ref('')
  const tableRef = ref(null)
  const selectedConflicts = ref([])

  const branchItems = computed(() => branches.value.map(b => ({ value: b.id, title: b.name, subtitle: b.code, raw: b })))
  const branchName = id => branches.value.find(b => b.id === id)?.name ?? `#${id}`

  const headers = computed(() => [
    ...(canEdit.value && statusFilter.value === 'open'
      ? [{ title: '', key: 'data-table-select', sortable: false, width: '44px' }]
      : []),
    { title: t('Type'), key: 'kind', sortable: false },
    ...(selectedId.value == null ? [{ title: t('Branch'), key: 'branch_id', sortable: false }] : []),
    { title: t('What'), key: 'summary', sortable: false },
    { title: t('Status'), key: 'status', sortable: false },
    { title: t('Reported'), key: 'created_at', sortable: true },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ])

  const queryParams = computed(() => ({
    branch_id: selectedId.value ?? 0,
    status: statusFilter.value ?? '',
    entity: entityFilter.value ?? '',
    sort_by: 'created_at',
    sort_order: 'desc',
  }))

  function summaryFor (branchId) {
    return conflictSummary.value.find(s => s.branch_id === branchId) ?? { open: 0, open_oversold: 0 }
  }
  const totals = computed(() => {
    const rows = selectedId.value == null
      ? conflictSummary.value
      : conflictSummary.value.filter(s => s.branch_id === selectedId.value)
    const oldest = rows.map(r => r.oldest_open_at).filter(Boolean).map(t => new Date(t).getTime())
    return {
      open: rows.reduce((n, r) => n + r.open, 0),
      oversold: rows.reduce((n, r) => n + r.open_oversold, 0),
      oldestOpenAt: oldest.length > 0 ? new Date(Math.min(...oldest)) : null,
    }
  })

  // Oversold alerts name the register that made the sale by id.
  const registerCodes = ref({})
  async function loadRegisterCodes () {
    const lists = await Promise.all(branches.value.map(b => listRegisters(b.id).catch(() => [])))
    const codes = {}
    for (const r of lists.flat()) codes[r.id] = r.code
    registerCodes.value = codes
  }

  function conflictKindLabel (kind) {
    return {
      phone_name_mismatch: t('Name differs'),
      invalid_phone: t('No usable phone'),
      phone_taken: t('Phone already taken'),
      oversold: t('Oversold'),
    }[kind] ?? kind
  }
  function branchPayload (item) {
    if (item.branch_payload && typeof item.branch_payload === 'object') return item.branch_payload
    try {
      return JSON.parse(item.branch_payload)
    } catch {
      return {}
    }
  }
  function oversoldSummary (item) {
    const p = branchPayload(item)
    const register = registerCodes.value[p.register_id]
    const invoice = p.invoice_code || t('an invoice')
    const at = register ? t(' at register {register}', { register }) : ''
    return t('Barcode {barcode}: {sold} sold on {invoice}{at}, leaving {onHand} in stock.', { barcode: p.barcode || '—', sold: p.quantity_sold, invoice, at, onHand: p.on_hand })
  }
  function conflictSummaryText (item) {
    const p = branchPayload(item)
    if (item.kind === 'oversold') return t('{ref}, {n} left', { ref: p.barcode || item.ref, n: p.on_hand })
    if (item.kind === 'phone_name_mismatch') return t('{name} vs {central}, {phone}', { name: p.name, central: item.central_name?.String || '—', phone: p.phone })
    return p.name || item.ref
  }
  function diffClass (a, b) {
    return a && b && a !== b ? 'text-error' : ''
  }
  function ago (value) {
    const seconds = Math.max(0, Math.round((Date.now() - new Date(value).getTime()) / 1000))
    if (seconds < 60) return t('just now')
    if (seconds < 3600) return t('{n} min ago', { n: Math.floor(seconds / 60) })
    if (seconds < 86_400) return t('{n} h ago', { n: Math.floor(seconds / 3600) })
    return t('{n} d ago', { n: Math.floor(seconds / 86_400) })
  }

  function reload () {
    tableRef.value?.reloadFromFirstPage?.()
    selectedConflicts.value = []
    refreshConflictSummary().catch(() => {})
  }

  async function dismiss (item) {
    try {
      await dismissConflict(item.id)
      notify.success(t('Dismissed.'))
      reload()
    } catch (error) {
      notify.error(String(error.message || error))
    }
  }

  async function bulkTakeBranch () {
    try {
      const res = await bulkResolve(selectedConflicts.value.map(c => c.id), 'branch')
      notify.success(t('Resolved {n} conflict(s).', { n: res?.resolved ?? 0 }))
      reload()
    } catch (error) {
      notify.error(String(error.message || error))
    }
  }

  const resolveDialog = ref(false)
  const resolveTarget = ref(null)
  const resolveSaving = ref(false)
  const resolveForm = reactive({ choice: 'branch', name: '', phone: '', clientId: null })
  const { clients, fetchClients } = useClients()
  const clientOptions = computed(() =>
    clients.value.map(c => ({ title: `${c.name} (${c.phone})`, value: c.id })),
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
      resolveForm.choice = 'done'
    }
    resolveDialog.value = true
  }

  async function submitResolve () {
    const item = resolveTarget.value
    if (!item) return
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
      notify.success(t('Resolved.'))
      resolveDialog.value = false
      reload()
    } catch (error) {
      notify.error(String(error.message || error))
    } finally {
      resolveSaving.value = false
    }
  }

  watch(selectedId, id => {
    selectedConflicts.value = []
    router.replace({ query: { ...route.query, branch: id == null ? undefined : String(id) } })
  })
  watch(statusFilter, () => {
    selectedConflicts.value = []
  })

  // New conflicts arrive as a burst of admin-WS nudges; one reload per burst.
  let nudgeTimer = null
  let offAdminMessage = null
  const { onMessage: onAdminMessage, ensureConnected } = useAdminSocket()
  onMounted(async () => {
    ensureConnected?.()
    offAdminMessage = onAdminMessage((msg) => {
      if (msg?.type !== 'branch_sync_conflict' || nudgeTimer) return
      nudgeTimer = setTimeout(() => {
        nudgeTimer = null
        tableRef.value?.reload?.()
        refreshConflictSummary().catch(() => {})
      }, 2000)
    })

    branchesLoading.value = true
    try {
      await fetchBranches(true)
      const wanted = Number(route.query.branch)
      selectedId.value = branches.value.find(b => b.id === wanted)?.id ?? null
      await Promise.all([refreshConflictSummary(), loadRegisterCodes()])
    } catch (error) {
      notify.error(error.message)
    } finally {
      branchesLoading.value = false
    }
  })
  onUnmounted(() => {
    clearTimeout(nudgeTimer)
    offAdminMessage?.()
  })
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bk-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bk-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.bk-content {
  flex: 1 1 auto;
  min-width: 0;
  overflow-y: auto;
}
.bk-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 12px;
}
.bk-stat {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
  padding: 12px 14px;
}
.bk-stat-label {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.bk-stat-value {
  font-size: 1.6rem;
  font-weight: 600;
  line-height: 1.3;
}
.bk-stat-sub {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.conflict-diff th,
.conflict-diff td {
  padding: 2px 16px 2px 0;
  text-align: start;
}
</style>
