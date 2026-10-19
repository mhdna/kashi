<template>
  <div class="page-root">
    <v-card flat>
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

      <div class="guide d-flex ga-6 pa-5">
        <nav class="guide-toc flex-shrink-0">
          <div v-for="group in visibleGroups" :key="group.name" class="mb-3">
            <div class="text-overline text-medium-emphasis">{{ group.name }}</div>
            <a
              v-for="section in group.sections"
              :key="section.id"
              class="d-block text-body-2 py-1"
              :href="`#${section.id}`"
            >{{ section.title }}</a>
          </div>
        </nav>

        <div class="guide-body flex-grow-1">
          <div v-if="visibleGroups.length === 0" class="text-medium-emphasis">
            Nothing in the guide matches "{{ search }}".
          </div>
          <template v-for="group in visibleGroups" :key="group.name">
            <div class="text-overline text-medium-emphasis mb-2">{{ group.name }}</div>
            <section v-for="section in group.sections" :id="section.id" :key="section.id" class="mb-7">
              <h2 class="text-h6 mb-2">{{ section.title }}</h2>
              <template v-for="(block, i) in section.blocks" :key="i">
                <p v-if="block.kind === 'p'" class="text-body-2 mb-2" v-html="block.text" />
                <ul v-else-if="block.kind === 'list'" class="text-body-2 mb-2 ps-5">
                  <li v-for="(item, j) in block.items" :key="j" class="mb-1" v-html="item" />
                </ul>
                <v-alert
                  v-else-if="block.kind === 'tip'"
                  class="mb-2"
                  density="compact"
                  type="info"
                  variant="tonal"
                >
                  <span v-html="block.text" />
                </v-alert>
              </template>
            </section>
          </template>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup>
  import { computed, ref } from 'vue'
  import { guideGroups, guideSections } from '@/data/userGuide'

  const search = ref('')

  // Plain text of an entry, for searching: tags stripped.
  function textOf (section) {
    const parts = [section.title]
    for (const block of section.blocks) {
      parts.push(block.text ?? '', ...(block.items ?? []))
    }
    return parts.join(' ').replace(/<[^>]+>/g, ' ').toLowerCase()
  }

  const visibleGroups = computed(() => {
    const q = (search.value ?? '').trim().toLowerCase()
    const sections = q ? guideSections.filter(s => textOf(s).includes(q)) : guideSections
    return guideGroups
      .map(name => ({ name, sections: sections.filter(s => s.group === name) }))
      .filter(g => g.sections.length > 0)
  })
</script>

<style scoped>
.guide-toc {
  position: sticky;
  top: 16px;
  align-self: flex-start;
  min-width: 200px;
  max-height: calc(100vh - 120px);
  overflow-y: auto;
}

.guide-toc a {
  color: rgb(var(--v-theme-primary));
  text-decoration: none;
}

.guide-body {
  max-width: 760px;
  min-width: 0;
}

@media (max-width: 960px) {
  .guide {
    flex-direction: column;
  }

  .guide-toc {
    position: static;
    max-height: none;
  }
}
</style>
