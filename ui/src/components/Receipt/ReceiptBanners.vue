<!--
  The receipt's banners, kept simple: one list of kinds of invoice (every
  invoice, any deal, buy X get Y, each of kashi's deals...), the words of the
  banner each one prints, one design for all of them, and a design of its own
  for a kind where wanted. Each receipt prints the one banner that fits best.
-->
<template>
  <div class="rbn" data-no-discard-guard>
    <v-select
      data-test="banner-global-style"
      density="compact"
      hide-details
      item-title="label"
      item-value="key"
      :items="styles"
      label="Design of all banners"
      :model-value="globalStyle"
      variant="outlined"
      @update:model-value="v => update(setGlobalBannerStyle(design, v))"
    />
    <div class="text-body-2 text-medium-emphasis mt-2 mb-3" data-test="banners-one">
      Write the words for each kind of invoice; leave it empty for no banner. An invoice prints the one that fits it best.
    </div>
    <div
      v-for="k in kinds"
      :key="k.when"
      class="rbn-item"
      :class="{ 'rbn-item--focus': focusWhen === k.when }"
      :data-test="`kind-${k.when || 'always'}`"
    >
      <div class="text-body-2 font-weight-medium mb-2">{{ k.label }}</div>
      <img v-if="of(k) && images[key(of(k))]" alt="" class="rbn-thumb" :src="images[key(of(k))]">
      <div class="rbn-fields">
        <v-text-field
          :data-test="`title-${k.when || 'always'}`"
          density="compact"
          hide-details
          label="Words"
          :maxlength="40"
          :model-value="of(k)?.title ?? ''"
          variant="outlined"
          @update:model-value="v => set(k, { title: v })"
        />
        <v-text-field
          :data-test="`subtitle-${k.when || 'always'}`"
          density="compact"
          hide-details
          label="Under them"
          :maxlength="80"
          :model-value="of(k)?.subtitle ?? ''"
          variant="outlined"
          @update:model-value="v => set(k, { subtitle: v })"
        />
        <v-select
          :data-test="`style-${k.when || 'always'}`"
          density="compact"
          hide-details
          item-title="title"
          item-value="value"
          :items="styleChoices"
          label="Design"
          :model-value="of(k)?.own_style ? of(k).style : ''"
          variant="outlined"
          @update:model-value="v => set(k, { style: v })"
        />
      </div>
      <v-btn
        v-if="of(k) && bannerMode(of(k), deals)"
        class="mt-1"
        :data-test="`see-${k.when || 'always'}`"
        density="compact"
        size="small"
        text="See it on the receipt"
        variant="text"
        @click="emit('see', bannerMode(of(k), deals))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue'
  import {
    bannerKinds, bannerMode, bannerOfKind, globalBannerStyle, setGlobalBannerStyle, setKindBanner,
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
  const styles = computed(() => props.catalog.banner_styles ?? [])
  const globalStyle = computed(() => globalBannerStyle(props.design))
  const styleChoices = computed(() => [{ value: '', title: 'Same as all' }, ...styles.value.map((s: any) => ({ value: s.key, title: s.label }))])
  // The kinds, and any a saved banner has that the list no longer offers
  // (a deal since removed), so it can still be seen and cleared.
  const kinds = computed(() => {
    const list = bannerKinds(props.deals)
    for (const b of banners.value) {
      const when = b.when ?? ''
      if (!list.some(k => k.when === when)) {list.push({ when, label: when.replace(/^deal:/, ''), place: b.place ?? '' })}
    }
    return list
  })
  const focusWhen = computed(() => (props.focus >= 0 ? banners.value[props.focus]?.when ?? '' : null))
  const of = (k: { when: string }) => bannerOfKind(props.design, k.when)

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
  watch(banners, list => { for (const b of list) {want(b)} }, { immediate: true, deep: true })

  function update (design: Record<string, any>) {
    emit('update:design', design)
  }
  function set (k: { when: string, place?: string }, patch: Record<string, any>) {
    update(setKindBanner(props.design, k as any, patch))
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
