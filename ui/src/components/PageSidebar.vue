<!--
  Shared master/detail rail. One look for every page that has a left-hand
  picker beside a table (attributes, price/discount lists, and the category
  pages via CategorySidebar). Styled to match the vertical-tab rail on the
  Colors & Sizes page: a flat list, no nav pill, primary text + an inline-end
  stripe on the active row.

  Items are plain objects the caller maps from its own domain type:
    { value, title, subtitle?, prependIcon?, avatarColor?, avatarIcon?,
      inactive?, group?, raw? }
  `raw` (falling back to the item itself) is what `edit` / `delete` emit.
-->
<template>
  <!-- Phones/tablets: the rail would eat most of the screen, so it folds into
    a horizontally scrolling chip strip above the content (global.scss flips
    the host layout to a column). Row actions move to a menu for the
    selected item since there's no hover on touch. -->
  <div v-if="compact" class="page-sidebar page-sidebar--compact d-flex align-center ga-1 px-2 py-1">
    <v-progress-circular
      v-if="loading"
      class="mx-2"
      color="primary"
      indeterminate
      size="20"
    />
    <v-chip-group
      v-else
      class="flex-grow-1 min-w-0"
      mandatory
      :model-value="internalSelection[0]"
      selected-class="text-primary"
      @update:model-value="selectCompact"
    >
      <v-chip
        v-if="showAll"
        :prepend-icon="allIcon"
        size="small"
        :text="allTitle"
        :value="ALL_VALUE"
        variant="outlined"
      />
      <v-chip
        v-for="item in items"
        :key="String(item.value)"
        :class="{ 'text-medium-emphasis': item.inactive }"
        :prepend-icon="item.avatarIcon || item.prependIcon"
        size="small"
        :text="item.title"
        :value="item.value"
        variant="outlined"
      />
    </v-chip-group>
    <span v-if="!loading && emptyText && items.length === 0" class="text-caption text-medium-emphasis px-2">
      {{ emptyText }}
    </span>

    <v-menu v-if="rowActions && canEdit && selectedItem">
      <template #activator="{ props: menuProps }">
        <v-btn
          v-bind="menuProps"
          aria-label="Selected item actions"
          density="comfortable"
          icon="mdi-dots-vertical"
          size="small"
          variant="text"
        />
      </template>
      <v-list density="compact">
        <v-list-item prepend-icon="mdi-pencil" title="Edit" @click="$emit('edit', selectedItem.raw ?? selectedItem)" />
        <v-list-item prepend-icon="mdi-delete" title="Delete" @click="$emit('delete', selectedItem.raw ?? selectedItem)" />
      </v-list>
    </v-menu>
    <v-btn
      v-if="addLabel && canEdit"
      :aria-label="addLabel"
      density="comfortable"
      icon="mdi-plus"
      size="small"
      variant="tonal"
      @click="$emit('add')"
    />
  </div>

  <div v-else class="page-sidebar d-flex flex-column" :style="{ flex: `0 0 ${width}px`, width: `${width}px` }">
    <div v-if="loading" class="d-flex justify-center pa-6">
      <v-progress-circular color="primary" indeterminate size="24" />
    </div>
    <v-list
      v-else
      v-model:selected="internalSelection"
      class="ps-list py-0 flex-grow-1"
      density="compact"
      mandatory
      select-strategy="single-independent"
    >
      <v-list-item
        v-if="showAll"
        class="ps-row"
        :prepend-icon="allIcon"
        :title="allTitle"
        :value="ALL_VALUE"
      />

      <template v-for="group in resolvedGroups" :key="group.key">
        <v-list-subheader v-if="group.label && group.items.length > 0">
          {{ group.label }}
        </v-list-subheader>
        <v-list-item
          v-for="item in group.items"
          :key="String(item.value)"
          class="ps-row"
          :class="{ 'text-medium-emphasis': item.inactive }"
          :value="item.value"
        >
          <template v-if="$slots.prepend || item.avatarIcon || item.prependIcon" #prepend>
            <slot :item="item.raw ?? item" name="prepend">
              <v-avatar
                v-if="item.avatarIcon"
                :color="item.avatarColor"
                rounded="md"
                size="28"
                variant="tonal"
              >
                <v-icon :color="item.avatarColor" :icon="item.avatarIcon" size="16" />
              </v-avatar>
              <v-icon v-else-if="item.prependIcon" :icon="item.prependIcon" />
            </slot>
          </template>

          <v-list-item-title>{{ item.title }}</v-list-item-title>
          <v-list-item-subtitle v-if="item.subtitle">{{ item.subtitle }}</v-list-item-subtitle>

          <template #append>
            <slot :item="item.raw ?? item" name="append" />
            <div v-if="rowActions && canEdit" class="ps-actions">
              <v-btn
                density="comfortable"
                icon="mdi-pencil"
                size="x-small"
                variant="text"
                @click.stop="$emit('edit', item.raw ?? item)"
              />
              <v-btn
                density="comfortable"
                icon="mdi-delete"
                size="x-small"
                variant="text"
                @click.stop="$emit('delete', item.raw ?? item)"
              />
            </div>
          </template>
        </v-list-item>
      </template>

      <div v-if="emptyText && items.length === 0" class="text-medium-emphasis text-caption pa-4">
        {{ emptyText }}
      </div>
    </v-list>

    <template v-if="addLabel && canEdit">
      <v-divider />
      <v-btn
        block
        class="ps-add rounded-0"
        height="44"
        prepend-icon="mdi-plus"
        :text="addLabel"
        variant="text"
        @click="$emit('add')"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useDisplay } from 'vuetify'
  import { usePageAccess } from '@/composables/usePageAccess'

  // Read-only roles keep the page but lose add/edit/delete (see usePageAccess).
  const { canEdit } = usePageAccess()

  // Sentinel for the optional "All" row — v-list-item can't key off null.
  const ALL_VALUE = '__ps_all__'

  const props = defineProps({
    // Currently selected `item.value`, or null when "All" is active.
    modelValue: { type: [Number, String, null], default: null },
    items: { type: Array, default: () => [] },
    // Subheader buckets: [{ key, label }]. Items land in a bucket by `item.group`.
    // Omit for a flat list.
    groups: { type: Array, default: () => [] },
    showAll: { type: Boolean, default: false },
    allTitle: { type: String, default: 'All' },
    allIcon: { type: String, default: 'mdi-format-list-bulleted' },
    // Hover pencil/trash on each row -> `edit` / `delete`.
    rowActions: { type: Boolean, default: false },
    // Non-empty renders a pinned bottom button -> `add`.
    addLabel: { type: String, default: '' },
    emptyText: { type: String, default: '' },
    // Show a centred spinner in place of the list.
    loading: { type: Boolean, default: false },
    width: { type: [Number, String], default: 220 },
  })
  const emit = defineEmits(['update:modelValue', 'add', 'edit', 'delete'])

  const internalSelection = computed({
    get () {
      return [props.modelValue == null ? ALL_VALUE : props.modelValue]
    },
    set (arr) {
      const value = arr[0]
      emit('update:modelValue', value == null || value === ALL_VALUE ? null : value)
    },
  })

  const { smAndDown: compact } = useDisplay()

  function selectCompact (value) {
    if (value == null) return
    emit('update:modelValue', value === ALL_VALUE ? null : value)
  }

  const selectedItem = computed(() =>
    props.modelValue == null ? null : props.items.find(item => item.value === props.modelValue) ?? null,
  )

  const resolvedGroups = computed(() => {
    if (props.groups.length === 0) {
      return [{ key: '__ps_flat__', label: '', items: props.items }]
    }
    return props.groups.map(group => ({
      key: group.key,
      label: group.label,
      items: props.items.filter(item => (item.group ?? null) === group.key),
    }))
  })
</script>

<style scoped>
.page-sidebar {
  min-height: 0;
  overflow: hidden;
}
.page-sidebar--compact {
  flex: 0 0 auto;
  width: 100%;
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.min-w-0 {
  min-width: 0;
}
.ps-list {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
}
/* Pinned bottom action: fixed row-height, never stretches to fill leftover
   space when the list is short. */
.ps-add {
  flex: 0 0 auto;
}
.ps-list :deep(.v-list-item) {
  border-radius: 0;
  min-height: 44px;
}
/* Match the Colors & Sizes vertical-tab rail: primary text + an inline-end
   stripe on the active row, instead of the nav tonal pill. */
.ps-list :deep(.v-list-item--active) {
  color: rgb(var(--v-theme-primary));
}
.ps-list :deep(.v-list-item--active .v-list-item__overlay) {
  opacity: 0.06;
}
.ps-list :deep(.v-list-item--active)::after {
  content: '';
  position: absolute;
  inset-block: 4px;
  inset-inline-end: 0;
  width: 3px;
  border-radius: 3px 0 0 3px;
  background: currentColor;
}
.ps-actions {
  display: none;
}
.ps-row:hover .ps-actions {
  display: flex;
}
</style>
