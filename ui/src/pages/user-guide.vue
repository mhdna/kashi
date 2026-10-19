<template>
  <div class="page-root">
    <v-card class="guide-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-book-open-variant" />
        <span>User Guide</span>
        <v-spacer />
        <v-text-field
          v-model="search"
          clearable
          density="compact"
          hide-details
          label="Search the guide"
          prepend-inner-icon="mdi-magnify"
          style="max-width: 280px"
          variant="outlined"
        />
      </v-card-title>
      <v-divider />

      <div class="guide-layout">
        <PageSidebar
          v-model="selectedId"
          empty-text="Nothing matches your search."
          :groups="groupDefs"
          :items="items"
          :width="240"
        />

        <v-divider vertical />

        <div class="guide-content">
          <div v-if="current" :key="current.id" class="guide-page pa-6">
            <div class="text-overline text-medium-emphasis">{{ current.group }}</div>
            <h1 class="text-h5 mb-3">{{ current.title }}</h1>
            <template v-for="(block, i) in current.blocks" :key="i">
              <h2 v-if="block.kind === 'heading'" class="text-subtitle-1 font-weight-bold mt-5 mb-2">
                {{ block.text }}
              </h2>
              <p v-else-if="block.kind === 'p'" class="text-body-1 mb-3" v-html="block.text" />
              <ol v-else-if="block.kind === 'steps'" class="guide-steps text-body-1 mb-3 ps-5">
                <li v-for="(item, j) in block.items" :key="j" class="mb-2" v-html="item" />
              </ol>
              <ul v-else-if="block.kind === 'list'" class="text-body-1 mb-3 ps-5">
                <li v-for="(item, j) in block.items" :key="j" class="mb-2" v-html="item" />
              </ul>
              <v-alert
                v-else-if="block.kind === 'tip'"
                class="mb-3"
                density="compact"
                type="info"
                variant="tonal"
              >
                <span v-html="block.text" />
              </v-alert>
            </template>

            <v-divider class="my-6" />
            <div class="d-flex justify-space-between ga-2">
              <v-btn
                v-if="previous"
                prepend-icon="mdi-chevron-left"
                :text="previous.title"
                variant="text"
                @click="selectedId = previous.id"
              />
              <span v-else />
              <v-btn
                v-if="next"
                append-icon="mdi-chevron-right"
                :text="next.title"
                variant="text"
                @click="selectedId = next.id"
              />
            </div>
          </div>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup>
  import { computed, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import PageSidebar from '@/components/PageSidebar.vue'
  import { guideGroups, guideSections } from '@/data/userGuide'

  const route = useRoute()
  const router = useRouter()

  const search = ref('')

  // Plain text of an entry, for searching: tags stripped.
  function textOf (section) {
    const parts = [section.title]
    for (const block of section.blocks) {
      parts.push(block.text ?? '', ...(block.items ?? []))
    }
    return parts.join(' ').replace(/<[^>]+>/g, ' ').toLowerCase()
  }

  const visible = computed(() => {
    const q = (search.value ?? '').trim().toLowerCase()
    return q ? guideSections.filter(s => textOf(s).includes(q)) : guideSections
  })

  const groupDefs = guideGroups.map(name => ({ key: name, label: name }))
  const items = computed(() => visible.value.map(s => ({ value: s.id, title: s.title, group: s.group })))

  // The open page lives in the address (?page=), so a link or a reload opens it.
  const selectedId = computed({
    get () {
      const wanted = route.query.page
      return visible.value.some(s => s.id === wanted) ? wanted : visible.value[0]?.id ?? null
    },
    set (id) {
      router.replace({ query: { ...route.query, page: id ?? undefined } })
    },
  })

  const index = computed(() => visible.value.findIndex(s => s.id === selectedId.value))
  const current = computed(() => visible.value[index.value] ?? null)
  const previous = computed(() => visible.value[index.value - 1] ?? null)
  const next = computed(() => visible.value[index.value + 1] ?? null)

  // Each page opens at the top.
  watch(selectedId, () => {
    document.querySelector('.guide-content')?.scrollTo({ top: 0 })
  })
</script>

<style scoped>
.page-root {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}

.guide-card {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}

.guide-layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.guide-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
}

.guide-page {
  max-width: 720px;
}

.guide-steps li::marker {
  color: rgb(var(--v-theme-primary));
  font-weight: 700;
}

@media (max-width: 960px) {
  .guide-layout {
    flex-direction: column;
  }
}
</style>
