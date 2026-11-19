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
          data-test="make-default"
          :loading="savingDefault"
          prepend-icon="mdi-star-outline"
          text="Make this the default"
          title="New branches, and Default Settings' Apply to all, get this receipt"
          variant="text"
          @click="makeDefault"
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
      <v-tabs v-model="tab" class="px-4" color="primary" density="compact">
        <v-tab data-test="tab-receipt" value="receipt">Receipt</v-tab>
        <v-tab data-test="tab-banners" value="banners">Banners</v-tab>
        <v-tab data-test="tab-shop" value="shop">Shop details</v-tab>
      </v-tabs>
      <v-progress-linear v-if="loading" indeterminate />
      <div class="rd-body">
        <div class="rd-work" :class="{ 'rd-work--shop': tab !== 'receipt' }">
          <div v-if="tab === 'receipt'" class="rd-picker">
            <ReceiptLookPicker v-model:layout="layout" v-model:theme="theme" :saved-theme="savedTheme">
              <template #preview>
                <span />
              </template>
            </ReceiptLookPicker>
          </div>

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
              <v-btn v-for="m in allModes" :key="m.mode" size="small" :value="m.mode">{{ m.label }}</v-btn>
            </v-btn-toggle>
            <ReceiptLivePaper
              :catalog="catalog"
              :design="design"
              :editable="canEdit && tab === 'receipt'"
              :height="preview.height"
              :image="preview.image"
              :image-width="preview.width"
              :lines="preview.lines"
              :loading="drawing"
              :marks="preview.marks"
              :max-length="catalog?.max_length ?? 300"
              :picked="picked"
              @banner="openBanner"
              @pick="g => picked = g"
              @update:design="d => design = d"
            />
            <div v-if="previewError" class="text-body-2 text-error" data-test="preview-error">{{ previewError }}</div>
            <div class="text-caption text-medium-emphasis rd-paper-note">
              <template v-if="canEdit && tab === 'receipt'">Click any words to change them, or a part for all of it. </template>Drawn with the till's own code<template v-if="target !== ALL"> and this branch's details</template>.
              <template v-if="mode === 'referral'"> The referral code prints only at branches with referrals on.</template>
            </div>
          </div>

          <div v-if="tab === 'banners' && catalog" class="rd-panel" data-test="banners-tab">
            <ReceiptBanners
              :catalog="catalog"
              :deals="deals"
              :design="design"
              :draw="drawBanner"
              :focus="bannerFocus"
              @see="m => mode = m"
              @update:design="d => design = d"
            />
          </div>

          <div v-if="tab === 'shop'" class="rd-panel" data-test="shop-tab">
            <div class="text-subtitle-1 font-weight-medium mb-1">
              {{ target === ALL ? 'Every branch' : branchName(target) }}
            </div>
            <div class="text-caption text-medium-emphasis mb-4">
              What the receipt prints about the shop. Saved to the branch's settings with Save.
            </div>
            <v-text-field
              v-model="shop.store_name"
              data-test="shop-store-name"
              density="compact"
              :disabled="!canEdit"
              label="Store name"
              variant="outlined"
              @update:model-value="shopTouched = true"
            />
            <v-text-field
              v-if="target !== ALL"
              v-model="shop.phone"
              data-test="shop-phone"
              density="compact"
              :disabled="!canEdit"
              label="Phone"
              variant="outlined"
              @update:model-value="shopTouched = true"
            />
            <v-text-field
              v-model="shop.website"
              data-test="shop-website"
              density="compact"
              :disabled="!canEdit"
              label="Website"
              variant="outlined"
              @update:model-value="shopTouched = true"
            />
            <v-textarea
              v-model="shop.return_policy"
              auto-grow
              data-test="shop-policy"
              density="compact"
              :disabled="!canEdit"
              label="Exchange policy"
              rows="2"
              variant="outlined"
              @update:model-value="shopTouched = true"
            />
            <div class="text-subtitle-2 mb-2">Follow us</div>
            <div v-for="p in socialPlatforms" :key="p.value" class="rd-social">
              <v-checkbox-btn
                :data-test="`social-${p.value}`"
                density="compact"
                :disabled="!canEdit"
                :label="p.title"
                :model-value="shop.social_platforms.includes(p.value)"
                @update:model-value="v => toggleSocial(p.value, v)"
              />
              <v-text-field
                v-if="shop.social_platforms.includes(p.value)"
                v-model="shop.social_handles[p.value]"
                :data-test="`handle-${p.value}`"
                density="compact"
                :disabled="!canEdit"
                hide-details
                placeholder="@handle"
                variant="outlined"
                @update:model-value="shopTouched = true"
              />
            </div>
            <div v-if="target === ALL" class="text-caption text-medium-emphasis mt-2">
              Saving for every branch sets the same name, website, policy and socials everywhere; each branch keeps its own phone.
            </div>
          </div>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, provide, ref, watch } from 'vue'
  import ReceiptBanners from '@/components/Receipt/ReceiptBanners.vue'
  import ReceiptLivePaper from '@/components/Receipt/ReceiptLivePaper.vue'
  import ReceiptLookPicker from '@/components/ReceiptLookPicker.vue'
  import { authFetch } from '@/composables/useApi'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSettings } from '@/composables/useBranchSettings'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { API_BASE } from '@/config'
  import {
    dealModes, designJSON, emptyDesign, emptyShop, footerOrder, groupInfo, moveSection, parseDesign, previewQuery,
    shopFromSettings, shopPayload, socialPlatforms,
  } from '@/utils/receiptDesign'
  import { receiptFontURL } from '@/utils/receiptFonts'

  const ALL = 0

  const baseModes = [
    { mode: 'sale', label: 'Sale' },
    { mode: 'referral', label: 'Sale with referral code' },
    { mode: 'client', label: 'Sale with a client' },
    { mode: 'exchange', label: 'Exchange' },
    { mode: 'return', label: 'Return' },
    { mode: 'item_deal', label: 'Buy 2 get 1' },
  ]

  const { canEdit } = usePageAccess()
  const { branches, fetchBranches } = useBranches()
  const { getBranchSettings, updateBranchSettings } = useBranchSettings()
  const notify = useNotify()

  const loading = ref(true)
  const saving = ref(false)
  const savingDefault = ref(false)
  const tab = ref('receipt')
  const target = ref<number>(ALL)
  const theme = ref('classic')
  const layout = ref('A')
  const savedTheme = ref('')
  const mode = ref('sale')
  // How each line of the receipt is changed (receipt_texts).
  const design = ref<any>(emptyDesign())
  const picked = ref('')
  const catalog = ref<any>(null)
  // The shop's details as the Shop details tab edits them; sent with the
  // preview once touched, so they show before Save.
  const shop = ref<any>(emptyShop())
  const shopTouched = ref(false)

  // One more mode for each of kashi's deals, its line on the receipt.
  const deals = ref<any[]>([])
  const allModes = computed(() => [...baseModes, ...dealModes(deals.value)])
  const currentMode = computed(() => allModes.value.find(m => m.mode === mode.value) ?? baseModes[0])

  const targets = computed(() => [
    { id: ALL, name: 'All branches' },
    ...branches.value.map((b: { id: number, name: string }) => ({ id: b.id, name: b.name })),
  ])
  function branchName (id: number) {
    return branches.value.find((b: { id: number }) => b.id === id)?.name ?? `Branch #${id}`
  }

  async function requestJSON (url: string, options?: RequestInit) {
    const res = await authFetch(url, options)
    const data = await res.json().catch(() => ({}))
    if (!res.ok) {throw new Error(data.error || `Request failed with status ${res.status}`)}
    return data
  }

  // Banners: a click on one on the receipt opens it in the Banners tab.
  const bannerFocus = ref(-1)
  function openBanner (i: number) {
    bannerFocus.value = i
    tab.value = 'banners'
  }
  // The till's fonts, for lines shown and typed in their own letters.
  provide('receiptFontFile', async (key: string, bold: boolean) => {
    const res = await authFetch(receiptFontURL(API_BASE, key, bold))
    if (!res.ok) {throw new Error(`font ${key}: ${res.status}`)}
    return res.arrayBuffer()
  })
  async function drawBanner (banner: string) {
    const data = await requestJSON(`${API_BASE}/receipt_designer/banner?${new URLSearchParams({ b: banner })}`)
    return data.image as string
  }

  function toggleSocial (platform: string, on: boolean) {
    const list = shop.value.social_platforms.filter((p: string) => p !== platform)
    shop.value = { ...shop.value, social_platforms: on ? [...list, platform] : list }
    shopTouched.value = true
  }

  // The receipt, drawn again a moment after each change; an answer to an
  // older request than the last one sent is dropped.
  const preview = ref<{ image: string, width: number, height: number, marks: any[], lines: any[] }>({ image: '', width: 576, height: 0, marks: [], lines: [] })
  const drawing = ref(false)
  const previewError = ref('')
  let previewTimer: ReturnType<typeof setTimeout> | undefined
  let previewToken = 0
  async function drawNow () {
    const token = ++previewToken
    drawing.value = true
    try {
      const query = previewQuery({
        theme: theme.value, layout: layout.value,
        mode: currentMode.value.deal ? 'client' : mode.value,
        deal: (currentMode.value as any).deal ?? null,
        branchId: target.value === ALL ? 0 : target.value, design: design.value,
        shop: shopTouched.value ? shop.value : null,
      })
      const data = await requestJSON(`${API_BASE}/receipt_designer/preview?${query}`)
      if (token !== previewToken) {return}
      preview.value = { image: data.image, width: data.width ?? 576, height: data.height, marks: data.marks ?? [], lines: data.lines ?? [] }
      previewError.value = ''
    } catch (error) {
      if (token === previewToken) {previewError.value = (error as Error).message}
    } finally {
      if (token === previewToken) {drawing.value = false}
    }
  }
  function drawSoon () {
    clearTimeout(previewTimer)
    previewTimer = setTimeout(drawNow, 120)
  }
  watch([theme, layout, mode, target, () => designJSON(design.value), () => (shopTouched.value ? JSON.stringify(shop.value) : '')], drawSoon)

  // Alt+Up / Alt+Down moves the picked part, when it is one that moves.
  function onKey (e: KeyboardEvent) {
    if (!e.altKey || (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') || !canEdit.value || !catalog.value) {return}
    const section = groupInfo(catalog.value, picked.value)?.section
    if (!section || !footerOrder(design.value, catalog.value).includes(section)) {return}
    e.preventDefault()
    design.value = moveSection(design.value, catalog.value, section, e.key === 'ArrowUp' ? -1 : 1)
  }
  onMounted(() => window.addEventListener('keydown', onKey))
  onUnmounted(() => {
    clearTimeout(previewTimer)
    window.removeEventListener('keydown', onKey)
  })

  // "All branches" shows the Default Settings template's receipt (what
  // "Make this the default" saved), not a blank one, so what the branches
  // were given shows. An empty template keeps what is on screen.
  async function loadAllBranchesDesign () {
    try {
      const data = await requestJSON(`${API_BASE}/default_branch_settings`)
      const s = data?.settings ?? {}
      if (s.receipt_texts) {design.value = parseDesign(s.receipt_texts)}
      if (s.receipt_theme) {theme.value = s.receipt_theme}
      if (s.receipt_layout) {layout.value = s.receipt_layout}
    } catch {
      // Without the template the page starts from what is on screen.
    }
  }

  // Choosing one branch shows the look, design and details it has now.
  watch(target, async id => {
    if (id === ALL) {
      savedTheme.value = ''
      await loadAllBranchesDesign()
      return
    }
    loading.value = true
    try {
      const settings = await getBranchSettings(id)
      theme.value = settings?.receipt_theme || 'classic'
      layout.value = settings?.receipt_layout || 'A'
      design.value = parseDesign(settings?.receipt_texts)
      shop.value = shopFromSettings(settings)
      shopTouched.value = false
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
      const all = target.value === ALL
      const ids = all ? branches.value.map((b: { id: number }) => b.id) : [target.value]
      const payload = {
        receipt_theme: theme.value,
        receipt_layout: layout.value,
        receipt_texts: designJSON(design.value),
        ...(shopTouched.value ? shopPayload(shop.value, { allBranches: all }) : {}),
      }
      // For every branch, one that no register has connected from yet has
      // nothing to save to (409): it is skipped and named, the rest are saved.
      const skipped: string[] = []
      let saved = 0
      for (const id of ids) {
        try {
          await updateBranchSettings(id, payload)
          saved++
        } catch (error) {
          if (!all || (error as { status?: number }).status !== 409) {throw error}
          skipped.push(branchName(id))
        }
      }
      if (!all) {savedTheme.value = theme.value}
      shopTouched.value = false
      const done = `Saved for ${saved === 1 ? (all ? '1 branch' : 'the branch') : `${saved} branches`}. Each register applies it within a few seconds if online.`
      notify.success(skipped.length > 0 ? `${done} Skipped ${skipped.join(', ')}: no register has connected there yet.` : done)
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      saving.value = false
    }
  }

  // The look and design on screen become the default template's.
  async function makeDefault () {
    savingDefault.value = true
    try {
      const data = await requestJSON(`${API_BASE}/default_branch_settings`)
      const settings = {
        ...data.settings,
        receipt_theme: theme.value,
        receipt_layout: layout.value,
        receipt_texts: designJSON(design.value),
      }
      await requestJSON(`${API_BASE}/default_branch_settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
      })
      notify.success('This receipt is now the default. Default Settings\' Apply to all sends it to every branch.')
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      savingDefault.value = false
    }
  }

  onMounted(async () => {
    try {
      const [, data] = await Promise.all([fetchBranches(), requestJSON(`${API_BASE}/receipt_designer/catalog`)])
      catalog.value = data.catalog
      await loadAllBranchesDesign()
      // The deals are a nice-to-have here: without them the other modes stay.
      requestJSON(`${API_BASE}/invoice_discounts`).then(r => { deals.value = r.invoice_discounts ?? [] }).catch(() => {})
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
/* Only the body shrinks and scrolls: the title, tabs and loading bar keep
   their height however tall the receipt is (squeezed, the tabs were cut). */
.rd-card > :not(.rd-body) {
  flex: 0 0 auto;
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
/* Three columns side by side: themes, the receipt, its panel. The panel
   keeps beside the receipt and scrolls on its own. */
.rd-work {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  align-items: start;
  gap: 24px;
}
/* Shop details: the receipt and the form, no theme list. */
.rd-work--shop {
  grid-template-columns: auto minmax(340px, 1fr);
}
@media (max-width: 900px) {
  .rd-work {
    grid-template-columns: minmax(0, 1fr);
  }
}
@media (max-width: 760px) {
  .rd-work {
    grid-template-columns: minmax(0, 1fr);
  }
}
.rd-picker {
  min-width: 0;
}
/* The picker's own preview column isn't used here: the paper is beside it. */
.rd-picker :deep(.rlp) {
  grid-template-columns: minmax(0, 1fr);
}
.rd-picker :deep(.rlp-preview) {
  display: none;
}
.rd-paper-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  min-width: 0;
  max-width: 100%;
  /* Room on the right for a part's name beside the paper, and under it for
     a toolbar opened near the bottom. */
  padding-right: 96px;
  padding-bottom: 220px;
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
  max-height: calc(100vh - 180px);
  overflow-y: auto;
  min-width: 0;
  padding: 16px;
  border-radius: 8px;
  background: rgb(var(--v-theme-surface));
}
.rd-social {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.rd-social :deep(.v-selection-control) {
  flex: 0 0 140px;
}
</style>
