<template>
  <!-- Stand-in for a page while it (or the settings it needs) loads: the rough
       shape most pages have -- a title row, a strip of figure cards, a table --
       so the page fades in where the shapes were instead of after a spinner. -->
  <v-container class="page-skeleton" fluid>
    <div class="d-flex align-center ga-3 mb-5">
      <v-skeleton-loader class="sk-title" color="transparent" type="heading" />
      <v-spacer />
      <v-skeleton-loader v-if="!mobile" class="sk-pill" color="transparent" type="chip" />
      <v-skeleton-loader class="sk-pill" color="transparent" type="chip" />
    </div>

    <v-row class="mb-3" dense>
      <v-col v-for="n in cards" :key="n" cols="6" md="3">
        <v-skeleton-loader class="sk-card" type="list-item-two-line" />
      </v-col>
    </v-row>

    <v-skeleton-loader class="sk-table" type="table-heading, table-thead, table-tbody" />
  </v-container>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useDisplay } from 'vuetify'

  const { mobile } = useDisplay()
  const cards = computed(() => (mobile.value ? 2 : 4))
</script>

<style scoped>
.page-skeleton {
  animation: sk-fade-in 0.25s ease-out;
}

.sk-title {
  width: min(320px, 55%);
}

.sk-title :deep(.v-skeleton-loader__heading) {
  margin: 0;
  height: 28px;
}

.sk-pill :deep(.v-skeleton-loader__chip) {
  margin: 0;
  width: 96px;
  max-width: none;
  height: 34px;
  border-radius: 24px;
}

.sk-card,
.sk-table {
  border-radius: 12px;
  overflow: hidden;
}

/* Don't flash anything for loads that finish almost at once. */
@keyframes sk-fade-in {
  0%,
  40% {
    opacity: 0;
  }

  100% {
    opacity: 1;
  }
}
</style>
