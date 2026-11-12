<template>
  <div
    v-if="syncingBranchIds.length > 0"
    class="sync-pill"
    :class="{ 'sync-pill--icon': mobile }"
    :title="`Syncing ${syncingBranchNames}`"
    role="status"
  >
    <v-progress-circular color="info" indeterminate size="16" width="2" />
    <span v-if="!mobile" class="sync-pill__text">
      Syncing <strong>{{ syncingBranchNames }}</strong>
    </span>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { useDisplay } from 'vuetify'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useBranches } from '@/composables/useBranches'

  const { branches, fetchBranches, fetchSyncingBranches } = useBranches()
  const { ensureConnected, onMessage } = useAdminSocket()
  const { mobile } = useDisplay()

  const syncingBranchIds = ref([])
  let unsubscribe = null

  function branchName (branchId) {
    return branches.value.find(b => b.id === branchId)?.name ?? `Branch #${branchId}`
  }

  const syncingBranchNames = computed(() => syncingBranchIds.value.map(branchName).join(', '))

  onMounted(async () => {
    await fetchBranches()
    try {
      syncingBranchIds.value = await fetchSyncingBranches()
    } catch {
      // Informational only -- the WS push still keeps this current from
      // here on even if the initial fetch failed.
    }
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (message.type === 'branch_sync_status_changed') {
        syncingBranchIds.value = message.branch_ids ?? []
      }
    })
  })

  onUnmounted(() => {
    unsubscribe?.()
  })
</script>

<style scoped>
  /* A soft pill with a ring spinner, like the till's sync light in kashi-pos. */
  .sync-pill {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    height: 32px;
    padding: 0 14px 0 10px;
    border-radius: 999px;
    background: rgba(var(--v-theme-info), 0.14);
    border: 1px solid rgba(var(--v-theme-info), 0.28);
    font-size: 0.8125rem;
    white-space: nowrap;
  }

  .sync-pill--icon {
    width: 34px;
    height: 34px;
    padding: 0;
    justify-content: center;
  }

  .sync-pill__text strong {
    font-weight: 600;
  }
</style>
