<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <div class="d-flex align-center flex-wrap ga-2 px-3 pb-1">
      <!-- Any number of branches; none picked means every branch. -->
      <v-chip-group v-model="branchIds" column multiple selected-class="text-primary">
        <v-chip filter :value="ALL" variant="outlined">
          All branches
        </v-chip>
        <v-chip
          v-for="branch in branches"
          :key="branch.id"
          filter
          :value="branch.id"
          variant="outlined"
        >
          {{ branch.name }}
        </v-chip>
      </v-chip-group>
    </div>
    <HeatmapTable :branch-ids="selected" />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import HeatmapTable from '@/components/Tables/HeatmapTable.vue'
  import { useBranches } from '@/composables/useBranches'

  // Which branches the grid sums, picked as chips. It lives in the URL
  // (?branch_ids=1,2), so a refresh, a bookmark or coming back from a day
  // keeps it; an old ?branch_id= link still works. This page keeps its own
  // selection rather than the Dashboard's single-branch filter.
  const ALL = 'all'
  const { branches, fetchBranches } = useBranches()
  const route = useRoute()
  const router = useRouter()

  function idsFromUrl (): number[] {
    const raw = String(route.query.branch_ids ?? route.query.branch_id ?? '')
    return raw.split(',').map(Number).filter(id => id > 0)
  }

  const selected = ref<number[]>(idsFromUrl())

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

  onMounted(() => fetchBranches())

  watch(selected, ids => {
    const query = { ...route.query }
    delete query.branch_id
    if (ids.length > 0) query.branch_ids = ids.join(',')
    else delete query.branch_ids
    router.replace({ query })
  })
</script>
