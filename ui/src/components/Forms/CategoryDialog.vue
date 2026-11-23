<template>
  <v-dialog max-width="540" :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <v-card class="px-4">
      <v-card-title>{{ category ? t('Edit Category') : t('Add a New Category') }}</v-card-title>
      <v-card-text>
        <form @submit.prevent="submit">
          <div class="d-flex align-center ga-3 mb-4">
            <v-avatar :color="pickedColor" rounded="lg" size="48" variant="tonal">
              <v-icon :color="pickedColor" :icon="pickedIcon" size="26" />
            </v-avatar>
            <v-text-field
              v-model="form.name"
              density="compact"
              :error-messages="nameError"
              hide-details="auto"
              :hint="nameHint"
              :label="t('Name')"
              persistent-hint
            />
          </div>

          <template v-if="showScope && !scope">
            <div class="text-caption text-medium-emphasis mb-1">{{ t('Scope') }}</div>
            <v-btn-toggle
              v-model="form.scope"
              class="mb-4"
              color="primary"
              density="comfortable"
              mandatory
              variant="outlined"
            >
              <v-btn value="central">{{ t('Central') }}</v-btn>
              <v-btn value="branch">{{ t('Branch') }}</v-btn>
            </v-btn-toggle>
          </template>

          <div class="text-caption text-medium-emphasis mb-1">{{ t('Icon') }}</div>
          <div class="d-flex flex-wrap ga-1 mb-4">
            <v-btn
              v-for="icon in EXPENSE_CATEGORY_ICONS"
              :key="icon"
              :active="pickedIcon === icon"
              :color="pickedIcon === icon ? pickedColor : undefined"
              density="comfortable"
              :icon="icon"
              size="small"
              variant="tonal"
              @click="form.icon = icon"
            />
          </div>

          <div class="text-caption text-medium-emphasis mb-1">{{ t('Colour') }}</div>
          <div class="d-flex flex-wrap ga-2 mb-4">
            <v-btn
              v-for="color in EXPENSE_CATEGORY_COLORS"
              :key="color"
              :color="color"
              :icon="pickedColor === color ? 'mdi-check' : 'mdi-circle'"
              size="x-small"
              @click="form.color = color"
            />
          </div>

          <v-switch
            v-model="form.is_active"
            color="primary"
            density="compact"
            hide-details
            :label="t('Active')"
          />

          <v-card-actions class="px-0">
            <v-spacer />
            <v-btn :text="t('Cancel')" @click="$emit('update:modelValue', false)" />
            <v-btn
              color="primary"
              :loading="submitting"
              :text="category ? t('Save') : t('Add Category')"
              type="submit"
            />
          </v-card-actions>
        </form>
      </v-card-text>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { useNotify } from '@/composables/useNotify'
  import { t } from '@/i18n'
  import {
    categoryColor,
    categoryIcon,
    DEFAULT_EXPENSE_CATEGORY_COLOR,
    DEFAULT_EXPENSE_CATEGORY_ICON,
    EXPENSE_CATEGORY_COLORS,
    EXPENSE_CATEGORY_ICONS,
  } from '@/data/expenseCategoryIcons'

  const props = defineProps({
    modelValue: { type: Boolean, default: false },
    // The category being edited, or null when adding a new one.
    category: { type: Object, default: null },
    // Optional hint shown under the name field (e.g. "Lender source, e.g. Owner").
    nameHint: { type: String, default: '' },
    // Show the Central / Branch scope toggle. Off for taxonomies with no
    // scope concept (e.g. asset categories).
    showScope: { type: Boolean, default: true },
    // Fixes a new category's scope ('central' or 'branch') and hides the
    // toggle: the page's Central / Branch tab already says which it is.
    scope: { type: String, default: '' },
    // Called with the { name, is_active, icon, color, scope } payload; should
    // throw on failure so the dialog can surface the message.
    onSubmit: { type: Function, required: true },
  })
  const emit = defineEmits(['update:modelValue', 'saved'])

  const form = reactive({
    name: '',
    scope: 'central',
    icon: DEFAULT_EXPENSE_CATEGORY_ICON,
    color: DEFAULT_EXPENSE_CATEGORY_COLOR,
    is_active: true,
  })
  const notify = useNotify()
  const submitting = ref(false)
  const nameError = ref('')

  const pickedIcon = computed(() => form.icon || DEFAULT_EXPENSE_CATEGORY_ICON)
  const pickedColor = computed(() => form.color || DEFAULT_EXPENSE_CATEGORY_COLOR)

  watch(() => props.modelValue, open => {
    if (!open) return
    nameError.value = ''
    if (props.category) {
      form.name = props.category.name
      form.scope = props.category.scope === 'branch' ? 'branch' : 'central'
      form.icon = categoryIcon(props.category)
      form.color = categoryColor(props.category)
      form.is_active = props.category.is_active
    } else {
      form.name = ''
      form.scope = props.scope === 'branch' ? 'branch' : 'central'
      form.icon = DEFAULT_EXPENSE_CATEGORY_ICON
      form.color = DEFAULT_EXPENSE_CATEGORY_COLOR
      form.is_active = true
    }
  })

  async function submit () {
    if (!form.name || form.name.trim().length < 2) {
      nameError.value = t('Name needs to be at least 2 characters.')
      return
    }
    nameError.value = ''
    submitting.value = true
    try {
      await props.onSubmit({
        name: form.name.trim(),
        is_active: !!form.is_active,
        icon: form.icon || DEFAULT_EXPENSE_CATEGORY_ICON,
        color: form.color || DEFAULT_EXPENSE_CATEGORY_COLOR,
        scope: form.scope === 'branch' ? 'branch' : 'central',
      })
      emit('saved')
      emit('update:modelValue', false)
    } catch (error) {
      notify.error(error.message)
    } finally {
      submitting.value = false
    }
  }
</script>
