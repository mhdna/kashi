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
      <div class="year-total flex-shrink-0">
        <span class="year-total__label">{{ year }} total</span>
        <span class="year-total__value">{{ yearTotal === null ? '…' : money(yearTotal) }}</span>
      </div>
    </div>
    <!-- A branch with several tills can be narrowed to some of them; none
         picked means all of its tills. -->
    <div
      v-for="branch in branchesWithRegisters"
      :key="branch.id"
      class="d-flex align-center ga-3 px-3 pb-1"
    >
      <span class="text-body-2 text-medium-emphasis register-label">{{ branch.name }}</span>
      <v-chip-group
        class="branch-chips flex-grow-1"
        :model-value="registerIdsOf(branch.id)"
        multiple
        :show-arrows="false"
        @update:model-value="ids => setRegisterIds(branch.id, ids)"
      >
        <v-chip
          v-for="register in registersOf(branch.id)"
          :key="register.id"
          :color="registerIdsOf(branch.id).includes(register.id) ? 'primary' : undefined"
          filter
          size="small"
          :value="register.id"
          :variant="registerIdsOf(branch.id).includes(register.id) ? 'flat' : 'outlined'"
        >
          {{ register.name || register.code }}
        </v-chip>
      </v-chip-group>
    </div>
    <HeatmapTable
      v-model:year="year"
      :branch-ids="selected"
      :register-ids="registerIds"
      :year-nav="false"
      @total="yearTotal = $event"
    />
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
  const { branches, fetchBranches, listAllRegisters } = useBranches()
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

  // Tills picked per branch live in one flat list (the API takes register ids
  // and narrows only each one's own branch). Hidden-from-reports tills have no
  // sales in the grid, so they aren't offered.
  const registers = ref<{ id: number, branch_id: number, code: string, name: string, hidden_from_reports: boolean }[]>([])
  const registerIds = ref<number[]>([])
  const registersOf = (branchId: number) => registers.value.filter(r => r.branch_id === branchId && !r.hidden_from_reports)
  const registerIdsOf = (branchId: number) => registerIds.value.filter(id => registersOf(branchId).some(r => r.id === id))
  // Only a picked branch that really has a choice to make gets a row.
  const branchesWithRegisters = computed(() =>
    reportBranches.value.filter(b => selected.value.includes(b.id) && registersOf(b.id).length > 1),
  )
  function setRegisterIds (branchId: number, ids: number[]) {
    registerIds.value = [...registerIds.value.filter(id => !registersOf(branchId).some(r => r.id === id)), ...ids]
  }

  // The year's sum, reported by the grid.
  const yearTotal = ref<number | null>(null)
  const money = (v: number) => '$' + Math.round(v).toLocaleString()

  onMounted(async () => {
    fetchBranches()
    registers.value = await listAllRegisters().catch(() => [])
  })

  watch(selected, ids => {
    const query = { ...route.query }
    delete query.branch_id
    if (ids.length > 0) query.branch_ids = ids.join(',')
    else delete query.branch_ids
    router.replace({ query })
  })
</script>

<style scoped>
.register-label {
  min-width: 110px;
}

/* The year's sum, top right: a tinted card in the primary colour. */
.year-total {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  padding: 4px 14px;
  border-radius: 10px;
  background: rgba(var(--v-theme-primary), 0.14);
  color: rgb(var(--v-theme-primary));
  line-height: 1.2;
}

.year-total__label {
  font-size: 0.7rem;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  opacity: 0.85;
}

.year-total__value {
  font-size: 1.2rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

/* Phones: one row of chips scrolled sideways, so the slide group needs to be
   allowed to shrink next to the year stepper. */
.branch-chips {
  min-width: 0;
}
</style>
