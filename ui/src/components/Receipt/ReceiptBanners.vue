<!--
  The receipt's banners: a gallery of ready-made ones, drawn by the till's own
  code (/receipt_designer/banner), to add with a click; then each banner on
  the receipt with its words, style, when it prints and where.
-->
<template>
  <div class="rbn" data-no-discard-guard>
    <div class="text-subtitle-1 font-weight-medium mb-1">On this receipt</div>
    <div v-if="banners.length === 0" class="text-body-2 text-medium-emphasis mb-3" data-test="banners-none">
      No banners yet. Pick one below.
    </div>
    <div
      v-for="(b, i) in banners"
      :key="i"
      class="rbn-item"
      :class="{ 'rbn-item--focus': i === focus }"
      :data-test="`banner-${i}`"
    >
      <div class="d-flex align-center ga-1 mb-2">
        <span class="text-body-2 font-weight-medium flex-grow-1">Banner {{ i + 1 }}</span>
        <v-btn
          aria-label="Move up"
          :data-test="`banner-up-${i}`"
          density="compact"
          :disabled="i === 0"
          icon="mdi-arrow-up"
          size="small"
          variant="text"
          @click="update(moveBanner(design, i, -1))"
        />
        <v-btn
          aria-label="Move down"
          :data-test="`banner-down-${i}`"
          density="compact"
          :disabled="i === banners.length - 1"
          icon="mdi-arrow-down"
          size="small"
          variant="text"
          @click="update(moveBanner(design, i, 1))"
        />
        <v-btn
          aria-label="Remove"
          :data-test="`banner-remove-${i}`"
          density="compact"
          icon="mdi-close"
          size="small"
          variant="text"
          @click="update(removeBanner(design, i))"
        />
      </div>
      <img v-if="images[key(b)]" alt="" class="rbn-thumb" :src="images[key(b)]">
      <div class="rbn-fields">
        <v-text-field
          :data-test="`banner-title-${i}`"
          density="compact"
          hide-details
          label="Words"
          :maxlength="40"
          :model-value="b.title"
          variant="outlined"
          @update:model-value="v => set(i, { title: v })"
        />
        <v-text-field
          :data-test="`banner-subtitle-${i}`"
          density="compact"
          hide-details
          label="Under them"
          :maxlength="80"
          :model-value="b.subtitle ?? ''"
          variant="outlined"
          @update:model-value="v => set(i, { subtitle: v })"
        />
        <v-select
          :data-test="`banner-style-${i}`"
          density="compact"
          hide-details
          item-title="label"
          item-value="key"
          :items="catalog.banner_styles ?? []"
          label="Style"
          :model-value="b.style"
          variant="outlined"
          @update:model-value="v => set(i, { style: v })"
        />
        <v-select
          :data-test="`banner-when-${i}`"
          density="compact"
          hide-details
          :items="whenOptions"
          label="Prints"
          :model-value="b.when ?? ''"
          variant="outlined"
          @update:model-value="v => set(i, { when: v })"
        />
        <ReceiptFontSelect
          :catalog="catalog"
          label="Font of the words"
          :model-value="b.font ?? ''"
          :test="`banner-font-${i}`"
          @update:model-value="v => set(i, { font: v })"
        />
        <v-select
          :data-test="`banner-place-${i}`"
          density="compact"
          hide-details
          :items="bannerPlaceOptions"
          label="Where"
          :model-value="b.place ?? ''"
          variant="outlined"
          @update:model-value="v => set(i, { place: v })"
        />
      </div>
      <v-btn
        v-if="bannerMode(b, deals)"
        class="mt-1"
        :data-test="`banner-see-${i}`"
        density="compact"
        size="small"
        text="See it on the receipt"
        variant="text"
        @click="emit('see', bannerMode(b, deals))"
      />
    </div>

    <div class="text-subtitle-1 font-weight-medium mt-4 mb-1">Ready-made</div>
    <div v-if="banners.length >= bannerMax" class="text-body-2 text-medium-emphasis mb-2">A receipt takes up to {{ bannerMax }} banners.</div>
    <div class="rbn-gallery">
      <button
        v-for="(p, i) in catalog.banner_presets ?? []"
        :key="i"
        :aria-label="`Add ${p.title}`"
        class="rbn-preset"
        :data-test="`preset-${i}`"
        :disabled="banners.length >= bannerMax"
        type="button"
        @click="update(addBanner(design, p))"
      >
        <img v-if="images[key(p)]" :alt="p.title" :src="images[key(p)]">
        <span v-else class="rbn-preset-text">{{ p.title }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import ReceiptFontSelect from '@/components/Receipt/ReceiptFontSelect.vue'
  import {
    addBanner, bannerMax, bannerMode, bannerPlaceOptions, bannerWhenOptions, moveBanner, removeBanner, updateBanner,
  } from '@/utils/receiptDesign'

  const props = withDefaults(defineProps<{
    catalog: Record<string, any>
    design: Record<string, any>
    // kashi's deals, to print a banner with one of them.
    deals?: Array<{ name: string }>
    // Draws a banner alone (its JSON) and answers with its picture.
    draw: (banner: string) => Promise<string>
    // The banner clicked on the receipt.
    focus?: number
  }>(), { deals: () => [], focus: -1 })
  const emit = defineEmits<{ 'update:design': [design: Record<string, any>], see: [mode: string] }>()

  const banners = computed(() => props.design.banners ?? [])
  const whenOptions = computed(() => bannerWhenOptions(props.deals))

  // Each banner's picture, by what it shows; drawn once each.
  const images = ref<Record<string, string>>({})
  function key (b: Record<string, any>) {
    return JSON.stringify({ style: b.style, title: b.title, subtitle: b.subtitle ?? '', font: b.font ?? '' })
  }
  async function want (b: Record<string, any>) {
    const k = key(b)
    if (images.value[k] || !b.style || !String(b.title ?? '').trim()) {return}
    images.value = { ...images.value, [k]: '' }
    try {
      const img = await props.draw(k)
      images.value = { ...images.value, [k]: img }
    } catch {
      // The words alone show instead.
    }
  }
  watch(() => [...(props.catalog.banner_presets ?? []), ...banners.value], list => { for (const b of list) {want(b)} }, { immediate: true, deep: true })

  function update (design: Record<string, any>) {
    emit('update:design', design)
  }
  function set (i: number, patch: Record<string, any>) {
    update(updateBanner(props.design, i, patch))
  }
</script>

<style scoped>
.rbn-item {
  margin-bottom: 12px;
  padding: 10px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
}
.rbn-item--focus {
  border-color: rgb(var(--v-theme-primary));
}
.rbn-thumb {
  display: block;
  width: 100%;
  margin-bottom: 8px;
  background: #fff;
}
.rbn-fields {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 8px;
}
.rbn-gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
}
.rbn-preset {
  padding: 6px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 6px;
  background: #fff;
  cursor: pointer;
}
.rbn-preset:hover:not(:disabled) {
  border-color: rgb(var(--v-theme-primary));
}
.rbn-preset:disabled {
  opacity: 0.5;
  cursor: default;
}
.rbn-preset img {
  display: block;
  width: 100%;
}
.rbn-preset-text {
  color: #000;
  font-weight: 700;
}
</style>
