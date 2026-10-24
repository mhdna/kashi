<template>
  <div class="page-root">
    <v-card class="bd-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-palette" />
        <span>Barcode Designer</span>
        <v-spacer />
        <v-chip v-if="dirty" color="warning" size="small" variant="tonal">Unsaved changes</v-chip>
        <v-btn
          :disabled="!customised || !design"
          prepend-icon="mdi-restore"
          text="Reset to theme"
          variant="tonal"
          @click="resetToTheme"
        />
        <v-btn
          v-if="canEdit"
          color="primary"
          :disabled="!dirty"
          :loading="saving"
          prepend-icon="mdi-content-save"
          text="Save"
          variant="flat"
          @click="save"
        />
      </v-card-title>
      <v-divider />

      <v-progress-linear v-if="loading" indeterminate />
      <div v-if="design" class="bd-body">
        <div class="bd-controls">
          <section class="bd-section">
            <div class="bd-section-title">Theme</div>
            <div class="bd-themes">
              <button
                v-for="t in themes"
                :key="t.id"
                class="bd-theme"
                :class="{ 'bd-theme--active': t.id === theme }"
                type="button"
                @click="applyTheme(t)"
              >
                <!-- eslint-disable-next-line vue/no-v-html -->
                <div class="bd-theme-thumb" v-html="t.svg" />
                <div class="bd-theme-name">{{ t.name }}</div>
                <div class="bd-theme-for">{{ t.best_for }}</div>
              </button>
            </div>
            <div class="text-caption text-medium-emphasis mt-2">
              {{ currentTheme?.description }}
              <span v-if="customised"> You have changed it from the theme; <b>Reset to theme</b> undoes that.</span>
            </div>
          </section>

          <section class="bd-section">
            <div class="bd-section-title">Look</div>
            <div class="bd-row">
              <v-select
                v-model="design.font"
                density="compact"
                hide-details
                :items="FONTS"
                label="Font"
                variant="outlined"
              />
              <div>
                <div class="text-caption text-medium-emphasis mb-1">Alignment</div>
                <v-btn-toggle
                  v-model="design.align"
                  color="primary"
                  density="comfortable"
                  divided
                  mandatory
                  variant="outlined"
                >
                  <v-btn icon="mdi-format-align-left" value="left" />
                  <v-btn icon="mdi-format-align-center" value="center" />
                  <v-btn icon="mdi-format-align-right" value="right" />
                </v-btn-toggle>
              </div>
            </div>
            <BarcodeDesignSlider
              v-model="design.margin"
              class="mt-3"
              label="Side margin"
              :max="60"
              :min="0"
            />
          </section>

          <v-expansion-panels class="bd-panels" :model-value="['price', 'name']" multiple variant="accordion">
            <v-expansion-panel v-for="el in TEXT_ELEMENTS" :key="el.key" :value="el.key">
              <v-expansion-panel-title>
                <v-switch
                  v-model="design[el.key].enabled"
                  color="primary"
                  density="compact"
                  hide-details
                  @click.stop
                />
                <span class="ms-2">{{ el.title }}</span>
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <div class="bd-stack" :class="{ 'bd-off': !design[el.key].enabled }">
                  <BarcodeDesignSlider
                    v-model="design[el.key].size"
                    label="Size"
                    :max="80"
                    :min="4"
                    :step="0.5"
                  />
                  <BarcodeDesignSlider v-model="design[el.key].y" label="Height from bottom" :max="144" :min="0" />
                  <div class="bd-checks">
                    <v-checkbox v-model="design[el.key].bold" density="compact" hide-details label="Bold" />
                    <v-checkbox v-model="design[el.key].upper" density="compact" hide-details label="Capitals" />
                    <v-checkbox v-model="design[el.key].fit" density="compact" hide-details label="Shrink to fit the label" />
                  </div>
                  <v-select
                    v-model="design[el.key].align"
                    density="compact"
                    hide-details
                    :items="ALIGN_OPTIONS"
                    label="Alignment"
                    variant="outlined"
                  />
                  <template v-if="el.key === 'price'">
                    <div class="bd-row">
                      <v-text-field
                        v-model="design.price.prefix"
                        density="compact"
                        hide-details
                        label="Before the price"
                        maxlength="8"
                        variant="outlined"
                      />
                      <v-text-field
                        v-model="design.price.suffix"
                        density="compact"
                        hide-details
                        label="After the price"
                        maxlength="8"
                        variant="outlined"
                      />
                    </div>
                    <v-checkbox v-model="design.price.rule" density="compact" hide-details label="Line under the price" />
                    <template v-if="design.price.rule">
                      <BarcodeDesignSlider v-model="design.price.rule_y" label="Line height" :max="144" :min="0" />
                      <BarcodeDesignSlider
                        v-model="design.price.rule_scale"
                        label="Line width"
                        :max="4"
                        :min="0.2"
                        :step="0.1"
                      />
                    </template>
                  </template>
                  <v-checkbox
                    v-if="el.key === 'name'"
                    v-model="design.name.variant"
                    density="compact"
                    hide-details
                    label="Add colour and size after the name"
                  />
                </div>
              </v-expansion-panel-text>
            </v-expansion-panel>

            <v-expansion-panel value="barcode">
              <v-expansion-panel-title>
                <v-switch
                  v-model="design.barcode.enabled"
                  color="primary"
                  density="compact"
                  hide-details
                  @click.stop
                />
                <span class="ms-2">Barcode</span>
              </v-expansion-panel-title>
              <v-expansion-panel-text>
                <div class="bd-stack" :class="{ 'bd-off': !design.barcode.enabled }">
                  <BarcodeDesignSlider v-model="design.barcode.height" label="Bar height" :max="120" :min="6" />
                  <BarcodeDesignSlider
                    v-model="design.barcode.module"
                    label="Bar width"
                    :max="3"
                    :min="0.8"
                    :step="0.05"
                  />
                  <BarcodeDesignSlider v-model="design.barcode.guard" label="Guard bars" :max="24" :min="0" />
                  <BarcodeDesignSlider v-model="design.barcode.y" label="Height from bottom" :max="144" :min="0" />
                  <v-select
                    v-model="design.barcode.align"
                    density="compact"
                    hide-details
                    :items="ALIGN_OPTIONS"
                    label="Alignment"
                    variant="outlined"
                  />
                  <v-checkbox v-model="design.digits.enabled" density="compact" hide-details label="Show the number under the bars" />
                  <template v-if="design.digits.enabled">
                    <BarcodeDesignSlider
                      v-model="design.digits.size"
                      label="Number size"
                      :max="40"
                      :min="4"
                      :step="0.5"
                    />
                    <BarcodeDesignSlider v-model="design.digits.gap" label="Number gap" :max="40" :min="0" />
                  </template>
                </div>
              </v-expansion-panel-text>
            </v-expansion-panel>
          </v-expansion-panels>
        </div>

        <aside class="bd-preview">
          <div class="bd-preview-sticky">
            <div class="bd-section-title">Preview</div>
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div class="bd-label" v-html="svg" />
            <v-btn-toggle
              v-model="sample"
              class="mt-3"
              color="primary"
              density="comfortable"
              divided
              mandatory
              variant="outlined"
            >
              <v-btn text="Typical" :value="0" />
              <v-btn text="Long name" :value="1" />
              <v-btn text="No price" :value="2" />
            </v-btn-toggle>
            <v-checkbox v-model="borders" density="compact" hide-details label="Side strips (separator pages on)" />
            <div class="text-caption text-medium-emphasis mt-2">
              Label {{ labelSize.width }} × {{ labelSize.height }} points (72 = 1 inch). Change the size in
              Settings, or per print in Barcodes. Sizes and positions are on a 288 × 144 label and scale with it.
            </div>
          </div>
        </aside>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, toRaw, watch } from 'vue'
  import BarcodeDesignSlider from '@/components/BarcodeDesignSlider.vue'
  import { useBarcodes } from '@/composables/useBarcodes'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'

  type Align = '' | 'center' | 'left' | 'right'
  interface TextElement { enabled: boolean, y: number, size: number, bold: boolean, align: Align, upper: boolean, fit: boolean }
  interface Design {
    font: string
    align: Exclude<Align, ''>
    margin: number
    price: TextElement & { prefix: string, suffix: string, rule: boolean, rule_y: number, rule_scale: number }
    name: TextElement & { variant: boolean }
    brand: TextElement
    code: TextElement
    barcode: { enabled: boolean, y: number, height: number, module: number, guard: number, align: Align }
    digits: { enabled: boolean, size: number, gap: number }
  }
  interface Theme { id: string, name: string, description: string, best_for: string, design: Design, svg: string }

  const { canEdit } = usePageAccess()
  const { getDesign, saveDesign, previewDesign } = useBarcodes()
  const notify = useNotify()

  const FONTS = [
    { title: 'Helvetica (sans-serif)', value: 'helvetica' },
    { title: 'Times (serif)', value: 'times' },
    { title: 'Courier (typewriter)', value: 'courier' },
  ]
  const ALIGN_OPTIONS = [
    { title: 'Follow the label', value: '' },
    { title: 'Left', value: 'left' },
    { title: 'Center', value: 'center' },
    { title: 'Right', value: 'right' },
  ]
  const TEXT_ELEMENTS = [
    { key: 'price', title: 'Price' },
    { key: 'name', title: 'Name' },
    { key: 'brand', title: 'Brand' },
    { key: 'code', title: 'Product code' },
  ] as const

  const loading = ref(true)
  const saving = ref(false)
  const themes = ref<Theme[]>([])
  const theme = ref('classic')
  const design = ref<Design | null>(null)
  const savedJson = ref('')
  const labelSize = ref({ width: 288, height: 144 })

  const sample = ref(0)
  const borders = ref(false)
  const svg = ref('')

  const currentTheme = computed(() => themes.value.find(t => t.id === theme.value))
  const dirty = computed(() => design.value !== null && JSON.stringify(design.value) !== savedJson.value)
  // Different from the theme it started from.
  const customised = computed(() =>
    design.value !== null && currentTheme.value !== undefined
    && JSON.stringify(design.value) !== JSON.stringify(currentTheme.value.design))

  // structuredClone rejects Vue's reactive proxies, which theme objects are,
  // so clone what they wrap.
  function clone (value: Design): Design {
    return structuredClone(toRaw(value))
  }

  function applyTheme (t: Theme) {
    theme.value = t.id
    design.value = clone(t.design)
  }

  function resetToTheme () {
    if (currentTheme.value) design.value = clone(currentTheme.value.design)
  }

  async function load () {
    try {
      const data = await getDesign()
      themes.value = data.themes
      labelSize.value = data.label
      theme.value = data.saved.theme
      design.value = clone(data.saved.design)
      savedJson.value = JSON.stringify(data.saved.design)
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      loading.value = false
    }
  }

  async function save () {
    if (!design.value) return
    saving.value = true
    try {
      const { saved } = await saveDesign({ theme: theme.value, design: design.value })
      theme.value = saved.theme
      design.value = clone(saved.design)
      savedJson.value = JSON.stringify(saved.design)
      notify.success('Label design saved. Every label you print from Barcodes uses it.')
    } catch (error) {
      notify.error((error as Error).message)
    } finally {
      saving.value = false
    }
  }

  // The preview is the label as the server draws it. Edits are debounced, and
  // an answer for an older edit never replaces a newer one.
  let timer: ReturnType<typeof setTimeout> | undefined
  let seq = 0
  watch([design, sample, borders], () => {
    if (!design.value) return
    clearTimeout(timer)
    timer = setTimeout(async () => {
      const mine = ++seq
      try {
        const res = await previewDesign({ design: design.value, sample: sample.value, borders: borders.value })
        if (mine === seq) svg.value = res.svg
      } catch (error) {
        if (mine === seq) notify.error((error as Error).message)
      }
    }, 120)
  }, { deep: true })

  onMounted(load)
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bd-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.bd-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(300px, 420px);
  gap: 24px;
  padding: 16px;
  align-items: start;
}
.bd-section {
  margin-bottom: 20px;
}
.bd-section-title {
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 10px;
}
.bd-themes {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 12px;
}
.bd-theme {
  text-align: left;
  padding: 8px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.bd-theme:hover {
  border-color: rgba(var(--v-theme-primary), 0.6);
}
.bd-theme--active {
  border-color: rgb(var(--v-theme-primary));
  box-shadow: 0 0 0 1px rgb(var(--v-theme-primary));
}
.bd-theme-thumb {
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.12);
  line-height: 0;
}
.bd-theme-thumb :deep(svg),
.bd-label :deep(svg) {
  width: 100%;
  height: auto;
  display: block;
}
.bd-theme-name {
  margin-top: 6px;
  font-weight: 600;
}
.bd-theme-for {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.bd-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  align-items: end;
}
.bd-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.bd-stack .bd-row {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.bd-off {
  opacity: 0.5;
}
.bd-checks {
  display: flex;
  flex-wrap: wrap;
  gap: 0 16px;
}
.bd-preview-sticky {
  position: sticky;
  top: 0;
}
/* The label is white paper in either app theme. */
.bd-label {
  background: #fff;
  border: 1px solid rgba(0, 0, 0, 0.2);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
  line-height: 0;
}
@media (max-width: 1000px) {
  .bd-body {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
