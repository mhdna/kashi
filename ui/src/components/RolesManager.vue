<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-1">
      <h3 class="text-subtitle-1">Roles</h3>
      <v-btn
        color="primary"
        prepend-icon="mdi-plus"
        size="small"
        variant="tonal"
        @click="openEditor()"
      >
        Add role
      </v-btn>
    </div>
    <p class="text-body-2 text-medium-emphasis mb-4">
      A role decides which pages a user can open, and whether they can change
      what's on them (add, edit, delete) or only look. Assign roles to users in
      the Users tab.
    </p>

    <v-table density="compact">
      <thead>
        <tr>
          <th class="text-left">Role</th>
          <th class="text-left">Access</th>
          <th class="text-left">Users</th>
          <th class="text-right">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in roles" :key="r.id">
          <td>{{ r.name }}</td>
          <td>
            <v-chip v-if="r.is_admin" color="primary" size="x-small" variant="tonal">
              Everything, including users and roles
            </v-chip>
            <span v-else class="text-body-2 text-medium-emphasis">{{ summary(r) }}</span>
          </td>
          <td>{{ r.user_count }}</td>
          <td class="text-right">
            <template v-if="!r.is_admin">
              <v-btn
                icon="mdi-pencil"
                size="small"
                title="Edit role"
                variant="text"
                @click="openEditor(r)"
              />
              <v-btn
                color="error"
                :disabled="r.user_count > 0"
                icon="mdi-delete"
                size="small"
                :title="r.user_count > 0 ? 'Reassign its users first' : 'Delete role'"
                variant="text"
                @click="confirmDelete(r)"
              />
            </template>
          </td>
        </tr>
      </tbody>
    </v-table>

    <!-- Add / edit role: one tab per section, its pages listed inside -->
    <v-dialog v-model="editorOpen" max-width="860">
      <v-card>
        <v-card-title class="text-subtitle-1">
          {{ editing ? `Edit role — ${editing.name}` : 'Add role' }}
        </v-card-title>
        <v-card-text class="pb-0">
          <v-text-field
            v-model="form.name"
            density="compact"
            label="Role name"
          />

          <div class="d-flex align-center mb-1">
            <span class="text-caption text-medium-emphasis">
              <strong>View</strong> opens a page read-only. <strong>Edit</strong> also
              shows its add, edit and delete controls. Switching a section off hides
              all of its pages but keeps their settings.
            </span>
            <v-spacer />
            <v-btn size="x-small" variant="text" @click="setAll('none')">None</v-btn>
            <v-btn size="x-small" variant="text" @click="setAll('view')">All view</v-btn>
            <v-btn size="x-small" variant="text" @click="setAll('edit')">All edit</v-btn>
          </div>
        </v-card-text>

        <v-divider />

        <div class="d-flex roles-editor">
          <v-tabs
            v-model="tab"
            class="roles-editor__tabs flex-shrink-0"
            color="primary"
            density="compact"
            direction="vertical"
          >
            <v-tab
              v-for="s in sections"
              :key="s.key"
              class="justify-space-between"
              :class="{ 'text-disabled': !form.sections[s.key] }"
              :value="s.key"
            >
              <span class="text-none">{{ s.label }}</span>
              <span class="text-caption ms-3">{{ tabCount(s) }}</span>
            </v-tab>
          </v-tabs>

          <v-divider vertical />

          <v-window v-model="tab" class="flex-grow-1 roles-editor__pane">
            <v-window-item
              v-for="s in sections"
              :key="s.key"
              class="pa-4"
              :value="s.key"
            >
              <div class="d-flex align-start ga-3 mb-2">
                <div class="flex-grow-1">
                  <div class="text-body-1 font-weight-medium">{{ s.label }}</div>
                  <div class="text-caption text-medium-emphasis">{{ s.description }}</div>
                </div>
                <v-switch
                  v-model="form.sections[s.key]"
                  color="primary"
                  density="compact"
                  hide-details
                  inset
                  :label="form.sections[s.key] ? 'On' : 'Off'"
                />
              </div>

              <template v-if="s.pages?.length">
                <div class="d-flex justify-end mb-1">
                  <v-btn :disabled="!form.sections[s.key]" size="x-small" variant="text" @click="setSection(s, 'none')">None</v-btn>
                  <v-btn :disabled="!form.sections[s.key]" size="x-small" variant="text" @click="setSection(s, 'view')">All view</v-btn>
                  <v-btn :disabled="!form.sections[s.key]" size="x-small" variant="text" @click="setSection(s, 'edit')">All edit</v-btn>
                </div>
                <div
                  v-for="p in s.pages"
                  :key="p.key"
                  class="d-flex align-center ga-3 py-1"
                >
                  <div class="flex-grow-1 text-body-2" :class="{ 'text-disabled': !form.sections[s.key] }">
                    {{ p.label }}
                  </div>
                  <v-btn-toggle
                    v-model="form.pages[p.key]"
                    color="primary"
                    density="compact"
                    :disabled="!form.sections[s.key]"
                    divided
                    mandatory
                    variant="outlined"
                  >
                    <v-btn size="small" value="none">None</v-btn>
                    <v-btn size="small" value="view">View</v-btn>
                    <v-btn size="small" value="edit">Edit</v-btn>
                  </v-btn-toggle>
                </div>
              </template>
            </v-window-item>
          </v-window>
        </div>

        <v-divider />
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="editorOpen = false">Cancel</v-btn>
          <v-btn
            color="primary"
            :disabled="!form.name.trim()"
            :loading="saving"
            variant="tonal"
            @click="submit"
          >
            Save
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Delete confirm -->
    <v-dialog v-model="deleteOpen" max-width="380">
      <v-card>
        <v-card-title class="text-subtitle-1">Delete role</v-card-title>
        <v-card-text>
          Delete <strong>{{ deleteTarget?.name }}</strong>? This cannot be undone.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleteOpen = false">Cancel</v-btn>
          <v-btn color="error" :loading="saving" variant="tonal" @click="submitDelete">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { onMounted, reactive, ref } from 'vue'
  import { useRoles } from '@/composables/useRoles'
  import { useNotify } from '@/composables/useNotify'

  const { roles, sections, fetchRoles, createRole, updateRole, deleteRole } = useRoles()
  const notify = useNotify()
  const saving = ref(false)

  onMounted(async () => {
    try {
      await fetchRoles()
    } catch (error_) {
      notify.error(error_.message)
    }
  })

  // "Sales (edit), Products (2/5 pages)" -- what the role can do, at a glance.
  function summary (role) {
    const { sections: on, pages } = role.permissions
    const parts = sections.value
      .filter(s => on[s.key])
      .map(s => {
        if (!s.pages?.length) {return s.label}
        const levels = s.pages.map(p => levelOf(pages[p.key]))
        const granted = levels.filter(l => l !== 'none').length
        if (granted === 0) {return null}
        if (granted === levels.length && levels.every(l => l === levels[0])) {
          return `${s.label} (${levels[0]})`
        }
        return `${s.label} (${granted}/${levels.length} pages)`
      })
      .filter(Boolean)
    return parts.length > 0 ? parts.join(', ') : 'No access'
  }

  // The editor keeps one of 'none' | 'view' | 'edit' per page plus an on/off
  // per section, and converts to/from the API's { sections, pages } shape at
  // the edges. Page levels survive a section being switched off, like on the
  // server.
  const editorOpen = ref(false)
  const editing = ref(null)
  const tab = ref(null)
  const form = reactive({ name: '', sections: {}, pages: {} })

  function levelOf (perm) {
    return perm?.edit ? 'edit' : (perm?.view ? 'view' : 'none')
  }

  function openEditor (role = null) {
    editing.value = role
    form.name = role?.name ?? ''
    form.sections = Object.fromEntries(
      sections.value.map(s => [s.key, !!role?.permissions.sections[s.key]]),
    )
    form.pages = Object.fromEntries(
      sections.value.flatMap(s => (s.pages ?? []).map(p => [p.key, levelOf(role?.permissions.pages[p.key])])),
    )
    tab.value = sections.value[0]?.key ?? null
    editorOpen.value = true
  }

  // Pages with access in an enabled section, e.g. "3/5"; "off" otherwise.
  function tabCount (section) {
    if (!form.sections[section.key]) {return 'off'}
    if (!section.pages?.length) {return 'on'}
    const granted = section.pages.filter(p => form.pages[p.key] !== 'none').length
    return `${granted}/${section.pages.length}`
  }

  function setSection (section, level) {
    for (const p of section.pages ?? []) {
      form.pages[p.key] = level
    }
  }

  // Every section and page at once. "None" switches sections off too.
  function setAll (level) {
    for (const s of sections.value) {
      form.sections[s.key] = level !== 'none'
      setSection(s, level)
    }
  }

  function toPermissions () {
    return {
      sections: { ...form.sections },
      pages: Object.fromEntries(
        Object.entries(form.pages).map(([key, level]) => [key, { view: level !== 'none', edit: level === 'edit' }]),
      ),
    }
  }

  async function submit () {
    saving.value = true
    try {
      await (editing.value
        ? updateRole(editing.value.id, form.name.trim(), toPermissions())
        : createRole(form.name.trim(), toPermissions()))
      editorOpen.value = false
    } catch (error_) {
      notify.error(error_.message)
    } finally {
      saving.value = false
    }
  }

  const deleteOpen = ref(false)
  const deleteTarget = ref(null)

  function confirmDelete (role) {
    deleteTarget.value = role
    deleteOpen.value = true
  }

  async function submitDelete () {
    saving.value = true
    try {
      await deleteRole(deleteTarget.value.id)
    } catch (error_) {
      notify.error(error_.message)
    } finally {
      saving.value = false
      deleteOpen.value = false
    }
  }
</script>

<style scoped>
  .roles-editor {
    height: min(60vh, 520px);
  }

  .roles-editor__tabs,
  .roles-editor__pane {
    overflow-y: auto;
  }
</style>
