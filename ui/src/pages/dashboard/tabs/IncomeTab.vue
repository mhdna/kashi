<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <div class="d-flex align-center ga-3 px-3 pb-1">
      <YearStepper v-model="year" class="flex-shrink-0" />
      <!-- Any number of branches; none picked means every branch. Phones get
           the same chips on one line that scrolls sideways, rather than
           wrapping into several. -->
      <v-chip-group
        v-model="branchIds"
        class="branch-chips flex-grow-1"
        :column="!xs"
        multiple
        :show-arrows="false"
      >
        <!-- A picked chip is filled in the primary colour; the rest stay
             outlined. -->
        <v-chip
          :color="isPicked(ALL) ? 'primary' : undefined"
          filter
          :size="xs ? 'small' : undefined"
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
          :size="xs ? 'small' : undefined"
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

  // Which branches the grid sums, picked as chips. It lives in the URL
  // (?branch_ids=1,2), so a refresh, a bookmark or coming back from a day
  // keeps it; an old ?branch_id= link still works. This page keeps its own
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
    return raw.split(',').map(Number).filter(id => id > 0)
  }

  const selected = ref<number[]>(idsFromUrl())
  // Picked here, next to the branches, rather than in the grid's corner, so
  // the day column can stay narrow.
  const year = ref(new Date().getFullYear())

  // The chip group also carries the "All branches" chip, shown as picked
  // exactly when nothing else is. Picking it clears the branches; picking a
  // branch while it's on just starts a selection.
  const branchIds = computed<(number | string)[]>({
    get: () => (selected.value.length === 0 ? [ALL] : selected.value),
    set: value => {
      const pickedAll = value.includes(ALL) && selected.value.length > 0
      selected.value = pickedAll ? [] : value.filter((v): v is number => typeof v === 'number')
    },
  })

  const isPicked = (value: number | string) => branchIds.value.includes(value)

  onMounted(() => fetchBranches())

  watch(selected, ids => {
    const query = { ...route.query }
    delete query.branch_id
    if (ids.length > 0) query.branch_ids = ids.join(',')
    else delete query.branch_ids
    router.replace({ query })
  })
</script>

<style scoped>
/* Phones: one row of chips scrolled sideways, so the slide group needs to be
   allowed to shrink next to the year stepper. */
.branch-chips {
  min-width: 0;
}
</style>
