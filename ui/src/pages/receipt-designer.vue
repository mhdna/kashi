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
        >
          <template #preview="{ layout: layoutInfo }">
            <div class="rd-paper-col">
              <v-btn-toggle
                v-model="mode"
                class="rd-modes"
                color="primary"
                data-test="receipt-modes"
                density="comfortable"
                divided
                mandatory
                variant="outlined"
              >
                <v-btn v-for="m in receiptModes" :key="m.mode" size="small" :value="m.mode">{{ m.label }}</v-btn>
              </v-btn-toggle>
              <ReceiptPaper
                v-if="modeFor(layoutInfo)"
                :catalog-texts="catalogTexts"
                :editable="canEdit"
                :mode-info="modeFor(layoutInfo)"
                :texts="texts"
                @update:texts="t => texts = t"
              />
              <div class="text-caption text-medium-emphasis rd-paper-note">
                <template v-if="canEdit">Click a part of the receipt to reword it, or the eye to leave it out.</template>
                <template v-if="mode === 'referral'"> The referral code prints only while referrals are on.</template>
              </div>
            </div>
          </template>
        </ReceiptLookPicker>
        <div class="text-caption text-medium-emphasis mt-4">
          The theme, layout and wording of the printed invoice. Each register applies them within a few
          seconds if online. The store name, phone, exchange policy text and socials come from Branch
          Settings; title and body fonts there apply to the Classic theme, the other themes bring their own.
          Previews show sample items.
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import ReceiptPaper from '@/components/Receipt/ReceiptPaper.vue'
  import ReceiptLookPicker from '@/components/ReceiptLookPicker.vue'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSettings } from '@/composables/useBranchSettings'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import catalog from '@/data/receiptThemes.json'
  import { emptyTexts, parseTexts, receiptModes, textsJSON } from '@/utils/receiptTexts'

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
  // The receipt's wording and the parts it leaves out (receipt_texts).
  const texts = ref<Record<string, any>>(emptyTexts())
  const mode = ref('sale')
  const catalogTexts = catalog.texts
  function modeFor (layoutInfo: { modes?: any[] }) {
    return layoutInfo?.modes?.find(m => m.mode === mode.value) ?? layoutInfo?.modes?.[0] ?? null
  }

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
      texts.value = parseTexts(settings?.receipt_texts)
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
      const payload = {
        receipt_theme: theme.value,
        receipt_layout: layout.value,
        receipt_texts: textsJSON(texts.value, { sections: catalogTexts.sections, maxLength: catalogTexts.max_length }),
      }
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
  max-width: 1040px;
}
.rd-target {
  flex: 0 1 240px;
}
.rd-paper-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  min-width: 0;
  max-width: 100%;
}
.rd-modes {
  flex-wrap: wrap;
  height: auto;
}
.rd-paper-note {
  max-width: 360px;
  text-align: center;
}
</style>
