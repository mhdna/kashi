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

    <v-alert
      v-if="error"
      class="mb-3"
      density="compact"
      type="error"
      variant="tonal"
    >
      {{ error }}
    </v-alert>

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

    <!-- Add / edit role -->
    <v-dialog v-model="editorOpen" max-width="620">
      <v-card>
        <v-card-title class="text-subtitle-1">
          {{ editing ? `Edit role — ${editing.name}` : 'Add role' }}
        </v-card-title>
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
            label="Role name"
          />

          <div class="d-flex align-center mt-2 mb-1">
            <span class="text-body-2 font-weight-medium">Access per section</span>
            <v-spacer />
            <v-btn size="x-small" variant="text" @click="setAll('none')">None</v-btn>
            <v-btn size="x-small" variant="text" @click="setAll('view')">All view</v-btn>
            <v-btn size="x-small" variant="text" @click="setAll('edit')">All edit</v-btn>
          </div>
          <p class="text-caption text-medium-emphasis mb-2">
            <strong>View</strong> opens the pages read-only. <strong>Edit</strong> also
            shows the add, edit and delete controls.
          </p>

          <div
            v-for="s in sections"
            :key="s.key"
            class="d-flex align-center ga-3 py-1"
          >
            <div class="flex-grow-1">
              <div class="text-body-2">{{ s.label }}</div>
              <div class="text-caption text-medium-emphasis">{{ s.description }}</div>
            </div>
            <v-btn-toggle
              v-model="form.levels[s.key]"
              color="primary"
              density="compact"
              divided
              mandatory
              variant="outlined"
            >
              <v-btn size="small" value="none">None</v-btn>
              <v-btn size="small" value="view">View</v-btn>
              <v-btn size="small" value="edit">Edit</v-btn>
            </v-btn-toggle>
          </div>
        </v-card-text>
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

  const { roles, sections, fetchRoles, createRole, updateRole, deleteRole } = useRoles()

  const error = ref('')
  const formError = ref('')
  const saving = ref(false)

  onMounted(async () => {
    try {
      await fetchRoles()
    } catch (error_) {
      error.value = error_.message
    }
  })

  // "Sales: edit, Products: view" -- what the role can do, at a glance.
  function summary (role) {
    const parts = sections.value
      .map(s => {
        const level = role.permissions[s.key]
        return level?.edit ? `${s.label} (edit)` : (level?.view ? `${s.label} (view)` : null)
      })
      .filter(Boolean)
    return parts.length > 0 ? parts.join(', ') : 'No access'
  }

  // The editor works with one of 'none' | 'view' | 'edit' per section and
  // converts to/from the API's { view, edit } shape at the edges.
  const editorOpen = ref(false)
  const editing = ref(null)
  const form = reactive({ name: '', levels: {} })

  function levelOf (perm) {
    return perm?.edit ? 'edit' : (perm?.view ? 'view' : 'none')
  }

  function openEditor (role = null) {
    editing.value = role
    form.name = role?.name ?? ''
    form.levels = Object.fromEntries(
      sections.value.map(s => [s.key, levelOf(role?.permissions[s.key])]),
    )
    formError.value = ''
    editorOpen.value = true
  }

  function setAll (level) {
    for (const s of sections.value) {
      form.levels[s.key] = level
    }
  }

  function toPermissions () {
    return Object.fromEntries(
      Object.entries(form.levels).map(([key, level]) => [key, { view: level !== 'none', edit: level === 'edit' }]),
    )
  }

  async function submit () {
    saving.value = true
    formError.value = ''
    try {
      await (editing.value
        ? updateRole(editing.value.id, form.name.trim(), toPermissions())
        : createRole(form.name.trim(), toPermissions()))
      editorOpen.value = false
    } catch (error_) {
      formError.value = error_.message
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
      error.value = error_.message
    } finally {
      saving.value = false
      deleteOpen.value = false
    }
  }
</script>
