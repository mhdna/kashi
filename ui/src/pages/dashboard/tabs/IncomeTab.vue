<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <div class="d-flex align-center ga-3 px-3 pb-1">
      <YearStepper v-model="year" class="flex-shrink-0" />
      <!-- Any number of branches; none picked means every branch. Phones get a
           dropdown, since a row of chips wraps into several lines there. -->
      <v-select
        v-if="xs"
        class="flex-grow-1"
        clearable
        density="compact"
        hide-details
        item-title="name"
        item-value="id"
        :items="reportBranches"
        :model-value="selected"
        multiple
        persistent-placeholder
        placeholder="All branches"
        style="min-width: 0"
        variant="outlined"
        @update:model-value="ids => (selected = ids ?? [])"
      >
        <template #selection="{ item, index }">
          <span v-if="index === 0" class="text-truncate">{{ item.title }}</span>
          <span v-else-if="index === 1" class="text-medium-emphasis ms-1">+{{ selected.length - 1 }}</span>
        </template>
      </v-select>
      <v-chip-group
        v-else
        v-model="branchIds"
        class="flex-grow-1"
        column
        multiple
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
