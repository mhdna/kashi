<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <div class="d-flex align-center ga-3 px-3 pb-1">
      <YearStepper v-model="year" class="flex-shrink-0" />
      <!-- One branch at a time, or every branch. Phones get a dropdown, since
           a row of chips wraps into several lines there. -->
      <v-select
        v-if="xs"
        class="flex-grow-1"
        clearable
        density="compact"
        hide-details
        item-title="name"
        item-value="id"
        :items="reportBranches"
        :model-value="selected[0] ?? null"
        persistent-placeholder
        placeholder="All branches"
        style="min-width: 0"
        variant="outlined"
        @update:model-value="id => (selected = id ? [id] : [])"
      />
      <v-chip-group
        v-else
        v-model="branchId"
        class="flex-grow-1"
        column
        mandatory
      >
        <!-- A picked chip is filled in the primary colour; the rest stay
             outlined. -->
        <v-chip
          :color="isPicked(ALL) ? 'primary' : undefined"
          filter
          :value="ALL"
          :variant="isPicked(ALL) ? 'flat' : 'outlined'"
        >
          All branches
        </v-chip>
        <v-chip
          v-for="branch in reportBranches"
          :key="branch.id"
          :color="isPicked(branch.id) ? 'primary' : undefined"
          filter
          :value="branch.id"
          :variant="isPicked(branch.id) ? 'flat' : 'outlined'"
        >
          {{ branch.name }}
        </v-chip>
      </v-chip-group>
    </div>
    <HeatmapTable v-model:year="year" :branch-ids="selected" :year-nav="false" />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { useDisplay } from 'vuetify'
  import YearStepper from '@/components/Buttons/YearStepper.vue'
  import HeatmapTable from '@/components/Tables/HeatmapTable.vue'
  import { useBranches } from '@/composables/useBranches'

  // Which branch the grid shows (or every branch), picked as chips. It lives
  // in the URL (?branch_ids=1), so a refresh, a bookmark or coming back from a
  // day keeps it; an old ?branch_id= link still works, and a link from when
  // several could be picked opens on its first. This page keeps its own
  // selection rather than the Dashboard's single-branch filter.
  const ALL = 'all'
  const { branches, fetchBranches } = useBranches()
  // Branches switched out of reports (Branches > "In reports") have no sales
  // in the grid, so they aren't offered.
  const reportBranches = computed(() => branches.value.filter(b => !b.hidden_from_reports))
  const { xs } = useDisplay()
  const route = useRoute()
  const router = useRouter()

  function idsFromUrl (): number[] {
    const raw = String(route.query.branch_ids ?? route.query.branch_id ?? '')
    return raw.split(',').map(Number).filter(id => id > 0).slice(0, 1)
  }

  const selected = ref<number[]>(idsFromUrl())
  // Picked here, next to the branches, rather than in the grid's corner, so
  // the day column can stay narrow.
  const year = ref(new Date().getFullYear())

  // Exactly one chip is picked: "All branches" or a single branch. Clicking
  // another switches to it.
  const branchId = computed<number | string>({
    get: () => selected.value[0] ?? ALL,
    set: value => {
      selected.value = typeof value === 'number' ? [value] : []
    },
  })

  const isPicked = (value: number | string) => branchId.value === value

  onMounted(() => fetchBranches())

  watch(selected, ids => {
    const query = { ...route.query }
    delete query.branch_id
    if (ids.length > 0) query.branch_ids = ids.join(',')
    else delete query.branch_ids
    router.replace({ query })
  })
</script>
