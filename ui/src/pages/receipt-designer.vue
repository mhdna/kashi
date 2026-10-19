<template>
  <div class="page-root">
    <v-card class="rd-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-palette" />
        <span>Receipt Designer</span>
        <v-spacer />
        <v-select
          v-model="target"
          class="rd-target"
          density="compact"
          hide-details
          item-title="name"
          item-value="id"
          :items="targets"
          label="Apply to"
          variant="outlined"
        />
        <v-btn
          v-if="canEdit"
          color="primary"
          :disabled="!theme || !layout"
          :loading="saving"
          prepend-icon="mdi-content-save"
          text="Save"
          variant="flat"
          @click="save"
        />
      </v-card-title>
      <v-divider />
      <v-progress-linear v-if="loading" indeterminate />
      <div class="rd-body">
        <ReceiptLookPicker
          v-model:layout="layout"
          v-model:theme="theme"
          :saved-theme="savedTheme"
        />
        <div class="text-caption text-medium-emphasis mt-4">
          The theme and layout of the printed invoice. Each register applies it within a few seconds if
          online. Title and body fonts in Branch Settings apply to the Classic theme; the other themes bring
          their own. Previews show sample items.
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import ReceiptLookPicker from '@/components/ReceiptLookPicker.vue'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSettings } from '@/composables/useBranchSettings'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'

  const ALL = 0

  const { canEdit } = usePageAccess()
  const { branches, fetchBranches } = useBranches()
  const { getBranchSettings, updateBranchSettings } = useBranchSettings()
  const notify = useNotify()

  const loading = ref(true)
  const saving = ref(false)
  const target = ref<number>(ALL)
  const theme = ref('classic')
  const layout = ref('A')
  const savedTheme = ref('')

  const targets = computed(() => [
    { id: ALL, name: 'All branches' },
    ...branches.value.map((b: { id: number, name: string }) => ({ id: b.id, name: b.name })),
  ])

  // Choosing one branch shows the look it has now; "All branches" starts from
  // whatever is on screen.
  watch(target, async id => {
    if (id === ALL) {
      savedTheme.value = ''
      return
    }
    loading.value = true
    try {
      const settings = await getBranchSettings(id)
      theme.value = settings?.receipt_theme || 'classic'
      layout.value = settings?.receipt_layout || 'A'
      savedTheme.value = theme.value
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      loading.value = false
    }
  })

  async function save () {
    saving.value = true
    try {
      const ids = target.value === ALL ? branches.value.map((b: { id: number }) => b.id) : [target.value]
      const payload = { receipt_theme: theme.value, receipt_layout: layout.value }
      for (const id of ids) await updateBranchSettings(id, payload)
      if (target.value !== ALL) savedTheme.value = theme.value
      notify.success(`Saved for ${ids.length === 1 ? 'the branch' : `${ids.length} branches`}. Each register applies it within a few seconds if online.`)
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      saving.value = false
    }
  }

  onMounted(async () => {
    try {
      await fetchBranches()
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      loading.value = false
    }
  })
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.rd-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.rd-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: 16px;
  max-width: 980px;
}
.rd-target {
  flex: 0 1 240px;
}
</style>
