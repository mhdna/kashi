<!--
  The receipt as the till prints it, drawn by kashi with the till's own code
  (/receipt_designer/preview), with every part clickable: a click picks the
  part for the panel beside it.
-->
<template>
  <div class="rlive" :style="{ width: `${width}px` }">
    <div v-if="image" class="rlive-paper">
      <img
        alt="The receipt as it prints"
        class="rlive-img"
        data-test="receipt-image"
        :src="image"
      >
      <button
        v-for="m in marks"
        :key="m.group"
        :aria-label="`Change ${groupLabel(m.group)}`"
        class="rlive-part"
        :class="{ 'rlive-part--picked': m.group === picked }"
        :data-test="`pick-${m.group}`"
        :disabled="!editable"
        :style="markStyle(m)"
        type="button"
        @click="emit('pick', m.group)"
      >
        <span class="rlive-label">{{ groupLabel(m.group) }}</span>
      </button>
    </div>
    <div v-else class="rlive-empty">Drawing the receipt</div>
    <v-progress-linear v-if="loading" class="rlive-loading" color="primary" indeterminate />
  </div>
</template>

<script setup lang="ts">
  const props = defineProps({
    image: { type: String, default: '' },
    height: { type: Number, default: 0 },
    marks: { type: Array, default: () => [] },
    groups: { type: Array, default: () => [] },
    picked: { type: String, default: '' },
    loading: { type: Boolean, default: false },
    editable: { type: Boolean, default: true },
    width: { type: Number, default: 360 },
  })
  const emit = defineEmits(['pick'])

  function groupLabel (key) {
    return props.groups.find(g => g.key === key)?.label ?? key
  }

  // Parts sit where the till drew them, as a share of the picture's height.
  function markStyle (m) {
    const h = props.height || 1
    return { top: `${(m.top / h) * 100}%`, height: `${((m.bottom - m.top) / h) * 100}%` }
  }
</script>

<style scoped>
.rlive {
  position: relative;
  max-width: 100%;
}
/* The receipt is white paper in either app theme. */
.rlive-paper {
  position: relative;
  background: #fff;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25);
}
.rlive-img {
  display: block;
  width: 100%;
  height: auto;
}
.rlive-part {
  position: absolute;
  left: 0;
  right: 0;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}
.rlive-part:disabled {
  cursor: default;
}
.rlive-part:not(:disabled):hover,
.rlive-part--picked {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: -2px;
}
.rlive-label {
  position: absolute;
  top: 2px;
  right: 4px;
  display: none;
  padding: 0 4px;
  font-size: 11px;
  color: rgb(var(--v-theme-primary));
  background: #fff;
}
.rlive-part:not(:disabled):hover .rlive-label,
.rlive-part--picked .rlive-label {
  display: block;
}
.rlive-empty {
  padding: 48px 16px;
  text-align: center;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.rlive-loading {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
}
</style>
