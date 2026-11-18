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
          data-test="target"
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
          <template #preview>
            <div class="rd-work">
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
                  <v-btn v-for="m in modes" :key="m.mode" size="small" :value="m.mode">{{ m.label }}</v-btn>
                </v-btn-toggle>
                <ReceiptLivePaper
                  :editable="canEdit"
                  :groups="catalog?.groups ?? []"
                  :height="preview.height"
                  :image="preview.image"
                  :loading="drawing"
                  :marks="preview.marks"
                  :picked="picked"
                  @pick="g => picked = g"
                />
                <div v-if="previewError" class="text-body-2 text-error" data-test="preview-error">{{ previewError }}</div>
                <div class="text-caption text-medium-emphasis rd-paper-note">
                  Drawn with the till's own code<template v-if="target !== ALL"> and this branch's name, phone, policy and socials</template>.
                  <template v-if="mode === 'referral'"> The referral code prints only at branches with referrals on.</template>
                </div>
              </div>
              <div v-if="catalog" class="rd-panel">
                <ReceiptPartPanel
                  :catalog="catalog"
                  :design="design"
                  :group="canEdit ? picked : ''"
                  @update:design="d => design = d"
                />
              </div>
            </div>
          </template>
        </ReceiptLookPicker>
        <div class="text-caption text-medium-emphasis mt-4">
          The theme, layout and every line of the printed invoice. Each register applies them within a few
          seconds if online, once it runs a version that knows them. Title and body fonts in Branch Settings
          apply to the Classic theme; the other themes bring their own. The receipt shows sample items.
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
  import ReceiptLivePaper from '@/components/Receipt/ReceiptLivePaper.vue'
  import ReceiptPartPanel from '@/components/Receipt/ReceiptPartPanel.vue'
  import ReceiptLookPicker from '@/components/ReceiptLookPicker.vue'
  import { authFetch } from '@/composables/useApi'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSettings } from '@/composables/useBranchSettings'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'
  import { designJSON, emptyDesign, parseDesign, previewQuery } from '@/utils/receiptDesign'

  const ALL = 0

  const modes = [
    { mode: 'sale', label: 'Sale' },
    { mode: 'referral', label: 'Sale with referral code' },
    { mode: 'client', label: 'Sale with a client' },
    { mode: 'exchange', label: 'Exchange' },
    { mode: 'return', label: 'Return' },
  ]

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
  const mode = ref('sale')
  // How each line of the receipt is changed (receipt_texts).
  const design = ref<any>(emptyDesign())
  const picked = ref('')
  const catalog = ref<any>(null)

  const targets = computed(() => [
    { id: ALL, name: 'All branches' },
    ...branches.value.map((b: { id: number, name: string }) => ({ id: b.id, name: b.name })),
  ])

  async function getJSON (url: string) {
    const res = await authFetch(url)
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {throw new Error(data.error || `Request failed with status ${res.status}`)}
    return data
  }

  // The receipt, drawn again a moment after each change; an answer to an
  // older request than the last one sent is dropped.
  const preview = ref<{ image: string, height: number, marks: any[] }>({ image: '', height: 0, marks: [] })
  const drawing = ref(false)
  const previewError = ref('')
  let previewTimer: ReturnType<typeof setTimeout> | undefined
  let previewToken = 0
  async function drawNow () {
    const token = ++previewToken
    drawing.value = true
    try {
      const query = previewQuery({
        theme: theme.value, layout: layout.value, mode: mode.value,
        branchId: target.value === ALL ? 0 : target.value, design: design.value,
      })
      const data = await getJSON(`${API_BASE}/receipt_designer/preview?${query}`)
      if (token !== previewToken) {return}
      preview.value = { image: data.image, height: data.height, marks: data.marks ?? [] }
      previewError.value = ''
    } catch (error) {
      if (token === previewToken) {previewError.value = (error as Error).message}
    } finally {
      if (token === previewToken) {drawing.value = false}
    }
  }
  function drawSoon () {
    clearTimeout(previewTimer)
    previewTimer = setTimeout(drawNow, 200)
  }
  watch([theme, layout, mode, target, () => designJSON(design.value)], drawSoon)
  onUnmounted(() => clearTimeout(previewTimer))

  // Choosing one branch shows the look and design it has now; "All
  // branches" starts from whatever is on screen.
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
      design.value = parseDesign(settings?.receipt_texts)
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
      const payload = { receipt_theme: theme.value, receipt_layout: layout.value, receipt_texts: designJSON(design.value) }
      for (const id of ids) {await updateBranchSettings(id, payload)}
      if (target.value !== ALL) {savedTheme.value = theme.value}
      notify.success(`Saved for ${ids.length === 1 ? 'the branch' : `${ids.length} branches`}. Each register applies it within a few seconds if online.`)
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      saving.value = false
    }
  }

  onMounted(async () => {
    try {
      const [, data] = await Promise.all([fetchBranches(), getJSON(`${API_BASE}/receipt_designer/catalog`)])
      catalog.value = data.catalog
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      loading.value = false
    }
    drawNow()
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
}
.rd-target {
  flex: 0 1 240px;
}
.rd-work {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: center;
  gap: 20px;
  width: 100%;
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
.rd-panel {
  position: sticky;
  top: 0;
  flex: 1 1 320px;
  max-width: 460px;
  min-width: 0;
  padding: 16px;
  border-radius: 8px;
  background: rgb(var(--v-theme-surface));
}
</style>
