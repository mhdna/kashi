<template>
  <v-container class="pa-0 fill-height mx-2 mb-4" fluid>
    <v-row class="fill-height" no-gutters>
      <v-col
        v-for="panel in panels"
        :key="panel.key"
        class="d-flex flex-column"
        cols="12"
        md="6"
      >
        <v-card border class="d-flex flex-column fill-height" flat tile>
          <v-card-title class="text-caption text-uppercase text-grey py-1 px-3">{{ panel.title }}</v-card-title>
          <v-divider />
          <v-list class="pa-0 flex-grow-1" density="compact">
            <v-list-item v-for="row in tableItems(panel.key)" :key="row.label" class="px-3">
              <template #title>{{ row.label }}</template>
              <template #append>
                <span class="text-caption mr-3">{{ row.value }}</span>
                <span class="text-caption text-grey">{{ row.pct }}</span>
              </template>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup>
  import { t } from '@/i18n'
  const panels = [
    { key: 'categories', title: t('Top Categories') },
    { key: 'subcategories', title: t('Top SubCategories') },
    { key: 'kinds', title: t('Top Kinds') },
    { key: 'brands', title: t('Top Brands') },
  ]

  const data = {
    categories: [{ label: t('Men\'s Wear'), value: 340 }, { label: t('Women\'s Wear'), value: 280 }, { label: t('Kids\' Wear'), value: 175 }, { label: t('Footwear'), value: 155 }, { label: t('Accessories'), value: 110 }, { label: t('Sportswear'), value: 85 }],
    subcategories: [{ label: t('Shirts'), value: 190 }, { label: t('Pants'), value: 160 }, { label: t('Dresses'), value: 145 }, { label: t('Jackets'), value: 120 }, { label: t('Shoes'), value: 100 }, { label: t('Bags'), value: 85 }, { label: t('Hats'), value: 55 }],
    kinds: [{ label: t('Casual'), value: 310 }, { label: t('Formal'), value: 220 }, { label: t('Athletic'), value: 175 }, { label: t('Luxury'), value: 120 }, { label: t('Seasonal'), value: 80 }],
    brands: [{ label: t('Nike'), value: 280 }, { label: t('Adidas'), value: 240 }, { label: t('Zara'), value: 210 }, { label: t('H&M'), value: 185 }, { label: t('Levi\'s'), value: 140 }, { label: t('Gucci'), value: 110 }, { label: t('Prada'), value: 85 }],
  }

  function tableItems (key) {
    const rows = data[key]
    const total = rows.reduce((s, r) => s + r.value, 0)
    return rows.map(r => ({ ...r, pct: ((r.value / total) * 100).toFixed(1) + '%' }))
  }
</script>

<style scoped>
.v-row {
  flex-wrap: wrap;
}

.v-col {
  height: 50%;
}

/* Stacked one per row on phones/tablets, each panel takes its natural
   height instead of half the viewport. */
@media (max-width: 959.98px) {
  .v-col {
    height: auto;
  }
}
</style>
