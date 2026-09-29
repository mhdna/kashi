<!--
  The one loyalty-tier chip used everywhere (tier ladders, client rows,
  detail panes). No background, no outline -- just a color dot for the tier
  plus the label. Pass a resolved `tier` object, or a `spent` amount in
  cents and it resolves the tier itself.
-->
<template>
  <v-chip v-if="resolved" class="tier-chip" :size="size" variant="text">
    <span class="tier-chip__dot" :style="{ backgroundColor: resolved.color }" />
    {{ resolved.label }}<template v-if="showThreshold"> · ${{ resolved.minAmount.toLocaleString() }}+</template>
  </v-chip>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useClientTiers } from '@/composables/useClientTiers'

  const props = defineProps({
    // A resolved tier object ({ label, color, minAmount }). Takes precedence.
    tier: { type: Object, default: null },
    // Lifetime spend in cents -- used to resolve the tier when `tier` is absent.
    spent: { type: Number, default: null },
    // 'x-small' | 'small' | 'default' | 'large' ...
    size: { type: String, default: 'small' },
    // Append the tier's minimum ("Gold · $800+") -- for tier-ladder displays.
    showThreshold: { type: Boolean, default: false },
  })

  const { tierFor } = useClientTiers()

  const resolved = computed(
    () => props.tier ?? (props.spent == null ? null : tierFor(props.spent)),
  )
</script>

<style scoped>
.tier-chip {
  font-size: 0.9375rem;
  font-weight: 500;
}

.tier-chip__dot {
  display: inline-block;
  width: 12px;
  height: 12px;
  margin-right: 8px;
  border-radius: 50%;
  vertical-align: middle;
}
</style>
