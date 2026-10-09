<template>
  <div class="d-flex flex-column flex-grow-1" style="min-height: 0">
    <div class="d-flex align-center flex-wrap ga-2 px-3 pb-1">
      <BranchMenu />
    </div>
    <HeatmapTable />
  </div>
</template>

<script setup lang="ts">
  import { onMounted, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import BranchMenu from '@/components/Buttons/BranchMenu.vue'
  import HeatmapTable from '@/components/Tables/HeatmapTable.vue'
  import { useIncomeFilters } from '@/pages/dashboard/composables/useIncomeFilters'

  // The branch filter is the same shared selection the Dashboard uses
  // (HeatmapTable already reads it and carries it to the day page). It also
  // lives in the URL (?branch_id=), so a refresh, a bookmark or coming back
  // from a day keeps the branch.
  const { selectedBranchId } = useIncomeFilters()
  const route = useRoute()
  const router = useRouter()

  onMounted(() => {
    const fromUrl = Number(route.query.branch_id)
    if (fromUrl > 0) selectedBranchId.value = fromUrl
  })

  watch(selectedBranchId, id => {
    const query = { ...route.query }
    if (id) query.branch_id = String(id)
    else delete query.branch_id
    router.replace({ query })
  })
</script>
