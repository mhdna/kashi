<template>
  <div class="client-sources">
    <!-- The answers a till offers when a cashier adds a new customer ("Where
         did they hear about us?"), with how many customers gave each. -->
    <p class="text-body-2 text-medium-emphasis mb-3">
      {{ t('The till asks this when a cashier adds a new customer. Hidden options are no longer offered.') }}
    </p>

    <div v-if="loading" class="py-6 text-center">
      <v-progress-circular indeterminate size="24" />
    </div>
    <div v-else>
      <div class="text-caption text-medium-emphasis mb-1" data-test="sources-total">
        {{ t('{n} customers answered', { n: answered }) }}
      </div>
      <v-card variant="outlined">
        <v-list class="py-0" density="compact">
          <template v-for="(row, index) in rows" :key="row.id">
            <v-divider v-if="index > 0" />
            <v-list-item class="px-3" data-test="source-row">
              <template v-if="editingId === row.id">
                <v-text-field
                  v-model="editName"
                  autofocus
                  data-test="source-edit-name"
                  density="compact"
                  :error-messages="editError ? t(editError) : ''"
                  hide-details="auto"
                  variant="outlined"
                  @keydown.enter.prevent="saveEdit(row)"
                  @keydown.esc="editingId = 0"
                />
              </template>
              <template v-else>
                <v-list-item-title class="text-body-2" :class="{ 'text-disabled': !row.is_active }">
                  <span class="font-weight-medium" data-test="source-name">{{ row.name }}</span>
                  <span v-if="!row.is_active" class="ms-2 text-caption">{{ t('Hidden') }}</span>
                </v-list-item-title>
                <v-list-item-subtitle class="text-caption" data-test="source-count">
                  {{ t('{n} customers, {share}%', { n: row.client_count, share: row.share }) }}
                </v-list-item-subtitle>
              </template>
              <template #append>
                <template v-if="editingId === row.id">
                  <v-btn size="small" :text="t('Cancel')" variant="text" @click="editingId = 0" />
                  <v-btn
                    color="primary"
                    data-test="source-save"
                    :loading="saving"
                    size="small"
                    :text="t('Save')"
                    variant="text"
                    @click="saveEdit(row)"
                  />
                </template>
                <template v-else-if="canEdit">
                  <v-btn
                    :aria-label="t('Move up')"
                    density="comfortable"
                    :disabled="index === 0"
                    icon="mdi-chevron-up"
                    size="small"
                    variant="text"
                    @click="move(index, -1)"
                  />
                  <v-btn
                    :aria-label="t('Move down')"
                    density="comfortable"
                    :disabled="index === rows.length - 1"
                    icon="mdi-chevron-down"
                    size="small"
                    variant="text"
                    @click="move(index, 1)"
                  />
                  <v-btn
                    :aria-label="t('Rename')"
                    data-test="source-rename"
                    density="comfortable"
                    icon="mdi-pencil"
                    size="small"
                    variant="text"
                    @click="startEdit(row)"
                  />
                  <v-btn
                    :aria-label="row.is_active ? t('Hide') : t('Show')"
                    data-test="source-toggle"
                    density="comfortable"
                    :icon="row.is_active ? 'mdi-eye-off' : 'mdi-eye'"
                    size="small"
                    variant="text"
                    @click="toggle(row)"
                  />
                  <v-btn
                    v-if="row.client_count === 0"
                    :aria-label="t('Delete')"
                    data-test="source-delete"
                    density="comfortable"
                    icon="mdi-delete-outline"
                    size="small"
                    variant="text"
                    @click="remove(row)"
                  />
                </template>
              </template>
            </v-list-item>
          </template>
        </v-list>
      </v-card>

      <form v-if="canEdit" class="d-flex align-start ga-2 mt-3" @submit.prevent="add">
        <v-text-field
          v-model="newName"
          data-test="source-new-name"
          density="compact"
          :error-messages="addError ? t(addError) : ''"
          hide-details="auto"
          :label="t('New option')"
          variant="outlined"
        />
        <v-btn
          color="primary"
          data-test="source-add"
          :loading="saving"
          :text="t('Add')"
          type="submit"
        />
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'
  import { t } from '@/i18n'
  import { moveSource, nextSortOrder, sourceNameError, totalAnswered, withShares } from '@/utils/clientSources'

  defineProps({
    canEdit: { type: Boolean, default: true },
  })
  const notify = useNotify()

  const raw = ref([])
  const loading = ref(false)
  const saving = ref(false)
  const newName = ref('')
  const addError = ref('')
  const editingId = ref(0)
  const editName = ref('')
  const editError = ref('')

  const rows = computed(() => withShares(raw.value))
  const answered = computed(() => totalAnswered(raw.value))

  async function send (method, path, body) {
    const res = await authFetch(`${API_BASE}${path}`, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    })
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {throw new Error(data?.error || `${res.status}`)}
    return data
  }

  async function load () {
    loading.value = true
    try {
      const data = await send('GET', '/client_sources')
      raw.value = data.client_sources || []
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  async function run (fn) {
    saving.value = true
    try {
      await fn()
      await load()
      return true
    } catch (error) {
      notify.error(error.message)
      return false
    } finally {
      saving.value = false
    }
  }

  const payload = row => ({ id: row.id, name: row.name, sort_order: row.sort_order, is_active: row.is_active })

  async function add () {
    addError.value = sourceNameError(newName.value, raw.value)
    if (addError.value) {return}
    const ok = await run(() => send('POST', '/client_sources', {
      name: newName.value.trim(), sort_order: nextSortOrder(raw.value), is_active: true,
    }))
    if (ok) {newName.value = ''}
  }

  function startEdit (row) {
    editingId.value = row.id
    editName.value = row.name
    editError.value = ''
  }

  async function saveEdit (row) {
    editError.value = sourceNameError(editName.value, raw.value, row.id)
    if (editError.value) {return}
    const ok = await run(() => send('PUT', '/client_sources', { ...payload(row), name: editName.value.trim() }))
    if (ok) {editingId.value = 0}
  }

  function toggle (row) {
    return run(() => send('PUT', '/client_sources', { ...payload(row), is_active: !row.is_active }))
  }

  function move (index, dir) {
    const changed = moveSource(raw.value, index, dir)
    if (changed.length === 0) {return}
    return run(async () => {
      for (const row of changed) {await send('PUT', '/client_sources', payload(row))}
    })
  }

  function remove (row) {
    return run(() => send('DELETE', `/client_sources/${row.id}`))
  }

  onMounted(load)

  defineExpose({ load })
</script>
