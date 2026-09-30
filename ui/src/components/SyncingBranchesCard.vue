<template>
  <v-card
    v-if="syncingBranchIds.length > 0"
    class="d-flex align-center"
    :class="mobile ? 'icon-pill' : 'py-1 px-2'"
    rounded="xl"
    variant="tonal"
  >
    <v-icon class="rotate" color="info" icon="mdi-cloud-download" size="18" />
    <span v-if="!mobile" class="ms-1 text-body-2">
      Syncing {{ syncingBranchNames }}
    </span>
  </v-card>
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
  .icon-pill {
    width: 34px;
    height: 34px;
    justify-content: center;
  }

  /* A static download icon reads as "idle"; spin it so a persistently
     syncing branch stays visually distinct from any other cloud icon. */
  .rotate {
    animation: spin 1.6s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
