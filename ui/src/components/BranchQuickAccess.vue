<template>
  <div>
    <div class="bs-section-title mb-0">Where each till's Quick Access comes from</div>
    <p class="text-body-2 text-medium-emphasis mb-3">
      <strong>Till</strong>: the till keeps and edits its own cards (the default).
      <strong>kashi</strong>: the till shows this branch's list below, read-only.
      A till can also switch itself; the most recent switch wins.
    </p>
    <v-table class="mb-6" density="compact">
      <thead>
        <tr>
          <th>Register</th>
          <th class="text-right">Quick Access from</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in registers" :key="r.id">
          <td>
            <span class="font-weight-medium">{{ r.code }}</span>
            <span v-if="r.name" class="text-medium-emphasis"> · {{ r.name }}</span>
            <v-chip v-if="!r.is_active" class="ms-2" size="x-small" text="Disabled" />
          </td>
          <td class="text-right">
            <v-btn-toggle
              color="primary"
              density="compact"
              :disabled="!canEdit || switchingId === r.id"
              mandatory
              :model-value="r.mode"
              variant="outlined"
              @update:model-value="mode => setMode(r, mode)"
            >
              <v-btn size="small" text="Till" value="till" />
              <v-btn size="small" text="kashi" value="kashi" />
            </v-btn-toggle>
          </td>
        </tr>
        <tr v-if="registers.length === 0">
          <td class="text-medium-emphasis py-4" colspan="2">This branch has no registers yet.</td>
        </tr>
      </tbody>
    </v-table>

    <div class="d-flex align-center justify-space-between">
      <div class="bs-section-title mb-0">Branch list</div>
      <div v-if="canEdit" class="d-flex ga-2">
        <v-btn
          :disabled="!dirty || saving"
          size="small"
          text="Discard"
          variant="text"
          @click="discard"
        />
        <v-btn
          color="primary"
          :disabled="!dirty"
          :loading="saving"
          size="small"
          text="Save list"
          variant="flat"
          @click="save"
        />
      </div>
    </div>
    <p class="text-body-2 text-medium-emphasis mb-3">
      Shown on every register set to <strong>kashi</strong>, in this order. Drag to reorder.
      The first till to connect after this feature arrived seeded it with its own cards.
    </p>

    <div v-if="canEdit" class="d-flex ga-2 mb-3">
      <v-autocomplete
        v-model="addVariantId"
        class="flex-grow-1"
        clearable
        density="compact"
        hide-details
        item-title="label"
        item-value="id"
        :items="variantOptions"
        label="Add a product"
        :loading="variantSearchLoading"
        no-filter
        variant="outlined"
        @update:model-value="addSelected"
        @update:search="onVariantSearch"
      />
    </div>

    <v-progress-linear v-if="loading" class="mb-2" indeterminate />
    <div v-else-if="items.length === 0" class="text-medium-emphasis py-4">No cards yet.</div>
    <div
      v-for="(item, index) in items"
      :key="item.product_variant_id"
      class="qa-row d-flex align-center ga-3 py-2"
      :class="{ 'qa-row--over': overIndex === index && dragIndex !== index }"
      :draggable="canEdit"
      @dragend="dragIndex = overIndex = null"
      @dragover.prevent="overIndex = index"
      @dragstart="dragIndex = index"
      @drop.prevent="dropOn(index)"
    >
      <v-icon v-if="canEdit" class="qa-handle" icon="mdi-drag-vertical" />
      <div class="qa-card" :style="cardStyle(item)">
        <div class="text-caption text-truncate">{{ item.code }}</div>
        <div class="text-caption font-weight-medium text-truncate">{{ item.name }}</div>
      </div>
      <div class="flex-grow-1 text-body-2">
        <div>{{ variantLine(item) }}</div>
        <div class="text-medium-emphasis">
          {{ item.barcode }}
          <v-chip
            v-if="!item.is_active"
            class="ms-1"
            color="warning"
            size="x-small"
            text="Inactive"
          />
        </div>
      </div>
      <template v-if="canEdit">
        <label class="qa-color" title="Card colour">
          <input type="color" :value="item.color || '#ffffff'" @input="setColor(item, 'color', $event)">
        </label>
        <label class="qa-color" title="Text colour">
          <input type="color" :value="item.text_color || autoTextColor(item.color)" @input="setColor(item, 'text_color', $event)">
        </label>
        <v-btn
          :disabled="!item.color && !item.text_color"
          icon="mdi-close"
          size="x-small"
          title="Default colours"
          variant="text"
          @click="item.color = ''; item.text_color = ''"
        />
        <v-btn
          icon="mdi-delete"
          size="small"
          title="Remove"
          variant="text"
          @click="items.splice(index, 1)"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import { useBranchQuickAccess } from '@/composables/useBranchQuickAccess'
  import { useNotify } from '@/composables/useNotify'
  import { useVariants } from '@/composables/useVariants'

  const props = defineProps<{ branchId: number | null | undefined, canEdit: boolean }>()

  const notify = useNotify()
  const { getBranchQuickAccess, saveBranchQuickAccess, setRegisterQuickAccessMode } = useBranchQuickAccess()
  const { variants: variantOptions, loading: variantSearchLoading, searchVariants } = useVariants()

  const loading = ref(false)
  const saving = ref(false)
  const switchingId = ref<number | null>(null)
  const registers = ref<any[]>([])
  const items = ref<any[]>([])
  // What's saved, to tell an edited list from the stored one.
  const savedSnapshot = ref('[]')

  function snapshot (list: any[]) {
    return JSON.stringify(list.map(i => [i.product_variant_id, i.color, i.text_color]))
  }
  const dirty = computed(() => snapshot(items.value) !== savedSnapshot.value)

  function apply (data: { items: any[], registers: any[] }) {
    items.value = data.items.map(i => ({ ...i }))
    registers.value = data.registers
    savedSnapshot.value = snapshot(items.value)
  }

  async function load () {
    if (props.branchId == null) return
    loading.value = true
    try {
      apply(await getBranchQuickAccess(props.branchId))
    } catch (error: any) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }
  // Switching branch drops unsaved edits, like the other tabs' forms do.
  watch(() => props.branchId, load, { immediate: true })

  async function save () {
    if (props.branchId == null) return
    saving.value = true
    try {
      apply(await saveBranchQuickAccess(props.branchId, items.value))
      notify.success('Quick Access saved -- tills set to kashi update within seconds')
    } catch (error: any) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  function discard () {
    void load()
  }

  async function setMode (register: any, mode: string) {
    if (!mode || mode === register.mode) return
    const previous = register.mode
    register.mode = mode
    switchingId.value = register.id
    try {
      await setRegisterQuickAccessMode(register.id, mode)
    } catch (error: any) {
      register.mode = previous
      notify.error(error.message)
    } finally {
      switchingId.value = null
    }
  }

  // --- adding ---------------------------------------------------------
  const addVariantId = ref<number | null>(null)
  let variantSearchTimer: ReturnType<typeof setTimeout> | undefined
  function onVariantSearch (query: string) {
    clearTimeout(variantSearchTimer)
    variantSearchTimer = setTimeout(() => searchVariants(query?.trim() ?? ''), 300)
  }
  function addSelected (id: number | null) {
    if (id == null) return
    const v = variantOptions.value.find((o: any) => o.id === id)
    addVariantId.value = null
    if (!v) return
    if (items.value.some(i => i.product_variant_id === v.id)) {
      notify.error('That product is already on the list')
      return
    }
    items.value.push({
      product_variant_id: v.id,
      code: v.product_code,
      name: v.product_name,
      color_name: v.color_name ?? '',
      size_name: v.size_name ?? '',
      barcode: v.barcode,
      is_active: v.is_active ?? true,
      color: '',
      text_color: '',
    })
  }

  // --- reordering -----------------------------------------------------
  const dragIndex = ref<number | null>(null)
  const overIndex = ref<number | null>(null)
  function dropOn (index: number) {
    const from = dragIndex.value
    dragIndex.value = overIndex.value = null
    if (from == null || from === index) return
    const [moved] = items.value.splice(from, 1)
    items.value.splice(index, 0, moved)
  }

  // --- colours --------------------------------------------------------
  function setColor (item: any, field: 'color' | 'text_color', event: Event) {
    item[field] = (event.target as HTMLInputElement).value
  }
  // Same rule as the till: black or white text, whichever reads better.
  function autoTextColor (bg: string) {
    if (!/^#[0-9a-f]{6}$/i.test(bg)) return '#000000'
    const [r, g, b] = [1, 3, 5].map(i => Number.parseInt(bg.slice(i, i + 2), 16))
    return (0.299 * r + 0.587 * g + 0.114 * b) > 150 ? '#000000' : '#ffffff'
  }
  function cardStyle (item: any) {
    if (!item.color && !item.text_color) return {}
    return {
      background: item.color || undefined,
      color: item.text_color || autoTextColor(item.color),
    }
  }
  function variantLine (item: any) {
    const cs = [item.color_name, item.size_name].filter(Boolean).join(' / ')
    return [item.name, cs].filter(Boolean).join(' · ')
  }
</script>

<style scoped>
.qa-row {
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.qa-row--over {
  border-top: 2px solid rgb(var(--v-theme-primary));
}
.qa-handle {
  cursor: grab;
}
.qa-card {
  width: 120px;
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  background: rgb(var(--v-theme-surface-variant));
}
.qa-color input {
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  background: none;
  cursor: pointer;
}
</style>
